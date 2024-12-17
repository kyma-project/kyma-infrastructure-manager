package backup

type OutputWriter struct {
	NewResultsDir string
	BackupDir     string
}

const (
	backupFolderName = "backup"
)

func NewOutputWriter(outputDir string) (OutputWriter, error) {
	return OutputWriter{}, nil
}

func (ow OutputWriter) Save(runtimeBackup RuntimeBackup) error {
	return nil
}
