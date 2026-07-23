package main

import (
	"fmt"
	"os"
	"strings"
)

// parsePaths splits a newline-separated input into a clean list of paths.
// Paths may contain spaces, so we only split on line breaks; blank lines and
// lines starting with '#' (comments) are ignored.
func parsePaths(raw string) []string {
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

// withKeyPrefix prepends prefix to fingerprint to form the final cache key,
// e.g. "build-fingerprint" + "abc123..." -> "build-fingerprint-abc123...". An empty
// prefix leaves the fingerprint unchanged.
func withKeyPrefix(prefix, fingerprint string) string {
	if prefix == "" {
		return fingerprint
	}
	return prefix + "-" + fingerprint
}

func log(format string, args ...any) {
	fmt.Printf(format+"\n", args...)
}

func failf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "❌ "+format+"\n", args...)
	os.Exit(1)
}
