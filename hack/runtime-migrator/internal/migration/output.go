package migration

import (
	"fmt"
	v1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/hack/runtime-migrator-app/internal/runtime"
	"os"
	"path"
	"sigs.k8s.io/yaml"
	"time"
)

type OutputWriter struct {
	outputDir            string
	runtimeDir           string
	comparisonResultsDir string
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
		outputDir:            outputDir,
		runtimeDir:           runtimesDir,
		comparisonResultsDir: comparisonResultsDir,
	}, nil
}

func (ow OutputWriter) SaveMigrationResults(results MigrationResults) error {
	return nil
}

func (ow OutputWriter) SaveRuntimeCR(runtime v1.Runtime) error {
	runtimeAsYaml, err := getYamlSpec(runtime)
	if err != nil {
		return err
	}

	return writeSpecToFile(ow.runtimeDir, runtime.Name, runtimeAsYaml)
}

func getYamlSpec(shoot v1.Runtime) ([]byte, error) {
	shootAsYaml, err := yaml.Marshal(shoot)
	return shootAsYaml, err
}

func writeSpecToFile(outputPath, runtimeID string, shootAsYaml []byte) error {
	var fileName = fmt.Sprintf("%s/%s", outputPath, runtimeID)

	const writePermissions = 0644
	return os.WriteFile(fileName, shootAsYaml, writePermissions)
}

func (ow OutputWriter) SaveComparisonResult(comparisonResult runtime.ShootComparisonResult) error {

	return nil
}
