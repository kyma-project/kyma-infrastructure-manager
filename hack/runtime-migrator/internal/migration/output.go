package migration

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"time"

	v1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"sigs.k8s.io/yaml"
)

type OutputWriter struct {
	NewResultsDir        string
	RuntimeDir           string
	ComparisonResultsDir string
}

const (
	runtimesFolderName   = "runtimes"
	comparisonFolderName = "comparison-results"
)

func NewOutputWriter(outputDir string) (OutputWriter, error) {

	newResultsDir := path.Join(outputDir, fmt.Sprintf("migration-%s", time.Now().Format(time.RFC3339)))

	err := os.MkdirAll(newResultsDir, os.ModePerm)
	if err != nil {
		return OutputWriter{}, fmt.Errorf("failed to create results directory: %v", err)
	}

	runtimesDir := path.Join(newResultsDir, runtimesFolderName)

	err = os.MkdirAll(runtimesDir, os.ModePerm)
	if err != nil {
		return OutputWriter{}, fmt.Errorf("failed to create runtimes directory: %v", err)
	}

	comparisonResultsDir := path.Join(newResultsDir, comparisonFolderName)

	err = os.MkdirAll(comparisonResultsDir, os.ModePerm)
	if err != nil {
		return OutputWriter{}, fmt.Errorf("failed to create comparison results directory: %v", err)
	}

	return OutputWriter{
		NewResultsDir:        newResultsDir,
		RuntimeDir:           runtimesDir,
		ComparisonResultsDir: comparisonResultsDir,
	}, nil
}

func (ow OutputWriter) SaveMigrationResults(results Results) (string, error) {
	resultFile, err := json.Marshal(results.Results)
	if err != nil {
		return "", err
	}

	fileName := fmt.Sprintf("%s/migration-results.json", ow.NewResultsDir)
	return fileName, writeFile(fileName, resultFile)
}

func (ow OutputWriter) SaveRuntimeCR(runtime v1.Runtime) error {
	return saveYaml(runtime, fmt.Sprintf("%s/%s.yaml", ow.RuntimeDir, runtime.Name))
}

func saveYaml[T any](object T, path string) error {
	yamlBytes, err := yaml.Marshal(object)
	if err != nil {
		return err
	}

	return writeFile(path, yamlBytes)
}

func writeFile(filePath string, content []byte) error {
	const writePermissions = 0644
	return os.WriteFile(filePath, content, writePermissions)
}
