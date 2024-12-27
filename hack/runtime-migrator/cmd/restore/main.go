package main

import (
	"fmt"
	"github.com/kyma-project/infrastructure-manager/hack/runtime-migrator-app/internal/init"
	"log/slog"
	"os"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func main() {
	slog.Info("Starting runtime-restorer")
	cfg := init.NewConfig()

	opts := zap.Options{
		Development: true,
	}
	logger := zap.New(zap.UseFlagOptions(&opts))
	logf.SetLogger(logger)

	gardenerNamespace := fmt.Sprintf("garden-%s", cfg.GardenerProjectName)

	_, err := init.SetupKubernetesKubeconfigProvider(cfg.GardenerKubeconfigPath, gardenerNamespace, expirationTime)
	if err != nil {
		slog.Error(fmt.Sprintf("Failed to create kubeconfig provider: %v", err))
		os.Exit(1)
	}

	_, err = init.CreateKcpClient(&cfg)
	if err != nil {
		slog.Error("Failed to create kcp client", slog.Any("error", err))
		os.Exit(1)
	}

	_, err = init.SetupGardenerShootClient(cfg.GardenerKubeconfigPath, gardenerNamespace)
	if err != nil {
		slog.Error("Failed to setup Gardener shoot client", slog.Any("error", err))
		os.Exit(1)
	}
}
