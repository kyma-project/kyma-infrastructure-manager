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
	runtimeDir := fmt.Sprintf("%s/%s", ow.BackupDir, runtimeID)
	err := os.MkdirAll(runtimeDir, os.ModePerm)
	if err != nil {
		return err
	}

	err = saveYaml(runtimeBackup.ShootForPatch, fmt.Sprintf("%s/%s-to-restore.yaml", runtimeDir, runtimeBackup.ShootForPatch.Name))
	if err != nil {
		return err
	}

	crbDir := fmt.Sprintf("%s/crb", runtimeDir)
	err = os.MkdirAll(crbDir, os.ModePerm)
	if err != nil {
		return err
	}

	for _, crb := range runtimeBackup.ClusterRoleBindings {
		err = saveYaml(crb, fmt.Sprintf("%s/%s.yaml", crbDir, crb.Name))
		if err != nil {
			return err
		}
	}

	oidcDir := fmt.Sprintf("%s/oidc", runtimeDir)
	err = os.MkdirAll(oidcDir, os.ModePerm)
	if err != nil {
		return err
	}

	for _, oidcConfig := range runtimeBackup.OIDCConfig {
		err = saveYaml(oidcConfig, fmt.Sprintf("%s/%s.yaml", oidcDir, oidcConfig.Name))
		if err != nil {
			return err
		}
	}

	return saveYaml(runtimeBackup.OriginalShoot, fmt.Sprintf("%s/%s/%s-original.yaml", ow.BackupDir, runtimeID, runtimeBackup.OriginalShoot.Name))
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
