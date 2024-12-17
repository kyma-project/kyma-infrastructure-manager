package config

import (
	"context"
	"encoding/json"
	"flag"
	"github.com/go-playground/validator/v10"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/extender/auditlogs"
	v12 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"log"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
	InputTypeTxt  = "txt"
	InputTypeJSON = "json"
)

func printConfig(cfg Config) {
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

	printConfig(result)

	return result
}

func GetAuditLogConfig(kcpClient client.Client) (auditlogs.Configuration, error) {
	var cm v12.ConfigMap
	key := types.NamespacedName{
		Name:      "audit-extension-config",
		Namespace: "kcp-system",
	}

	err := kcpClient.Get(context.Background(), key, &cm)
	if err != nil {
		return nil, err
	}

	configBytes := []byte(cm.Data["config"])

	var data auditlogs.Configuration
	if err := json.Unmarshal(configBytes, &data); err != nil {
		return nil, err
	}

	validate := validator.New(validator.WithRequiredStructEnabled())

	for _, nestedMap := range data {
		for _, auditLogData := range nestedMap {
			if err := validate.Struct(auditLogData); err != nil {
				return nil, err
			}
		}
	}

	return data, nil
}
