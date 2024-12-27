package main

import (
	"context"
	"fmt"
	gardener_types "github.com/gardener/gardener/pkg/client/core/clientset/versioned/typed/core/v1beta1"
	"github.com/kyma-project/infrastructure-manager/hack/runtime-migrator-app/internal/backup"
	init2 "github.com/kyma-project/infrastructure-manager/hack/runtime-migrator-app/internal/init"
	"github.com/kyma-project/infrastructure-manager/hack/runtime-migrator-app/internal/restore"
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

type Restore struct {
	shootClient        gardener_types.ShootInterface
	kubeconfigProvider kubeconfig.Provider
	outputWriter       backup.OutputWriter
	results            backup.Results
	cfg                init2.Config
}

func (r Restore) Do(ctx context.Context, runtimeIDs []string) error {
	listCtx, cancel := context.WithTimeout(ctx, timeoutK8sOperation)
	defer cancel()

	shootList, err := r.shootClient.List(listCtx, v1.ListOptions{})
	if err != nil {
		return err
	}

	_ = restore.NewRestorer(r.cfg.IsDryRun, r.kubeconfigProvider)

	for _, runtimeID := range runtimeIDs {
		shootToBackup, err := shoot.Fetch(ctx, shootList, r.shootClient, runtimeID)
		if err != nil {
			errMsg := fmt.Sprintf("Failed to fetch shoot: %v", err)
			r.results.ErrorOccurred(runtimeID, "", errMsg)
			slog.Error(errMsg, "runtimeID", runtimeID)

			continue
		}

		if shoot.IsBeingDeleted(shootToBackup) {
			errMsg := fmt.Sprintf("Shoot is being deleted: %v", err)
			r.results.ErrorOccurred(runtimeID, shootToBackup.Name, errMsg)
			slog.Error(errMsg, "runtimeID", runtimeID)
			continue
		}

		//

		if r.cfg.IsDryRun {
			slog.Info("Runtime processed successfully (dry-run)", "runtimeID", runtimeID)

			continue
		}

		slog.Info("Runtime backup created successfully successfully", "runtimeID", runtimeID)
		r.results.OperationSucceeded(runtimeID, shootToBackup.Name)
	}

	return nil
}
