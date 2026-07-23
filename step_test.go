package main

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", p, err)
	}
	return p
}

func TestComputeFingerprintDeterministicAndOrderIndependent(t *testing.T) {
	dir := t.TempDir()
	a := writeFile(t, dir, "package.json", `{"name":"demo"}`)
	b := writeFile(t, dir, "package-lock.json", `{"lockfileVersion":3}`)

	h1, err := computeFingerprint([]string{a, b})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	h2, err := computeFingerprint([]string{b, a})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if h1 != h2 {
		t.Fatalf("fingerprint not order-independent: %s != %s", h1, h2)
	}
	if len(h1) != 64 {
		t.Fatalf("expected 64-char sha256 hex, got %d chars", len(h1))
	}
}

func TestComputeFingerprintChangesWithContent(t *testing.T) {
	dir := t.TempDir()
	a := writeFile(t, dir, "package.json", `{"name":"demo"}`)

	before, err := computeFingerprint([]string{a})
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, dir, "package.json", `{"name":"demo","version":"2"}`)
	after, err := computeFingerprint([]string{a})
	if err != nil {
		t.Fatal(err)
	}
	if before == after {
		t.Fatal("fingerprint did not change when file content changed")
	}
}

func TestComputeFingerprintErrors(t *testing.T) {
	if _, err := computeFingerprint([]string{"  ", ""}); err == nil {
		t.Fatal("expected error for empty path list")
	}
	if _, err := computeFingerprint([]string{filepath.Join(t.TempDir(), "nope.json")}); err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestWithKeyPrefix(t *testing.T) {
	if got, want := withKeyPrefix("", "abc123"), "abc123"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
	if got, want := withKeyPrefix("build-fingerprint", "abc123"), "build-fingerprint-abc123"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestParsePaths(t *testing.T) {
	in := "package.json\n  package-lock.json  \n\n# a comment\nios/Podfile.lock\n"
	got := parsePaths(in)
	want := []string{"package.json", "package-lock.json", "ios/Podfile.lock"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("index %d: got %q want %q", i, got[i], want[i])
		}
	}
}
