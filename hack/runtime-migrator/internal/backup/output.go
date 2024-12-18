package backup

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"sigs.k8s.io/yaml"
	"time"
)

type OutputWriter struct {
	NewResultsDir string
	BackupDir     string
}

const (
	backupFolderName = "backup"
)

func NewOutputWriter(outputDir string) (OutputWriter, error) {
	newResultsDir := path.Join(outputDir, fmt.Sprintf("backup-%s", time.Now().Format(time.RFC3339)))

	err := os.MkdirAll(newResultsDir, os.ModePerm)
	if err != nil {
		return OutputWriter{}, fmt.Errorf("failed to create results directory: %v", err)
	}

	backupDir := path.Join(newResultsDir, backupFolderName)

	err = os.MkdirAll(backupDir, os.ModePerm)
	if err != nil {
		return OutputWriter{}, fmt.Errorf("failed to create backup directory: %v", err)
	}

	return OutputWriter{
		NewResultsDir: newResultsDir,
		BackupDir:     backupDir,
	}, nil
}

func (ow OutputWriter) Save(runtimeID string, runtimeBackup RuntimeBackup) error {
	err := os.MkdirAll(fmt.Sprintf("%s/%s", ow.BackupDir, runtimeID), os.ModePerm)
	if err != nil {
		return err
	}

	return saveYaml(runtimeBackup.Shoot, fmt.Sprintf("%s/%s/%s.yaml", ow.BackupDir, runtimeID, runtimeBackup.Shoot.Name))
}

func (ow OutputWriter) SaveBackupResults(results Results) (string, error) {
	resultFile, err := json.Marshal(results.Results)
	if err != nil {
		return "", err
	}

	fileName := fmt.Sprintf("%s/backup-results.json", ow.NewResultsDir)
	return fileName, writeFile(fileName, resultFile)
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
