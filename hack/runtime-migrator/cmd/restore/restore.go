package main

import (
	"context"
	"fmt"
	gardener_types "github.com/gardener/gardener/pkg/client/core/clientset/versioned/typed/core/v1beta1"
	"github.com/kyma-project/infrastructure-manager/hack/runtime-migrator-app/internal/initialisation"
	"github.com/kyma-project/infrastructure-manager/hack/runtime-migrator-app/internal/restore"
	"github.com/kyma-project/infrastructure-manager/hack/runtime-migrator-app/internal/shoot"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/kubeconfig"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"log/slog"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

const (
	timeoutK8sOperation = 20 * time.Second
	expirationTime      = 60 * time.Minute
)

type Restore struct {
	shootClient           gardener_types.ShootInterface
	dynamicGardenerClient client.Client
	kubeconfigProvider    kubeconfig.Provider
	outputWriter          restore.OutputWriter
	results               restore.Results
	cfg                   initialisation.RestoreConfig
}

const fieldManagerName = "kim"

func NewRestore(cfg initialisation.RestoreConfig, kubeconfigProvider kubeconfig.Provider, shootClient gardener_types.ShootInterface, dynamicGardenerClient client.Client) (Restore, error) {
	outputWriter, err := restore.NewOutputWriter(cfg.OutputPath)
	if err != nil {
		return Restore{}, err
	}

	return Restore{
		shootClient:           shootClient,
		dynamicGardenerClient: dynamicGardenerClient,
		kubeconfigProvider:    kubeconfigProvider,
		outputWriter:          outputWriter,
		results:               restore.NewRestoreResults(outputWriter.NewResultsDir),
		cfg:                   cfg,
	}, err
}

func (r Restore) Do(ctx context.Context, runtimeIDs []string) error {
	listCtx, cancel := context.WithTimeout(ctx, timeoutK8sOperation)
	defer cancel()

	shootList, err := r.shootClient.List(listCtx, v1.ListOptions{})
	if err != nil {
		return err
	}

	restorer := restore.NewRestorer(r.cfg.BackupDir)

	for _, runtimeID := range runtimeIDs {
		currentShoot, err := shoot.Fetch(ctx, shootList, r.shootClient, runtimeID)
		if err != nil {
			errMsg := fmt.Sprintf("Failed to fetch shoot: %v", err)
			r.results.ErrorOccurred(runtimeID, "", errMsg)
			slog.Error(errMsg, "runtimeID", runtimeID)

			continue
		}

		if shoot.IsBeingDeleted(currentShoot) {
			errMsg := fmt.Sprintf("Shoot is being deleted: %v", err)
			r.results.ErrorOccurred(runtimeID, currentShoot.Name, errMsg)
			slog.Error(errMsg, "runtimeID", runtimeID)
			continue
		}

		shootToRestore, err := restorer.Do(context.Background(), runtimeID, currentShoot.Name)
		if err != nil {
			errMsg := fmt.Sprintf("Failed to restore runtime: %v", err)
			r.results.ErrorOccurred(runtimeID, currentShoot.Name, errMsg)
			slog.Error(errMsg, "runtimeID", runtimeID)
			continue
		}

		if r.cfg.IsDryRun {
			slog.Info("Runtime processed successfully (dry-run)", "runtimeID", runtimeID)

			continue
		}

		err = r.dynamicGardenerClient.Patch(ctx, &shootToRestore, client.Apply, &client.PatchOptions{
			FieldManager: fieldManagerName,
			Force:        ptr.To(true),
		})

		if err != nil {
			errMsg := fmt.Sprintf("Failed to restore runtime: %v", err)
			r.results.ErrorOccurred(runtimeID, currentShoot.Name, errMsg)
			slog.Error(errMsg, "runtimeID", runtimeID)
			continue
		}

		slog.Info("Runtime backup created successfully successfully", "runtimeID", runtimeID)
		r.results.OperationSucceeded(runtimeID, currentShoot.Name)
	}

	return nil
}
