package step

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/bitrise-io/go-steputils/v2/cache/keytemplate"
	"github.com/bitrise-io/go-steputils/v2/export"
	"github.com/bitrise-io/go-steputils/v2/stepconf"
	"github.com/bitrise-io/go-utils/v2/env"
	"github.com/bitrise-io/go-utils/v2/log"
)

// fakeEvaluator is a stand-in KeyEvaluator for exercising Run's control flow
// without touching the filesystem.
type fakeEvaluator struct {
	out string
	err error
}

func (f fakeEvaluator) Evaluate(string) (string, error) { return f.out, f.err }

// realEvaluator wires a keytemplate.Model exactly like production does.
func realEvaluator() KeyEvaluator {
	return keytemplate.NewModel(env.NewRepository(), log.NewLogger())
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

func runKey(t *testing.T, key string) string {
	t.Helper()
	s := NewFingerprintStep(log.NewLogger(), nil, realEvaluator(), export.Exporter{})
	res, err := s.Run(Config{Key: key})
	if err != nil {
		t.Fatalf("Run(%q): %v", key, err)
	}
	return res.BundleHashString
}

func TestRun_ChecksumDeterministicWithLiteralPrefix(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "package.json"), `{"name":"demo"}`)
	writeFile(t, filepath.Join(dir, "package-lock.json"), `{"lockfileVersion":3}`)
	t.Chdir(dir)

	key := `rn-{{ checksum "package.json" "package-lock.json" }}`
	h1 := runKey(t, key)
	h2 := runKey(t, key)

	if h1 != h2 {
		t.Fatalf("not deterministic: %s != %s", h1, h2)
	}
	if got := h1[:3]; got != "rn-" {
		t.Fatalf("literal prefix not preserved, got %q", h1)
	}
	if len(h1) != len("rn-")+64 {
		t.Fatalf("expected rn- + 64-char sha256, got %q (len %d)", h1, len(h1))
	}
}

func TestRun_ChangesWhenChecksummedFileChanges(t *testing.T) {
	dir := t.TempDir()
	pkg := filepath.Join(dir, "package.json")
	writeFile(t, pkg, `{"version":"1"}`)
	t.Chdir(dir)

	before := runKey(t, `{{ checksum "package.json" }}`)
	writeFile(t, pkg, `{"version":"2"}`)
	after := runKey(t, `{{ checksum "package.json" }}`)

	if before == after {
		t.Fatal("hash did not change when a checksummed file changed")
	}
}

func TestRun_GlobMatchesNestedFilesAndDetectsNewFile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "android", "build.gradle"), "buildscript {}")
	writeFile(t, filepath.Join(dir, "android", "app", "build.gradle"), "dependencies {}")
	t.Chdir(dir)

	key := `{{ checksum "android/**/*.gradle" }}`
	before := runKey(t, key)
	if len(before) != 64 {
		t.Fatalf("expected 64-char sha256 from glob, got %q", before)
	}

	// A brand-new nested native file must move the fingerprint.
	writeFile(t, filepath.Join(dir, "android", "feature", "build.gradle"), "// new module")
	after := runKey(t, key)
	if before == after {
		t.Fatal("glob did not detect a newly added matching file")
	}
}

func TestRun_EmptyResultFails(t *testing.T) {
	s := NewFingerprintStep(log.NewLogger(), nil, fakeEvaluator{out: ""}, export.Exporter{})
	if _, err := s.Run(Config{Key: `{{ checksum "nope" }}`}); err == nil {
		t.Fatal("expected an error when the evaluated key is empty")
	}
}

func TestRun_EvaluatorErrorPropagates(t *testing.T) {
	s := NewFingerprintStep(log.NewLogger(), nil, fakeEvaluator{err: errors.New("bad template")}, export.Exporter{})
	if _, err := s.Run(Config{Key: "{{ invalid"}); err == nil {
		t.Fatal("expected the evaluator error to propagate")
	}
}

func TestProcessConfig_RequiresKey(t *testing.T) {
	parser := stepconf.NewInputParser(env.NewRepository())
	s := NewFingerprintStep(log.NewLogger(), parser, nil, export.Exporter{})

	// key unset -> required validation fails.
	t.Setenv("key", "")
	t.Setenv("verbose", "false")
	if _, err := s.ProcessConfig(); err == nil {
		t.Fatal("expected error when required 'key' is empty")
	}

	// key set -> parsed through.
	t.Setenv("key", `{{ checksum "package.json" }}`)
	cfg, err := s.ProcessConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Key != `{{ checksum "package.json" }}` {
		t.Fatalf("unexpected key: %q", cfg.Key)
	}
}
