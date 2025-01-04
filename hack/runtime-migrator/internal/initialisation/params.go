package initialisation

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"
)

type Config struct {
	GardenerKubeconfigPath string
	KcpKubeconfigPath      string
	GardenerProjectName    string
	OutputPath             string
	IsDryRun               bool
	InputType              string
	InputFilePath          string
}

const (
	InputTypeTxt        = "txt"
	InputTypeJSON       = "json"
	TimeoutK8sOperation = 20 * time.Second
)

func PrintConfig(cfg Config) {
	log.Println("gardener-kubeconfig-path:", cfg.GardenerKubeconfigPath)
	log.Println("kcp-kubeconfig-path:", cfg.KcpKubeconfigPath)
	log.Println("gardener-project-name:", cfg.GardenerProjectName)
	log.Println("output-path:", cfg.OutputPath)
	log.Println("dry-run:", cfg.IsDryRun)
	log.Println("input-type:", cfg.InputType)
	log.Println("input-file-path:", cfg.InputFilePath)
	log.Println("")
}

// newConfig - creates new application configuration base on passed flags
func NewConfig() Config {
	result := Config{}
	flag.StringVar(&result.KcpKubeconfigPath, "kcp-kubeconfig-path", "/path/to/kcp/kubeconfig", "Path to the Kubeconfig file of KCP cluster.")
	flag.StringVar(&result.GardenerKubeconfigPath, "gardener-kubeconfig-path", "/path/to/gardener/kubeconfig", "Kubeconfig file for Gardener cluster.")
	flag.StringVar(&result.GardenerProjectName, "gardener-project-name", "gardener-project-name", "Name of the Gardener project.")
	flag.StringVar(&result.OutputPath, "output-path", "/tmp/", "Path where generated yamls will be saved. Directory has to exist.")
	flag.BoolVar(&result.IsDryRun, "dry-run", true, "Dry-run flag. Has to be set to 'false' otherwise it will not apply the Custom Resources on the KCP cluster.")
	flag.StringVar(&result.InputType, "input-type", InputTypeJSON, "Type of input to be used. Possible values: **txt** (see the example hack/runtime-migrator/input/runtimeids_sample.txt), and **json** (see the example hack/runtime-migrator/input/runtimeids_sample.json).")
	flag.StringVar(&result.InputFilePath, "input-file-path", "/path/to/input/file", "Path to the input file containing RuntimeCRs to be migrated.")

	flag.Parse()

	return result
}

func GetRuntimeIDsFromInputFile(cfg Config) ([]string, error) {
	var runtimeIDs []string
	var err error

	if cfg.InputType == InputTypeJSON {
		file, err := os.Open(cfg.InputFilePath)
		if err != nil {
			return nil, err
		}
		decoder := json.NewDecoder(file)
		err = decoder.Decode(&runtimeIDs)
	} else if cfg.InputType == InputTypeTxt {
		file, err := os.Open(cfg.InputFilePath)
		if err != nil {
			return nil, err
		}
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			runtimeIDs = append(runtimeIDs, scanner.Text())
		}
		err = scanner.Err()
	} else {
		return nil, fmt.Errorf("invalid input type: %s", cfg.InputType)
	}
	return runtimeIDs, err
}

type RestoreConfig struct {
	Config
	BackupDir   string
	RestoreCRB  bool
	RestoreOIDC bool
}

func NewRestoreConfig() RestoreConfig {
	restoreConfig := RestoreConfig{}

	flag.StringVar(&restoreConfig.KcpKubeconfigPath, "kcp-kubeconfig-path", "/path/to/kcp/kubeconfig", "Path to the Kubeconfig file of KCP cluster.")
	flag.StringVar(&restoreConfig.GardenerKubeconfigPath, "gardener-kubeconfig-path", "/path/to/gardener/kubeconfig", "Kubeconfig file for Gardener cluster.")
	flag.StringVar(&restoreConfig.GardenerProjectName, "gardener-project-name", "gardener-project-name", "Name of the Gardener project.")
	flag.StringVar(&restoreConfig.OutputPath, "output-path", "/tmp/", "Path where generated yamls will be saved. Directory has to exist.")
	flag.BoolVar(&restoreConfig.IsDryRun, "dry-run", true, "Dry-run flag. Has to be set to 'false' otherwise it will not apply the Custom Resources on the KCP cluster.")
	flag.StringVar(&restoreConfig.InputType, "input-type", InputTypeJSON, "Type of input to be used. Possible values: **txt** (see the example hack/runtime-migrator/input/runtimeids_sample.txt), and **json** (see the example hack/runtime-migrator/input/runtimeids_sample.json).")
	flag.StringVar(&restoreConfig.InputFilePath, "input-file-path", "/path/to/input/file", "Path to the input file containing RuntimeCRs to be migrated.")
	flag.StringVar(&restoreConfig.BackupDir, "backup-path", "/path/to/backup/dir", "Path to the directory containing backup.")
	flag.BoolVar(&restoreConfig.RestoreCRB, "restore-crbs", true, "Flag determining whether CRBs should be restored")
	flag.BoolVar(&restoreConfig.RestoreOIDC, "restore-oidcs", true, "Flag determining whether OIDCs should be restored")
	flag.Parse()

	return restoreConfig
}

func PrintRestoreConfig(cfg RestoreConfig) {
	log.Println("backup-path:", cfg.BackupDir)
	PrintConfig(cfg.Config)
}
