package main

import (
	"context"
	"fmt"
	"github.com/kyma-project/infrastructure-manager/hack/runtime-migrator-app/internal/config"
	"log/slog"
	"os"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func main() {
	slog.Info("Starting runtime-backuper")
	cfg := config.NewConfig()

	opts := zap.Options{
		Development: true,
	}
	logger := zap.New(zap.UseFlagOptions(&opts))
	logf.SetLogger(logger)

	gardenerNamespace := fmt.Sprintf("garden-%s", cfg.GardenerProjectName)

	kubeconfigProvider, err := config.SetupKubernetesKubeconfigProvider(cfg.GardenerKubeconfigPath, gardenerNamespace, expirationTime)
	if err != nil {
		slog.Error(fmt.Sprintf("Failed to create kubeconfig provider: %v", err))
		os.Exit(1)
	}

	shootClient, err := config.SetupGardenerShootClient(cfg.GardenerKubeconfigPath, gardenerNamespace)
	if err != nil {
		slog.Error("Failed to setup Gardener shoot client", slog.Any("error", err))
		os.Exit(1)
	}

	backup, err := NewBackup(cfg, kubeconfigProvider, shootClient)
	if err != nil {
		slog.Error("Failed to initialize backup", slog.Any("error", err))
		os.Exit(1)
	}

	slog.Info("Reading runtimeIds from input file")
	runtimeIds, err := config.GetRuntimeIDsFromInputFile(cfg)
	if err != nil {
		slog.Error("Failed to read runtime Ids from input", slog.Any("error", err))
		os.Exit(1)
	}

	err = backup.Do(context.Background(), runtimeIds)
	if err != nil {
		slog.Error("Failed to read runtime Ids from input", slog.Any("error", err))
		os.Exit(1)
	}
}
