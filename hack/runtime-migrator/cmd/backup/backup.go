package main

import (
	"context"
	"fmt"
	gardener_types "github.com/gardener/gardener/pkg/client/core/clientset/versioned/typed/core/v1beta1"
	"github.com/kyma-project/infrastructure-manager/hack/runtime-migrator-app/internal/backup"
	"github.com/kyma-project/infrastructure-manager/hack/runtime-migrator-app/internal/input"
	"github.com/kyma-project/infrastructure-manager/hack/runtime-migrator-app/internal/shoot"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/kubeconfig"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"log/slog"
	"time"
)

const (
	timeoutK8sOperation = 20 * time.Second
	expirationTime      = 60 * time.Minute
)

type Backup struct {
	shootClient        gardener_types.ShootInterface
	kubeconfigProvider kubeconfig.Provider
	outputWriter       backup.OutputWriter
	results            backup.Results
	cfg                input.Config
}

func NewBackup(cfg input.Config, kubeconfigProvider kubeconfig.Provider, shootClient gardener_types.ShootInterface) (Backup, error) {
	outputWriter, err := backup.NewOutputWriter(cfg.OutputPath)
	if err != nil {
		return Backup{}, err
	}

	return Backup{
		shootClient:        shootClient,
		kubeconfigProvider: kubeconfigProvider,
		outputWriter:       outputWriter,
		results:            backup.NewBackupResults(outputWriter.NewResultsDir),
		cfg:                cfg,
	}, nil
}

func (b Backup) Do(ctx context.Context, runtimeIDs []string) error {
	listCtx, cancel := context.WithTimeout(ctx, timeoutK8sOperation)
	defer cancel()

	shootList, err := b.shootClient.List(listCtx, v1.ListOptions{})
	if err != nil {
		return err
	}

	backuper := backup.NewBackuper(b.cfg.IsDryRun, b.kubeconfigProvider)

	for _, runtimeID := range runtimeIDs {
		shootToBackup, err := shoot.Fetch(ctx, shootList, b.shootClient, runtimeID)
		if err != nil {
			errMsg := fmt.Sprintf("Failed to fetch shoot: %v", err)
			b.results.ErrorOccurred(runtimeID, "", errMsg)
			slog.Error(errMsg, "runtimeID", runtimeID)

			continue
		}

		if shoot.IsBeingDeleted(shootToBackup) {
			errMsg := fmt.Sprintf("Shoot is being deleted: %v", err)
			b.results.ErrorOccurred(runtimeID, shootToBackup.Name, errMsg)
			slog.Error(errMsg, "runtimeID", runtimeID)
			continue
		}

		runtimeBackup, err := backuper.Do(ctx, *shootToBackup)
		if err != nil {
			errMsg := fmt.Sprintf("Failed to backup runtime: %v", err)
			b.results.ErrorOccurred(runtimeID, shootToBackup.Name, errMsg)
			slog.Error(errMsg, "runtimeID", runtimeID)

			continue
		}

		if b.cfg.IsDryRun {
			slog.Info("Runtime processed successfully (dry-run)", "runtimeID", runtimeID)

			continue
		}

		if err := b.outputWriter.Save(runtimeID, runtimeBackup); err != nil {
			errMsg := fmt.Sprintf("Failed to store backup: %v", err)
			b.results.ErrorOccurred(runtimeID, shootToBackup.Name, errMsg)
			slog.Error(errMsg, "runtimeID", runtimeID)

			continue
		}

		slog.Info("Runtime backup created successfully successfully", "runtimeID", runtimeID)
		b.results.OperationSucceeded(runtimeID, shootToBackup.Name)
	}

	resultsFile, err := b.outputWriter.SaveBackupResults(b.results)
	if err != nil {
		return err
	}

	slog.Info(fmt.Sprintf("Backup completed. Successfully stored backups: %d, Failed backups: %d", b.results.Succeeded, b.results.Failed))
	slog.Info(fmt.Sprintf("Backup results saved in: %s", resultsFile))

	return nil
}
