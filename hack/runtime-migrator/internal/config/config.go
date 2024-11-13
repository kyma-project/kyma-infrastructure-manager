package config

import (
	"flag"
	"fmt"
	v1 "github.com/kyma-project/infrastructure-manager/api/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/clientcmd"
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
	flag.StringVar(&result.InputType, "input-type", InputTypeJSON, "Type of input to be used. Possible values: **all** (will migrate all gardener shoots), and **json** (will migrate only cluster whose runtimeIds were passed as an input, see the example hack/runtime-migrator/input/runtimeids_sample.json).")
	flag.StringVar(&result.InputFilePath, "input-file-path", "/path/to/input/file", "Path to the input file containing RuntimeCRs to be migrated.")

	flag.Parse()

	printConfig(result)

	return result
}

func addToScheme(s *runtime.Scheme) error {
	for _, add := range []func(s *runtime.Scheme) error{
		corev1.AddToScheme,
		v1.AddToScheme,
	} {
		if err := add(s); err != nil {
			return fmt.Errorf("unable to add scheme: %w", err)
		}
	}
	return nil
}

type GetClient = func() (client.Client, error)

func (cfg *Config) Client() (client.Client, error) {
	restCfg, err := clientcmd.BuildConfigFromFlags("", cfg.KcpKubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("unable to fetch rest config: %w", err)
	}

	scheme := runtime.NewScheme()
	if err := addToScheme(scheme); err != nil {
		return nil, err
	}

	return client.New(restCfg, client.Options{
		Scheme: scheme,
	})
}

func CreateKcpClient(cfg *Config) (client.Client, error) {
	restCfg, err := clientcmd.BuildConfigFromFlags("", cfg.KcpKubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("unable to fetch rest config: %w", err)
	}

	scheme := runtime.NewScheme()
	if err := addToScheme(scheme); err != nil {
		return nil, err
	}

	var k8sClient, _ = client.New(restCfg, client.Options{
		Scheme: scheme,
	})

	return k8sClient, nil
}
