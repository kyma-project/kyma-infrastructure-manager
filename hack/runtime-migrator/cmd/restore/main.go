package main

import (
	"fmt"
	"github.com/kyma-project/infrastructure-manager/hack/runtime-migrator-app/internal/initialisation"
	"log/slog"
	"os"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func main() {
	slog.Info("Starting runtime-restorer")
	cfg := initialisation.NewRestoreConfig()

	initialisation.PrintRestoreConfig(cfg)

	opts := zap.Options{
		Development: true,
	}
	logger := zap.New(zap.UseFlagOptions(&opts))
	logf.SetLogger(logger)

	gardenerNamespace := fmt.Sprintf("garden-%s", cfg.GardenerProjectName)

	_, err := initialisation.SetupKubernetesKubeconfigProvider(cfg.GardenerKubeconfigPath, gardenerNamespace, expirationTime)
	if err != nil {
		slog.Error(fmt.Sprintf("Failed to create kubeconfig provider: %v", err))
		os.Exit(1)
	}

	_, err = initialisation.CreateKcpClient(&cfg.Config)
	if err != nil {
		slog.Error("Failed to create kcp client", slog.Any("error", err))
		os.Exit(1)
	}

	_, err = initialisation.SetupGardenerShootClient(cfg.GardenerKubeconfigPath, gardenerNamespace)
	if err != nil {
		slog.Error("Failed to setup Gardener shoot client", slog.Any("error", err))
		os.Exit(1)
	}
}
