package step

import (
	"fmt"
	"os"

	"github.com/bitrise-io/go-steputils/v2/export"
	"github.com/bitrise-io/go-steputils/v2/stepconf"
	"github.com/bitrise-io/go-utils/v2/log"
)

const bundleHashStringKey = "BUNDLE_HASH_STRING"

// Input maps the step inputs (see step.yml) to Go fields.
type Input struct {
	ProjectDir  string `env:"project_dir"`
	Paths       string `env:"paths,required"`
	IgnorePaths string `env:"ignore_paths"`
	KeyPrefix   string `env:"key_prefix"`
	Verbose     bool   `env:"verbose"`
}

// Config is the processed and validated configuration derived from Input.
type Config struct {
	ProjectDir  string
	Paths       []string
	IgnorePaths []string
	KeyPrefix   string
}

// Result holds the step's computed output.
type Result struct {
	BundleHashString string
}

// FingerprintStep hashes the project's native inputs (files and directories,
// minus ignore patterns) and exports the result as BUNDLE_HASH_STRING.
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

	projectDir := input.ProjectDir
	if projectDir == "" {
		projectDir = "."
	}

	return Config{
		ProjectDir:  projectDir,
		Paths:       parseList(input.Paths),
		IgnorePaths: parseList(input.IgnorePaths),
		KeyPrefix:   input.KeyPrefix,
	}, nil
}

// Run hashes the configured paths (minus the ignore patterns) into the
// fingerprint and applies the optional key prefix.
func (s FingerprintStep) Run(config Config) (Result, error) {
	if len(config.Paths) == 0 {
		return Result{}, fmt.Errorf("no paths provided in 'paths'")
	}

	fingerprint, err := fingerprintPaths(os.DirFS(config.ProjectDir), config.Paths, config.IgnorePaths, s.logger)
	if err != nil {
		return Result{}, fmt.Errorf("fingerprint paths: %w", err)
	}
	if fingerprint == "" {
		return Result{}, fmt.Errorf("no files matched by 'paths' (after applying 'ignore_paths') — nothing to fingerprint")
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
