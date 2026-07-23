package step

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
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

// fingerprintPaths returns a deterministic SHA-256 hex digest over the files
// selected by paths (relative to projectDir), excluding anything matched by
// ignore. Directory entries are walked recursively; glob/doublestar patterns
// are supported. Each file contributes its projectDir-relative path (so the
// digest is independent of where the project is checked out) bound to its own
// content hash.
func fingerprintPaths(projectDir string, paths, ignore []string, logger log.Logger) (string, error) {
	seen := map[string]bool{}
	var rels []string
	add := func(rel string) {
		rel = filepath.ToSlash(rel)
		if !seen[rel] {
			seen[rel] = true
			rels = append(rels, rel)
		}
	}

	for _, p := range paths {
		p = filepath.ToSlash(p)
		if strings.ContainsAny(p, "*?[") {
			matches, err := doublestar.Glob(os.DirFS(projectDir), p)
			if err != nil {
				logger.Warnf("Invalid pattern %q: %s", p, err)
				continue
			}
			if len(matches) == 0 {
				logger.Warnf("No match for pattern: %s", p)
			}
			for _, m := range matches {
				collect(projectDir, m, ignore, add, logger)
			}
			continue
		}
		collect(projectDir, p, ignore, add, logger)
	}

	if len(rels) == 0 {
		return "", nil
	}
	sort.Strings(rels)

	outer := sha256.New()
	for _, rel := range rels {
		fileHash, err := hashFileContent(filepath.Join(projectDir, filepath.FromSlash(rel)))
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

// collect adds rel (a projectDir-relative path) to the set. If rel is a
// directory it is walked recursively; ignored entries and symlinks are skipped.
func collect(projectDir, rel string, ignore []string, add func(string), logger log.Logger) {
	full := filepath.Join(projectDir, filepath.FromSlash(rel))
	info, err := os.Lstat(full)
	if err != nil {
		logger.Warnf("Skipping %q: %s", rel, err)
		return
	}

	if info.Mode()&os.ModeSymlink != 0 {
		logger.Debugf("Skipping symlink: %s", rel)
		return
	}

	if !info.IsDir() {
		if isIgnored(rel, ignore) {
			return
		}
		if info.Mode().IsRegular() {
			add(rel)
		}
		return
	}

	_ = filepath.WalkDir(full, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			logger.Warnf("Skipping %q: %s", path, walkErr)
			return nil //nolint:nilerr
		}
		childRel, err := filepath.Rel(projectDir, path)
		if err != nil {
			return nil
		}
		childRel = filepath.ToSlash(childRel)

		if d.IsDir() {
			if childRel != rel && isIgnored(childRel, ignore) {
				return fs.SkipDir
			}
			return nil
		}
		if d.Type()&fs.ModeSymlink != 0 || !d.Type().IsRegular() {
			return nil
		}
		if isIgnored(childRel, ignore) {
			return nil
		}
		add(childRel)
		return nil
	})
}

// isIgnored reports whether the projectDir-relative path matches any ignore
// pattern, either as a doublestar glob or as a directory prefix.
func isIgnored(rel string, ignore []string) bool {
	for _, pat := range ignore {
		pat = filepath.ToSlash(pat)
		if ok, _ := doublestar.Match(pat, rel); ok {
			return true
		}
		trimmed := strings.TrimSuffix(pat, "/")
		if rel == trimmed || strings.HasPrefix(rel, trimmed+"/") {
			return true
		}
	}
	return false
}

func hashFileContent(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("read %s: %w", path, err)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
