package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

// computeFingerprint returns a deterministic, content-based SHA-256 hex digest
// for the given files.
//
// The digest is order-independent (paths are sorted first) and changes whenever
// any file's path or content changes — the same property Bitrise's
// `{{ checksum ... }}` template relies on to build cache keys. We bind each
// file's path to its own content hash so that renaming a file, or swapping which
// file holds which content, also changes the fingerprint.
func computeFingerprint(paths []string) (string, error) {
	cleaned := make([]string, 0, len(paths))
	for _, p := range paths {
		if p = strings.TrimSpace(p); p != "" {
			cleaned = append(cleaned, p)
		}
	}
	if len(cleaned) == 0 {
		return "", fmt.Errorf("no file paths provided")
	}
	sort.Strings(cleaned)

	outer := sha256.New()
	for _, p := range cleaned {
		fileHash, err := hashFileContent(p)
		if err != nil {
			return "", err
		}
		if _, err := io.WriteString(outer, p+":"+fileHash+"\n"); err != nil {
			return "", err
		}
	}
	return hex.EncodeToString(outer.Sum(nil)), nil
}

func hashFileContent(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()

	info, err := f.Stat()
	if err != nil {
		return "", fmt.Errorf("stat %s: %w", path, err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("%s is a directory, expected a file", path)
	}

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("read %s: %w", path, err)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
