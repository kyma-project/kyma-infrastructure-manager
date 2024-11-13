package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/gardener/gardener/pkg/apis/core/v1beta1"
	gardener_types "github.com/gardener/gardener/pkg/client/core/clientset/versioned/typed/core/v1beta1"
	"github.com/kyma-project/infrastructure-manager/hack/runtime-migrator-app/internal/config"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/kubeconfig"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

const (
	contextTimeout      = 5 * time.Minute
	expirationTime      = 60 * time.Minute
	runtimeIDAnnotation = "kcp.provisioner.kyma-project.io/runtime-id"
)

func main() {
	slog.Info("Starting runtime-migrator")
	cfg := config.NewConfig()

	opts := zap.Options{
		Development: true,
	}
	logger := zap.New(zap.UseFlagOptions(&opts))
	logf.SetLogger(logger)

	converterConfig, err := config.LoadConverterConfig(cfg)
	if err != nil {
		slog.Error(fmt.Sprintf("Unable to load converter config: %v", err))
		os.Exit(1)
	}

	gardenerNamespace := fmt.Sprintf("garden-%s", cfg.GardenerProjectName)

	kubeconfigProvider, err := setupKubernetesKubeconfigProvider(cfg.GardenerKubeconfigPath, gardenerNamespace, expirationTime)
	if err != nil {
		slog.Error(fmt.Sprintf("Failed to create kubeconfig provider: %v", err))
		os.Exit(1)
	}

	kcpClient, err := config.CreateKcpClient(&cfg)
	if err != nil {
		slog.Error("Failed to create kcp client: %v ", slog.Any("error", err))
		os.Exit(1)
	}

	gardenerShootClient, err := setupGardenerShootClient(cfg.GardenerKubeconfigPath, gardenerNamespace)
	if err != nil {
		slog.Error("Failed to setup Gardener shoot client: %v", slog.Any("error", err))
		os.Exit(1)
	}

	slog.Info("Migrating runtimes")
	migrator, err := NewMigration(cfg, converterConfig, kubeconfigProvider, kcpClient, gardenerShootClient)
	if err != nil {
		slog.Error("Failed to create migrator: %v", slog.Any("error", err))
		os.Exit(1)
	}

	slog.Info("Reading runtimeIds from stdin")
	runtimeIds, err := getRuntimeIDsFromStdin(cfg)
	if err != nil {
		slog.Error("Failed to read runtime Ids from input: %v", slog.Any("error", err))
		os.Exit(1)
	}

	err = migrator.Do(runtimeIds)
	if err != nil {
		slog.Error(fmt.Sprintf("Failed to migrate runtimes: %v", slog.Any("error", err)))
		os.Exit(1)
	}
}

func setupKubernetesKubeconfigProvider(kubeconfigPath string, namespace string, expirationTime time.Duration) (kubeconfig.Provider, error) {
	restConfig, err := gardener.NewRestConfigFromFile(kubeconfigPath)
	if err != nil {
		return kubeconfig.Provider{}, err
	}

	gardenerClientSet, err := gardener_types.NewForConfig(restConfig)
	if err != nil {
		return kubeconfig.Provider{}, err
	}

	gardenerClient, err := client.New(restConfig, client.Options{})
	if err != nil {
		return kubeconfig.Provider{}, err
	}

	shootClient := gardenerClientSet.Shoots(namespace)
	dynamicKubeconfigAPI := gardenerClient.SubResource("adminkubeconfig")

	err = v1beta1.AddToScheme(gardenerClient.Scheme())
	if err != nil {
		return kubeconfig.Provider{}, errors.Wrap(err, "failed to register Gardener schema")
	}

	return kubeconfig.NewKubeconfigProvider(shootClient,
		dynamicKubeconfigAPI,
		namespace,
		int64(expirationTime.Seconds())), nil
}

func getRuntimeIDsFromStdin(cfg config.Config) ([]string, error) {
	var runtimeIDs []string
	var err error

	if cfg.InputType == config.InputTypeJSON {
		decoder := json.NewDecoder(os.Stdin)
		err = decoder.Decode(&runtimeIDs)
	} else if cfg.InputType == config.InputTypeTxt {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			runtimeIDs = append(runtimeIDs, scanner.Text())
		}
		err = scanner.Err()
	}
	return runtimeIDs, err
}

func setupGardenerShootClient(kubeconfigPath, gardenerNamespace string) (gardener_types.ShootInterface, error) {
	restConfig, err := gardener.NewRestConfigFromFile(kubeconfigPath)
	if err != nil {
		return nil, err
	}

	gardenerClientSet, err := gardener_types.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	shootClient := gardenerClientSet.Shoots(gardenerNamespace)

	return shootClient, nil
}
