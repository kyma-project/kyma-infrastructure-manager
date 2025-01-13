package main

import (
	"context"
	"fmt"
	"github.com/gardener/gardener/pkg/apis/core/v1beta1"
	gardener_types "github.com/gardener/gardener/pkg/client/core/clientset/versioned/typed/core/v1beta1"
	authenticationv1alpha1 "github.com/gardener/oidc-webhook-authenticator/apis/authentication/v1alpha1"
	"github.com/kyma-project/infrastructure-manager/hack/runtime-migrator-app/internal/backup"
	"github.com/kyma-project/infrastructure-manager/hack/runtime-migrator-app/internal/initialisation"
	"github.com/kyma-project/infrastructure-manager/hack/runtime-migrator-app/internal/restore"
	"github.com/kyma-project/infrastructure-manager/hack/runtime-migrator-app/internal/shoot"
	v12 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
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
	kcpClient             client.Client
	outputWriter          restore.OutputWriter
	results               restore.Results
	cfg                   initialisation.RestoreConfig
}

const fieldManagerName = "kim-restore"

func NewRestore(cfg initialisation.RestoreConfig, kcpClient client.Client, shootClient gardener_types.ShootInterface, dynamicGardenerClient client.Client) (Restore, error) {
	outputWriter, err := restore.NewOutputWriter(cfg.OutputPath)
	if err != nil {
		return Restore{}, err
	}

	return Restore{
		shootClient:           shootClient,
		dynamicGardenerClient: dynamicGardenerClient,
		kcpClient:             kcpClient,
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

	restorer := restore.NewBackupReader(r.cfg.BackupDir, r.cfg.RestoreCRB, r.cfg.RestoreOIDC)

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

		objectsToRestore, err := restorer.Do(runtimeID, currentShoot.Name)
		if err != nil {
			errMsg := fmt.Sprintf("Failed to read runtime from backup directory: %v", err)
			r.results.ErrorOccurred(runtimeID, currentShoot.Name, errMsg)
			slog.Error(errMsg, "runtimeID", runtimeID)

			continue
		}

		if currentShoot.Generation > objectsToRestore.OriginalShoot.Generation+1 {
			slog.Warn("Verify the current state of the system. Restore should be performed manually, as the backup may overwrite user's changes.", "runtimeID", runtimeID)
			r.results.AutomaticRestoreImpossible(runtimeID, currentShoot.Name)

			continue
		}

		if r.cfg.IsDryRun {
			slog.Info("Runtime processed successfully (dry-run)", "runtimeID", runtimeID)
			r.results.OperationSucceeded(runtimeID, currentShoot.Name, nil, nil)

			continue
		}

		appliedCRBs, appliedOIDC, err := r.applyResources(ctx, objectsToRestore, runtimeID)
		if err != nil {
			errMsg := fmt.Sprintf("Failed to apply resources: %v", err)
			r.results.ErrorOccurred(runtimeID, currentShoot.Name, errMsg)
			slog.Error(errMsg, "runtimeID", runtimeID)

			continue
		}

		slog.Info("Runtime restore performed successfully", "runtimeID", runtimeID)
		r.results.OperationSucceeded(runtimeID, currentShoot.Name, appliedCRBs, appliedOIDC)
	}

	resultsFile, err := r.outputWriter.SaveRestoreResults(r.results)
	if err != nil {
		return err
	}

	slog.Info(fmt.Sprintf("Restore completed. Successfully restored backups: %d, Failed operations: %d", r.results.Succeeded, r.results.Failed))
	slog.Info(fmt.Sprintf("Restore results saved in: %s", resultsFile))

	return nil
}

