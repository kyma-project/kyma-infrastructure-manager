package main

import (
	"context"
	"fmt"
	"github.com/kyma-project/infrastructure-manager/hack/runtime-migrator-app/internal/initialisation"
	"log/slog"
	"os"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func main() {
	slog.Info("Starting runtime-backuper")
	cfg := initialisation.NewConfig()

	initialisation.PrintConfig(cfg)

	opts := zap.Options{
		Development: true,
	}
	logger := zap.New(zap.UseFlagOptions(&opts))
	logf.SetLogger(logger)

	gardenerNamespace := fmt.Sprintf("garden-%s", cfg.GardenerProjectName)

	shootClient, _, err := initialisation.SetupGardenerShootClients(cfg.GardenerKubeconfigPath, gardenerNamespace)
	if err != nil {
		slog.Error("Failed to setup Gardener shoot client", slog.Any("error", err))
		os.Exit(1)
	}

	kcpClient, err := initialisation.CreateKcpClient(&cfg)
	if err != nil {
		slog.Error("Failed to create kcp client", slog.Any("error", err))
		os.Exit(1)
	}

	backup, err := NewBackup(cfg, kcpClient, shootClient)
	if err != nil {
		slog.Error("Failed to initialize backup", slog.Any("error", err))
		os.Exit(1)
	}

	slog.Info("Reading runtimeIds from input file")
	runtimeIds, err := initialisation.GetRuntimeIDsFromInputFile(cfg)
	if err != nil {
		slog.Error("Failed to read runtime Ids from input", slog.Any("error", err))
		os.Exit(1)
	}

	err = backup.Do(context.Background(), runtimeIds)
	if err != nil {
		slog.Error("Failed to backup runtimes", slog.Any("error", err))
		os.Exit(1)
	}
}
