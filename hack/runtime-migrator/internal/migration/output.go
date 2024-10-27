package migration

import (
	"encoding/json"
	"fmt"
	v1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/hack/runtime-migrator-app/internal/runtime"
	"os"
	"path"
	"sigs.k8s.io/yaml"
	"time"
)

type OutputWriter struct {
	NewResultsDir        string
	RuntimeDir           string
	ComparisonResultsDir string
}

func NewOutputWriter(outputDir string) (OutputWriter, error) {

	newResultsDir := path.Join(outputDir, fmt.Sprintf("migration-%s", time.Now().Format(time.RFC3339)))

	err := os.MkdirAll(newResultsDir, os.ModePerm)
	if err != nil {
		return OutputWriter{}, fmt.Errorf("failed to create results directory: %v", err)
	}

	runtimesDir := path.Join(newResultsDir, "runtimes")

	err = os.MkdirAll(runtimesDir, os.ModePerm)
	if err != nil {
		return OutputWriter{}, fmt.Errorf("failed to create runtimes directory: %v", err)
	}

	comparisonResultsDir := path.Join(newResultsDir, "comparison-results")

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

func (ow OutputWriter) SaveMigrationResults(results MigrationResults) (string, error) {
	resultFile, err := json.Marshal(results.Results)
	if err != nil {
		return "", err
	}

	fileName := fmt.Sprintf("%s/migration-results.json", ow.NewResultsDir)
	const writePermissions = 0644

	return fileName, os.WriteFile(fileName, resultFile, writePermissions)
}

func (ow OutputWriter) SaveRuntimeCR(runtime v1.Runtime) error {
	runtimeAsYaml, err := getYamlSpec(runtime)
	if err != nil {
		return err
	}

	return writeSpecToFile(ow.RuntimeDir, runtime.Name, runtimeAsYaml)
}

func getYamlSpec(shoot v1.Runtime) ([]byte, error) {
	shootAsYaml, err := yaml.Marshal(shoot)
	return shootAsYaml, err
}

func writeSpecToFile(outputPath, runtimeID string, shootAsYaml []byte) error {
	var fileName = fmt.Sprintf("%s/%s.yaml", outputPath, runtimeID)

	const writePermissions = 0644
	return os.WriteFile(fileName, shootAsYaml, writePermissions)
}

func (ow OutputWriter) SaveComparisonResult(comparisonResult runtime.ShootComparisonResult) error {

	return nil
}
