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

func (ow OutputWriter) SaveMigrationResults(results Results) (string, error) {
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
	comparisonResultsForRuntimeDir := path.Join(ow.ComparisonResultsDir, comparisonResult.RuntimeID)
	err := os.MkdirAll(comparisonResultsForRuntimeDir, 0644)
	if err != nil {
		return err
	}

	if comparisonResult.Diff != nil {
		err = writeResultsToDiffFiles(comparisonResult.OriginalShoot.Name, comparisonResult.Diff, comparisonResultsForRuntimeDir)
		if err != nil {
			return err
		}
	}

	err = saveShootToFile(path.Join(comparisonResultsForRuntimeDir, "original-shoot.yaml"), comparisonResult.OriginalShoot)
	if err != nil {
		return err
	}

	return saveShootToFile(path.Join(comparisonResultsForRuntimeDir, "converted-shoot.yaml"), comparisonResult.ConvertedShoot)
}

func saveShootToFile(filePath string, shoot interface{}) error {
	shootAsYaml, err := yaml.Marshal(shoot)
	if err != nil {
		return err
	}

	err = os.WriteFile(filePath, shootAsYaml, 0644)
	if err != nil {
		return err
	}

	return nil
}

func writeResultsToDiffFiles(shootName string, difference *runtime.Difference, resultsDir string) error {
	writeAndCloseFunc := func(filePath string, text string) error {
		file, err := os.Create(filePath)
		if err != nil {
			return err
		}
		defer func() {
			if file != nil {
				err := file.Close()
				if err != nil {
					fmt.Printf("failed to close file: %v", err)
				}
			}
		}()

		_, err = file.Write([]byte(text))

		return err
	}

	diffFile := path.Join(resultsDir, fmt.Sprintf("%s.diff", shootName))

	return writeAndCloseFunc(diffFile, string(*difference))
}
