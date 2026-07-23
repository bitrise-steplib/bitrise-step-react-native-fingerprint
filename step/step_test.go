package step

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bitrise-io/go-steputils/v2/export"
	"github.com/bitrise-io/go-steputils/v2/stepconf"
	"github.com/bitrise-io/go-utils/v2/env"
	"github.com/bitrise-io/go-utils/v2/log"
)

func newStep() FingerprintStep {
	return NewFingerprintStep(log.NewLogger(), nil, export.Exporter{})
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// runPaths fingerprints a project dir via the paths/ignore mechanism.
func runPaths(t *testing.T, dir string, paths, ignore []string, prefix string) (string, error) {
	t.Helper()
	res, err := newStep().Run(Config{ProjectDir: dir, Paths: paths, IgnorePaths: ignore, KeyPrefix: prefix})
	return res.BundleHashString, err
}

func mustRunPaths(t *testing.T, dir string, paths, ignore []string, prefix string) string {
	t.Helper()
	h, err := runPaths(t, dir, paths, ignore, prefix)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	return h
}

func TestFingerprint_DeterministicWithPrefix(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "package.json"), `{"name":"demo"}`)
	writeFile(t, filepath.Join(dir, "ios", "Podfile.lock"), "PODS: []")
	writeFile(t, filepath.Join(dir, "android", "app", "build.gradle"), "dependencies {}")

	// Mix of a file and folders; trailing slash on folders is optional.
	paths := []string{"package.json", "ios/", "android/"}
	h1 := mustRunPaths(t, dir, paths, nil, "bundle")
	h2 := mustRunPaths(t, dir, paths, nil, "bundle")

	if h1 != h2 {
		t.Fatalf("not deterministic: %s != %s", h1, h2)
	}
	if !strings.HasPrefix(h1, "bundle-") {
		t.Fatalf("prefix not applied: %q", h1)
	}
	if got := strings.TrimPrefix(h1, "bundle-"); len(got) != 64 {
		t.Fatalf("expected 64-char sha256, got %d (%q)", len(got), h1)
	}
}

func TestFingerprint_FolderWithOrWithoutTrailingSlashMatch(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "android", "app", "build.gradle"), "x")

	withSlash := mustRunPaths(t, dir, []string{"android/"}, nil, "")
	noSlash := mustRunPaths(t, dir, []string{"android"}, nil, "")
	if withSlash != noSlash {
		t.Fatalf("trailing slash changed the result: %s != %s", withSlash, noSlash)
	}
}

func TestFingerprint_DetectsNestedChangeAndNewFile(t *testing.T) {
	dir := t.TempDir()
	gradle := filepath.Join(dir, "android", "app", "build.gradle")
	writeFile(t, gradle, "dependencies { v1 }")

	before := mustRunPaths(t, dir, []string{"android"}, nil, "")
	writeFile(t, gradle, "dependencies { v2 }")
	if after := mustRunPaths(t, dir, []string{"android"}, nil, ""); after == before {
		t.Fatal("did not detect a nested file content change")
	}

	afterEdit := mustRunPaths(t, dir, []string{"android"}, nil, "")
	writeFile(t, filepath.Join(dir, "android", "feature", "build.gradle"), "// new module")
	if after := mustRunPaths(t, dir, []string{"android"}, nil, ""); after == afterEdit {
		t.Fatal("did not detect a newly added nested file")
	}
}

func TestFingerprint_IgnoreExcludesMatches(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "android", "app", "build.gradle"), "real input")
	writeFile(t, filepath.Join(dir, "android", "build", "output.txt"), "generated v1")

	ignore := []string{"android/build"}
	before := mustRunPaths(t, dir, []string{"android"}, ignore, "")

	// Changing an ignored file must NOT move the fingerprint.
	writeFile(t, filepath.Join(dir, "android", "build", "output.txt"), "generated v2")
	if after := mustRunPaths(t, dir, []string{"android"}, ignore, ""); after != before {
		t.Fatal("ignored file change moved the fingerprint")
	}

	// Changing a real input still must.
	writeFile(t, filepath.Join(dir, "android", "app", "build.gradle"), "real input changed")
	if after := mustRunPaths(t, dir, []string{"android"}, ignore, ""); after == before {
		t.Fatal("real input change not detected")
	}
}

func TestFingerprint_GlobIgnorePattern(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "android", "app", "build.gradle"), "keep")
	writeFile(t, filepath.Join(dir, "android", "app", "debug.log"), "noise v1")

	ignore := []string{"**/*.log"}
	before := mustRunPaths(t, dir, []string{"android"}, ignore, "")
	writeFile(t, filepath.Join(dir, "android", "app", "debug.log"), "noise v2")
	if after := mustRunPaths(t, dir, []string{"android"}, ignore, ""); after != before {
		t.Fatal("glob-ignored file change moved the fingerprint")
	}
}

func TestFingerprint_MissingPathSkipped(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "package.json"), `{"name":"demo"}`)

	h, err := runPaths(t, dir, []string{"package.json", "does-not-exist", "ios"}, nil, "")
	if err != nil {
		t.Fatalf("missing paths should be skipped, got error: %v", err)
	}
	if len(h) != 64 {
		t.Fatalf("expected a 64-char hash from the present file, got %q", h)
	}
}

func TestFingerprint_GlobInPaths(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "android", "build.gradle"), "a")
	writeFile(t, filepath.Join(dir, "android", "app", "build.gradle"), "b")

	before := mustRunPaths(t, dir, []string{"android/**/*.gradle"}, nil, "")
	if len(before) != 64 {
		t.Fatalf("glob path produced no hash: %q", before)
	}
	writeFile(t, filepath.Join(dir, "android", "feature", "build.gradle"), "c")
	if after := mustRunPaths(t, dir, []string{"android/**/*.gradle"}, nil, ""); after == before {
		t.Fatal("glob path did not pick up a new match")
	}
}

func TestRun_EmptyWhenEverythingIgnoredFails(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "android", "app", "build.gradle"), "x")
	if _, err := runPaths(t, dir, []string{"android"}, []string{"android"}, ""); err == nil {
		t.Fatal("expected an error when all paths are ignored")
	}
}

func TestProcessConfig_ParsesListsAndDefaultsProjectDir(t *testing.T) {
	t.Setenv("project_dir", "")
	t.Setenv("paths", "package.json\n  ios/  \n\n# comment\nandroid/\n")
	t.Setenv("ignore_paths", "ios/Pods\nandroid/build")
	t.Setenv("key_prefix", "")
	t.Setenv("verbose", "false")

	parser := stepconf.NewInputParser(env.NewRepository())
	cfg, err := NewFingerprintStep(log.NewLogger(), parser, export.Exporter{}).ProcessConfig()
	if err != nil {
		t.Fatalf("ProcessConfig: %v", err)
	}
	if cfg.ProjectDir != "." {
		t.Fatalf("project_dir default: got %q", cfg.ProjectDir)
	}
	if strings.Join(cfg.Paths, ",") != "package.json,ios/,android/" {
		t.Fatalf("paths parse: got %v", cfg.Paths)
	}
	if strings.Join(cfg.IgnorePaths, ",") != "ios/Pods,android/build" {
		t.Fatalf("ignore parse: got %v", cfg.IgnorePaths)
	}
}
