package main

import (
	"fmt"
	v1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/hack/runtime-migrator-app/internal/runtime"
	"os"
	"path"
)

type OutputWriter struct {
	outputDir            string
	comparisonResultsDir string
}

func NewOutputWriter(outputDir string) (OutputWriter, error) {

	err := os.MkdirAll(outputDir, os.ModePerm)
	if err != nil {
		return OutputWriter{}, fmt.Errorf("failed to create results directory: %v", err)
	}

	runtimesDir := path.Join(outputDir, "runtimes")

	err = os.MkdirAll(runtimesDir, os.ModePerm)
	if err != nil {
		return OutputWriter{}, fmt.Errorf("failed to create runtimes directory: %v", err)
	}

	comparisonResultsDir := path.Join(outputDir, "comparison-results")

	err = os.MkdirAll(comparisonResultsDir, os.ModePerm)
	if err != nil {
		return OutputWriter{}, fmt.Errorf("failed to create comparison results directory: %v", err)
	}

	return OutputWriter{
		outputDir:            outputDir,
		comparisonResultsDir: comparisonResultsDir,
	}, nil
}

func (ow OutputWriter) SaveMigrationResults(results MigrationResults) error {
	return nil
}

func (ow OutputWriter) SaveRuntimeCR(runtime v1.Runtime) error {
	return nil
}

func (ow OutputWriter) SaveComparisonResult(comparisonResult runtime.ShootComparisonResult) error {

	return nil
}