func (r Restore) applyResources(ctx context.Context, objectsToRestore backup.RuntimeBackup, runtimeID string) ([]v12.ClusterRoleBinding, []authenticationv1alpha1.OpenIDConnect, error) {
	err := r.applyShoot(ctx, objectsToRestore.ShootForPatch)
	if err != nil {
		return nil, nil, err
	}

	clusterClient, err := initialisation.GetRuntimeClient(ctx, r.kcpClient, runtimeID)
	if err != nil {
		return nil, nil, err
	}

	appliedCRBs, err := r.applyCRBs(ctx, clusterClient, objectsToRestore.ClusterRoleBindings)
	if err != nil {
		return nil, nil, err
	}

	appliedOIDC, err := r.applyOIDC(ctx, clusterClient, objectsToRestore.OIDCConfig)
	if err != nil {
		return nil, nil, err
	}

	return appliedCRBs, appliedOIDC, nil
}

func (r Restore) applyShoot(ctx context.Context, shoot v1beta1.Shoot) error {
	patchCtx, cancel := context.WithTimeout(ctx, timeoutK8sOperation)
	defer cancel()

	return r.dynamicGardenerClient.Patch(patchCtx, &shoot, client.Apply, &client.PatchOptions{
		FieldManager: fieldManagerName,
		Force:        ptr.To(true),
	})
}

func (r Restore) applyCRBs(ctx context.Context, clusterClient client.Client, crbs []v12.ClusterRoleBinding) ([]v12.ClusterRoleBinding, error) {
	appliedCRBs := make([]v12.ClusterRoleBinding, 0)

	for _, crb := range crbs {
		key := client.ObjectKey{
			Name:      crb.Name,
			Namespace: crb.Namespace,
		}
		applied, err := applyCRBIfDoesntExist(ctx, key, &crb, clusterClient)
		if err != nil {
			return nil, err
		}

		if applied {
			appliedCRBs = append(appliedCRBs, crb)
		}
	}

	return appliedCRBs, nil
}

func (r Restore) applyOIDC(ctx context.Context, clusterClient client.Client, oidcConfigs []authenticationv1alpha1.OpenIDConnect) ([]authenticationv1alpha1.OpenIDConnect, error) {
	appliedOIDCs := make([]authenticationv1alpha1.OpenIDConnect, 0)

	for _, oidc := range oidcConfigs {
		key := client.ObjectKey{
			Name:      oidc.Name,
			Namespace: oidc.Namespace,
		}
		applied, err := applyOIDCIfDoesntExist(ctx, key, &oidc, clusterClient)
		if err != nil {
			return nil, err
		}

		if applied {
			appliedOIDCs = append(appliedOIDCs, oidc)
		}
	}

	return appliedOIDCs, nil
}

func applyCRBIfDoesntExist(ctx context.Context, key client.ObjectKey, object *v12.ClusterRoleBinding, clusterClient client.Client) (bool, error) {
	getCtx, cancelGet := context.WithTimeout(ctx, timeoutK8sOperation)
	defer cancelGet()

	var existingObject v12.ClusterRoleBinding

	err := clusterClient.Get(getCtx, key, &existingObject, &client.GetOptions{})
	if err == nil {
		return false, nil
	}

	if err != nil && !errors.IsNotFound(err) {
		return false, err
	}

	createCtx, cancelCreate := context.WithTimeout(ctx, timeoutK8sOperation)
	defer cancelCreate()

	return true, clusterClient.Create(createCtx, object, &client.CreateOptions{})
}

func applyOIDCIfDoesntExist(ctx context.Context, key client.ObjectKey, object *authenticationv1alpha1.OpenIDConnect, clusterClient client.Client) (bool, error) {
	getCtx, cancelGet := context.WithTimeout(ctx, timeoutK8sOperation)
	defer cancelGet()

	var existingObject authenticationv1alpha1.OpenIDConnect

	err := clusterClient.Get(getCtx, key, &existingObject, &client.GetOptions{})
	if err == nil {
		return false, nil
	}

	if err != nil && !errors.IsNotFound(err) {
		return false, err
	}

	createCtx, cancelCreate := context.WithTimeout(ctx, timeoutK8sOperation)
	defer cancelCreate()

	return true, clusterClient.Create(createCtx, object, &client.CreateOptions{})
}
