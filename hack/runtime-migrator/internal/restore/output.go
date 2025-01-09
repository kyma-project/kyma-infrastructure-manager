package restore

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"time"
)

type OutputWriter struct {
	NewResultsDir string
}

func NewOutputWriter(outputDir string) (OutputWriter, error) {
	newResultsDir := path.Join(outputDir, fmt.Sprintf("restore-%s", time.Now().Format(time.RFC3339)))

	err := os.MkdirAll(newResultsDir, os.ModePerm)
	if err != nil {
		return OutputWriter{}, fmt.Errorf("failed to create results directory: %v", err)
	}

	return OutputWriter{
		NewResultsDir: newResultsDir,
	}, nil
}

func (ow OutputWriter) SaveRestoreResults(results Results) (string, error) {
	resultFile, err := json.Marshal(results.Results)
	if err != nil {
		return "", err
	}

	fileName := fmt.Sprintf("%s/restore-results.json", ow.NewResultsDir)
	return fileName, writeFile(fileName, resultFile)
}

func writeFile(filePath string, content []byte) error {
	const writePermissions = 0644
	return os.WriteFile(filePath, content, writePermissions)
}
