package step

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/bmatcuk/doublestar/v4"
)

// parseList splits a newline-separated input into a clean list. Blank lines and
// lines starting with '#' (comments) are ignored; paths may contain spaces.
func parseList(raw string) []string {
	var out []string
	for line := range strings.SplitSeq(raw, "\n") {
		line = strings.TrimSpace(strings.TrimRight(line, "\r"))
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		out = append(out, line)
	}
	return out
}

// withKeyPrefix prepends prefix (joined with '-') to the fingerprint. An empty
// prefix leaves the fingerprint unchanged.
func withKeyPrefix(prefix, fingerprint string) string {
	if prefix == "" {
		return fingerprint
	}
	return prefix + "-" + fingerprint
}

// fingerprintPaths returns a deterministic SHA-256 hex digest over the files in
// fsys selected by paths, excluding anything matched by ignore. Every entry is
// treated as a doublestar pattern — a plain path is just a trivial pattern —
// and directory matches are walked recursively. Each file contributes its
// fsys-relative path (so the digest is independent of where the project is
// checked out) bound to its own content hash.
//
// A malformed pattern or an unexpected I/O error is fatal: we never return a
// digest computed over a partially-read input set (that could hide a real
// change and reuse a stale native build).
func fingerprintPaths(fsys fs.FS, paths, ignore []string, logger log.Logger) (string, error) {
	ignore = normalizePatterns(ignore) // normalize once, not per walked entry

	seen := map[string]bool{}
	var rels []string
	for _, p := range paths {
		matches, err := doublestar.Glob(fsys, path.Clean(filepath.ToSlash(p)))
		if err != nil {
			return "", fmt.Errorf("invalid path pattern %q: %w", p, err)
		}
		if len(matches) == 0 {
			// Not fatal: the default paths intentionally list optional inputs
			// (yarn.lock, pnpm-lock.yaml, app.config.js, …) that may be absent.
			logger.Warnf("No match for %q", p)
			continue
		}
		for _, match := range matches {
			files, err := collect(fsys, match, ignore)
			if err != nil {
				return "", err
			}
			for _, rel := range files {
				if !seen[rel] {
					seen[rel] = true
					rels = append(rels, rel)
				}
			}
		}
	}

	if len(rels) == 0 {
		return "", nil
	}
	sort.Strings(rels)

	outer := sha256.New()
	for _, rel := range rels {
		fileHash, err := hashFileContent(fsys, rel, logger)
		if err != nil {
			return "", err
		}
		logger.Debugf("  %s  %s", fileHash[:12], rel)
		// sha256's Write never returns an error, but check defensively.
		if _, err := io.WriteString(outer, rel+"\x00"+fileHash+"\n"); err != nil {
			return "", err
		}
	}
	return hex.EncodeToString(outer.Sum(nil)), nil
}

// collect returns the fsys-relative paths of the regular files at root, walking
// it recursively if it is a directory. Ignored entries (and their subtrees) and
// symlinks are skipped; an unexpected I/O error aborts so we never fingerprint a
// partially-read input set.
func collect(fsys fs.FS, root string, ignore []string) ([]string, error) {
	info, err := fs.Stat(fsys, root)
	if err != nil {
		return nil, fmt.Errorf("stat %s: %w", root, err)
	}

	if !info.IsDir() {
		// Skip ignored files and non-regular entries (sockets, devices, …).
		if isIgnored(root, ignore) || !info.Mode().IsRegular() {
			return nil, nil
		}
		return []string{root}, nil
	}

	var out []string
	err = fs.WalkDir(fsys, root, func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return fmt.Errorf("access %s: %w", p, walkErr)
		}
		if d.IsDir() {
			if p != root && isIgnored(p, ignore) {
				return fs.SkipDir
			}
			return nil
		}
		// Skip symlinks and other irregular entries: they are not deterministic
		// build inputs, and following symlinks risks walk loops.
		if d.Type()&fs.ModeSymlink != 0 || !d.Type().IsRegular() {
			return nil
		}
		if !isIgnored(p, ignore) {
			out = append(out, p)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// normalizePatterns cleans each pattern to a slash path once, so matching does
// not repeat the work for every walked entry.
func normalizePatterns(patterns []string) []string {
	out := make([]string, len(patterns))
	for i, p := range patterns {
		out[i] = path.Clean(filepath.ToSlash(p))
	}
	return out
}

// isIgnored reports whether the fsys-relative path matches any (already
// normalized) ignore pattern, as a doublestar glob or a directory prefix.
func isIgnored(rel string, ignore []string) bool {
	for _, pat := range ignore {
		if ok, _ := doublestar.Match(pat, rel); ok {
			return true
		}
		if rel == pat || strings.HasPrefix(rel, pat+"/") {
			return true
		}
	}
	return false
}

func hashFileContent(fsys fs.FS, name string, logger log.Logger) (string, error) {
	f, err := fsys.Open(name)
	if err != nil {
		return "", fmt.Errorf("open %s: %w", name, err)
	}
	defer func() {
		// A Close error on a read-only file carries no data-loss risk and must
		// not fail the fingerprint, but log it rather than swallow it silently.
		if cerr := f.Close(); cerr != nil {
			logger.Warnf("Close %s: %s", name, cerr)
		}
	}()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("read %s: %w", name, err)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
