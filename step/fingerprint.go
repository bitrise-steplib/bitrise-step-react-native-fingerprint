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
func fingerprintPaths(fsys fs.FS, paths, ignore []string, logger log.Logger) (string, error) {
	ignore = normalizePatterns(ignore) // normalize once, not per walked entry

	seen := map[string]bool{}
	var rels []string
	for _, p := range paths {
		pattern := path.Clean(filepath.ToSlash(p))
		matches, err := doublestar.Glob(fsys, pattern)
		if err != nil {
			logger.Warnf("Invalid path pattern %q: %s", p, err)
			continue
		}
		if len(matches) == 0 {
			logger.Warnf("No match for %q", p)
			continue
		}
		for _, match := range matches {
			for _, rel := range collect(fsys, match, ignore, logger) {
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
		fileHash, err := hashFileContent(fsys, rel)
		if err != nil {
			return "", err
		}
		logger.Debugf("  %s  %s", fileHash[:12], rel)
		if _, err := io.WriteString(outer, rel+"\x00"+fileHash+"\n"); err != nil {
			return "", err
		}
	}
	return hex.EncodeToString(outer.Sum(nil)), nil
}

// collect returns the fsys-relative paths of the regular files at root, walking
// it recursively if it is a directory. Ignored entries and symlinks are skipped
// (ignored directories are not descended into).
func collect(fsys fs.FS, root string, ignore []string, logger log.Logger) []string {
	info, err := fs.Stat(fsys, root)
	if err != nil {
		logger.Warnf("Skipping %q: %s", root, err)
		return nil
	}

	if !info.IsDir() {
		if isIgnored(root, ignore) || !info.Mode().IsRegular() {
			return nil
		}
		return []string{root}
	}

	var out []string
	_ = fs.WalkDir(fsys, root, func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			logger.Warnf("Skipping %q: %s", p, walkErr)
			return nil //nolint:nilerr
		}
		if d.IsDir() {
			if p != root && isIgnored(p, ignore) {
				return fs.SkipDir
			}
			return nil
		}
		if d.Type()&fs.ModeSymlink != 0 || !d.Type().IsRegular() {
			return nil
		}
		if !isIgnored(p, ignore) {
			out = append(out, p)
		}
		return nil
	})
	return out
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

func hashFileContent(fsys fs.FS, name string) (string, error) {
	f, err := fsys.Open(name)
	if err != nil {
		return "", fmt.Errorf("open %s: %w", name, err)
	}
	defer func() { _ = f.Close() }()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("read %s: %w", name, err)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
