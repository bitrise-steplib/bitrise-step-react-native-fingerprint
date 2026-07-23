package main

import (
	"os"

	"github.com/bitrise-io/go-steputils/v2/cache/keytemplate"
	"github.com/bitrise-io/go-steputils/v2/export"
	"github.com/bitrise-io/go-steputils/v2/stepconf"
	"github.com/bitrise-io/go-utils/v2/command"
	"github.com/bitrise-io/go-utils/v2/env"
	"github.com/bitrise-io/go-utils/v2/fileutil"
	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-steplib/bitrise-step-react-native-fingerprint/step"
)

func main() {
	os.Exit(run())
}

func run() int {
	logger := log.NewLogger()
	fingerprintStep := createStep(logger)

	config, err := fingerprintStep.ProcessConfig()
	if err != nil {
		logger.Errorf("Process config: %s", err)
		return 1
	}

	result, err := fingerprintStep.Run(config)
	if err != nil {
		logger.Errorf("Run: %s", err)
		return 1
	}

	if err := fingerprintStep.ExportOutputs(result); err != nil {
		logger.Errorf("Export outputs: %s", err)
		return 1
	}

	return 0
}

func createStep(logger log.Logger) step.FingerprintStep {
	envRepository := env.NewRepository()
	inputParser := stepconf.NewInputParser(envRepository)
	keyEvaluator := keytemplate.NewModel(envRepository, logger)
	cmdFactory := command.NewFactory(envRepository)
	outputExporter := export.NewExporter(cmdFactory, fileutil.NewFileManager())

	return step.NewFingerprintStep(logger, inputParser, keyEvaluator, outputExporter)
}
