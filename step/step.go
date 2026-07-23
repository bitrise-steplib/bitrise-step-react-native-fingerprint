package step

import (
	"fmt"

	"github.com/bitrise-io/go-steputils/v2/export"
	"github.com/bitrise-io/go-steputils/v2/stepconf"
	"github.com/bitrise-io/go-utils/v2/log"
)

const bundleHashStringKey = "BUNDLE_HASH_STRING"

// Input maps the step inputs (see step.yml) to Go fields.
type Input struct {
	Key     string `env:"key,required"`
	Verbose bool   `env:"verbose"`
}

// Config is the processed and validated configuration derived from Input.
type Config struct {
	Key string
}

// Result holds the step's computed output.
type Result struct {
	BundleHashString string
}

// KeyEvaluator evaluates a Bitrise cache-key template — the same syntax the
// restore-cache / save-cache steps accept, e.g. `{{ checksum "package.json" }}`,
// `{{ getenv "X" }}`, and the `.OS` / `.Arch` / `.Branch` variables.
// It is satisfied by go-steputils/v2/cache/keytemplate.Model.
type KeyEvaluator interface {
	Evaluate(key string) (string, error)
}

// FingerprintStep evaluates a cache-key template and exports the result as
// BUNDLE_HASH_STRING for subsequent cache and build-gating steps.
type FingerprintStep struct {
	logger         log.Logger
	inputParser    stepconf.InputParser
	keyEvaluator   KeyEvaluator
	outputExporter export.Exporter
}

// NewFingerprintStep wires the step with its dependencies.
func NewFingerprintStep(logger log.Logger, inputParser stepconf.InputParser, keyEvaluator KeyEvaluator, outputExporter export.Exporter) FingerprintStep {
	return FingerprintStep{
		logger:         logger,
		inputParser:    inputParser,
		keyEvaluator:   keyEvaluator,
		outputExporter: outputExporter,
	}
}

// ProcessConfig parses and validates the step inputs.
func (s FingerprintStep) ProcessConfig() (Config, error) {
	var input Input
	if err := s.inputParser.Parse(&input); err != nil {
		return Config{}, fmt.Errorf("parse inputs: %w", err)
	}
	stepconf.Print(input)
	s.logger.Println()

	// keytemplate logs the files that feed each checksum at debug level.
	s.logger.EnableDebugLog(input.Verbose)

	return Config{Key: input.Key}, nil
}

// Run evaluates the key template into the final fingerprint string.
func (s FingerprintStep) Run(config Config) (Result, error) {
	hashString, err := s.keyEvaluator.Evaluate(config.Key)
	if err != nil {
		return Result{}, fmt.Errorf("evaluate key template %q: %w", config.Key, err)
	}

	// keytemplate degrades to an empty string (with warnings) when, for example,
	// a checksum matches no files. An empty cache key would silently collide
	// across builds — and repacking onto the wrong cached binary ships a broken
	// build — so we fail loudly instead.
	if hashString == "" {
		return Result{}, fmt.Errorf("evaluated key is empty — check that the files referenced in 'key' exist and any glob patterns match at least one file")
	}

	return Result{BundleHashString: hashString}, nil
}

// ExportOutputs writes BUNDLE_HASH_STRING via envman for subsequent steps.
func (s FingerprintStep) ExportOutputs(result Result) error {
	if err := s.outputExporter.ExportOutput(bundleHashStringKey, result.BundleHashString); err != nil {
		return fmt.Errorf("export %s: %w", bundleHashStringKey, err)
	}

	s.logger.Println()
	s.logger.Donef("Exported %s=%s", bundleHashStringKey, result.BundleHashString)
	s.logger.Printf("Use it as the restore-cache / save-cache key; gate the build on restore-cache's BITRISE_CACHE_HIT output.")
	return nil
}
