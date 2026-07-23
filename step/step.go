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
	FilePaths string `env:"file_paths,required"`
	KeyPrefix string `env:"key_prefix"`
	Verbose   bool   `env:"verbose"`
}

// Config is the processed and validated configuration derived from Input.
type Config struct {
	FilePaths []string
	KeyPrefix string
	Verbose   bool
}

// Result holds the step's computed output.
type Result struct {
	BundleHashString string
}

// FingerprintStep computes a deterministic fingerprint of dependency files and
// exports it as BUNDLE_HASH_STRING.
type FingerprintStep struct {
	logger         log.Logger
	inputParser    stepconf.InputParser
	outputExporter export.Exporter
}

// NewFingerprintStep wires the step with its dependencies.
func NewFingerprintStep(logger log.Logger, inputParser stepconf.InputParser, outputExporter export.Exporter) FingerprintStep {
	return FingerprintStep{
		logger:         logger,
		inputParser:    inputParser,
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
	s.logger.EnableDebugLog(input.Verbose)

	paths := parsePaths(input.FilePaths)
	if len(paths) == 0 {
		return Config{}, fmt.Errorf("no file paths provided in 'file_paths' (all lines were blank or comments)")
	}

	return Config{
		FilePaths: paths,
		KeyPrefix: input.KeyPrefix,
		Verbose:   input.Verbose,
	}, nil
}

// Run computes the fingerprint and applies the optional key prefix.
func (s FingerprintStep) Run(config Config) (Result, error) {
	s.logger.Debugf("Fingerprinting %d file(s):", len(config.FilePaths))
	for _, p := range config.FilePaths {
		s.logger.Debugf("  - %s", p)
	}

	fingerprint, err := computeFingerprint(config.FilePaths)
	if err != nil {
		return Result{}, fmt.Errorf("compute fingerprint: %w", err)
	}

	return Result{BundleHashString: withKeyPrefix(config.KeyPrefix, fingerprint)}, nil
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
