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

func (ow OutputWriter) SaveComparisonResult(comparisonResult ShootComparisonResult) error {
	comparisonResultsForRuntimeDir := path.Join(ow.ComparisonResultsDir, comparisonResult.RuntimeID)
	err := os.MkdirAll(comparisonResultsForRuntimeDir, os.ModePerm)
	if err != nil {
		return err
	}

	if comparisonResult.Diff != nil {
		err = writeResultsToDiffFile(comparisonResult.OriginalShoot.Name, comparisonResult.Diff, comparisonResultsForRuntimeDir)
		if err != nil {
			return err
		}
	}

	err = saveYaml(comparisonResult.OriginalShoot, path.Join(comparisonResultsForRuntimeDir, "original-shoot.yaml"))
	if err != nil {
		return err
	}

	return saveYaml(comparisonResult.ConvertedShoot, path.Join(comparisonResultsForRuntimeDir, "converted-shoot.yaml"))
}

func writeResultsToDiffFile(shootName string, difference *Difference, resultsDir string) error {
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
