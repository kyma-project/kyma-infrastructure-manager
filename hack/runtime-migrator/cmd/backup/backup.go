package main

import (
	"context"
	"fmt"
	"github.com/gardener/gardener/pkg/apis/core/v1beta1"
	gardener_types "github.com/gardener/gardener/pkg/client/core/clientset/versioned/typed/core/v1beta1"
	"github.com/kyma-project/infrastructure-manager/hack/runtime-migrator-app/internal/backup"
	"github.com/kyma-project/infrastructure-manager/hack/runtime-migrator-app/internal/config"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/kubeconfig"
	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"log/slog"
	"time"
)

const (
	timeoutK8sOperation = 20 * time.Second
	expirationTime      = 60 * time.Minute
	runtimeIDAnnotation = "kcp.provisioner.kyma-project.io/runtime-id"
)

type Backup struct {
	shootClient        gardener_types.ShootInterface
	kubeconfigProvider kubeconfig.Provider
	outputWriter       backup.OutputWriter
	results            backup.Results
	cfg                config.Config
}

func NewBackup(cfg config.Config, kubeconfigProvider kubeconfig.Provider, shootClient gardener_types.ShootInterface) (Backup, error) {
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
		shoot, err := b.fetchShoot(ctx, shootList, runtimeID)
		if err != nil {
			errMsg := fmt.Sprintf("Failed to fetch shoot: %v", err)
			b.results.ErrorOccurred(runtimeID, "", errMsg)
			slog.Error(errMsg, "runtimeID", runtimeID)
			continue
		}

		if shootIsBeingDeleted(shoot) {
			continue
		}

		runtimeBackup, err := backuper.Do(ctx, *shoot)
		if err != nil {
			errMsg := fmt.Sprintf("Failed to backup runtime: %v", err)
			b.results.ErrorOccurred(runtimeID, shoot.Name, errMsg)
			slog.Error(errMsg, "runtimeID", runtimeID)
			continue
		}

		if b.cfg.IsDryRun {
			slog.Info("Runtime processed successfully (dry-run)", "runtimeID", runtimeID)
		} else {
			if err := b.outputWriter.Save(runtimeID, runtimeBackup); err != nil {
				errMsg := fmt.Sprintf("Failed to store backup: %v", err)
				b.results.ErrorOccurred(runtimeID, shoot.Name, errMsg)
				slog.Error(errMsg, "runtimeID", runtimeID)
				continue
			}
		}

		b.results.OperationSucceeded(runtimeID, shoot.Name)
		slog.Info("Runtime backup created successfully successfully", "runtimeID", runtimeID)
	}

	resultsFile, err := b.outputWriter.SaveBackupResults(b.results)
	if err != nil {
		return err
	}

	slog.Info(fmt.Sprintf("Backup completed. Successfully sopred backups: %d, Failed backups: %d", b.results.Succeeded, b.results.Failed))
	slog.Info(fmt.Sprintf("Backup results saved in: %s", resultsFile))

	return nil
}

func (b Backup) fetchShoot(ctx context.Context, shootList *v1beta1.ShootList, runtimeID string) (*v1beta1.Shoot, error) {
	shoot := findShoot(runtimeID, shootList)
	if shoot == nil {
		return nil, errors.New("shoot was deleted or the runtimeID is incorrect")
	}

	getCtx, cancel := context.WithTimeout(ctx, timeoutK8sOperation)
	defer cancel()

	// We are fetching the shoot from the gardener to make sure the runtime didn't get deleted during the migration process
	refreshedShoot, err := b.shootClient.Get(getCtx, shoot.Name, v1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return nil, errors.New("shoot was deleted")
		}

		return nil, err
	}

	return refreshedShoot, nil
}

func findShoot(runtimeID string, shootList *v1beta1.ShootList) *v1beta1.Shoot {
	for _, shoot := range shootList.Items {
		if shoot.Annotations[runtimeIDAnnotation] == runtimeID {
			return &shoot
		}
	}
	return nil
}

func shootIsBeingDeleted(shoot *v1beta1.Shoot) bool {
	return !shoot.DeletionTimestamp.IsZero()
}
