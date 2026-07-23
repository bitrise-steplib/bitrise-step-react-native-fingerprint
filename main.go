package main

import (
	"fmt"
	"os"

	"github.com/bitrise-io/go-steputils/stepconf"
	"github.com/bitrise-io/go-steputils/tools"
)

// Config maps the step inputs (see step.yml) to Go fields.
type Config struct {
	FilePaths string `env:"file_paths,required"`
	KeyPrefix string `env:"key_prefix"`
	Verbose   bool   `env:"verbose"`
}

func main() {
	var cfg Config
	if err := stepconf.Parse(&cfg); err != nil {
		failf("Invalid input: %s", err)
	}
	stepconf.Print(cfg)
	fmt.Println()

	paths := parsePaths(cfg.FilePaths)
	if cfg.Verbose {
		log("Fingerprinting %d file(s):", len(paths))
		for _, p := range paths {
			log("  - %s", p)
		}
	}

	fingerprint, err := computeFingerprint(paths)
	if err != nil {
		failf("Failed to compute fingerprint: %s", err)
	}
	hashString := withKeyPrefix(cfg.KeyPrefix, fingerprint)

	if err := tools.ExportEnvironmentWithEnvman("BUNDLE_HASH_STRING", hashString); err != nil {
		failf("Failed to export BUNDLE_HASH_STRING: %s", err)
	}

	fmt.Println()
	log("Exported BUNDLE_HASH_STRING=%s", hashString)
	log("Use it as the restore-cache / save-cache key; gate the build on restore-cache's BITRISE_CACHE_HIT.")
	os.Exit(0)
}
