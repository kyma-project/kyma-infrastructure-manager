package main

import (
	"context"
	"fmt"
	gardener_types "github.com/gardener/gardener/pkg/client/core/clientset/versioned/typed/core/v1beta1"
	runtimev1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/hack/runtime-migrator-app/internal/backup"
	"github.com/kyma-project/infrastructure-manager/hack/runtime-migrator-app/internal/initialisation"
	"github.com/kyma-project/infrastructure-manager/hack/runtime-migrator-app/internal/shoot"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/kubeconfig"
	"github.com/pkg/errors"
	rbacv1 "k8s.io/api/rbac/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"log/slog"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"slices"
	"time"
)

const (
	timeoutK8sOperation = 20 * time.Second
	fieldManagerName    = "kim-backup"
)

type Backup struct {
	shootClient        gardener_types.ShootInterface
	kubeconfigProvider kubeconfig.Provider
	kcpClient          client.Client
	outputWriter       backup.OutputWriter
	results            backup.Results
	cfg                initialisation.BackupConfig
}

func NewBackup(cfg initialisation.BackupConfig, kcpClient client.Client, shootClient gardener_types.ShootInterface) (Backup, error) {
	outputWriter, err := backup.NewOutputWriter(cfg.OutputPath)
	if err != nil {
		return Backup{}, err
	}

	return Backup{
		shootClient:  shootClient,
		kcpClient:    kcpClient,
		outputWriter: outputWriter,
		results:      backup.NewBackupResults(outputWriter.NewResultsDir),
		cfg:          cfg,
	}, nil
}

func (b Backup) Do(ctx context.Context, runtimeIDs []string) error {
	listCtx, cancel := context.WithTimeout(ctx, timeoutK8sOperation)
	defer cancel()

	shootList, err := b.shootClient.List(listCtx, v1.ListOptions{})
	if err != nil {
		return err
	}

	backuper := backup.NewBackuper(b.cfg.IsDryRun, b.kcpClient, timeoutK8sOperation)

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

		runtimeClient, err := initialisation.GetRuntimeClient(ctx, b.kcpClient, runtimeID)
		if err != nil {
			errMsg := fmt.Sprintf("Failed to get kubernetes client for runtime: %v", err)
			b.results.ErrorOccurred(runtimeID, shootToBackup.Name, errMsg)
			slog.Error(errMsg, "runtimeID", runtimeID)

			continue
		}
		runtimeBackup, err := backuper.Do(ctx, runtimeClient, *shootToBackup)
		if err != nil {
			errMsg := fmt.Sprintf("Failed to backup runtime: %v", err)
			b.results.ErrorOccurred(runtimeID, shootToBackup.Name, errMsg)
			slog.Error(errMsg, "runtimeID", runtimeID)

			continue
		}

		if err := b.outputWriter.Save(runtimeID, runtimeBackup); err != nil {
			errMsg := fmt.Sprintf("Failed to store backup: %v", err)
			b.results.ErrorOccurred(runtimeID, shootToBackup.Name, errMsg)
			slog.Error(errMsg, "runtimeID", runtimeID)

			continue
		}

		if b.cfg.IsDryRun {
			slog.Info("Runtime processed successfully (dry-run)", "runtimeID", runtimeID)
			b.results.OperationSucceeded(runtimeID, shootToBackup.Name, nil, false)

			continue
		}

		deprecatedCRBs, err := labelDeprecatedCRBs(ctx, runtimeClient)
		if err != nil {
			errMsg := fmt.Sprintf("Failed to deprecate Cluster Role Bindings: %v", err)
			b.results.ErrorOccurred(runtimeID, shootToBackup.Name, errMsg)
			slog.Error(errMsg, "runtimeID", runtimeID)

			continue
		}

		if b.cfg.SetControlledByKim {
			err := setControlledByKim(ctx, b.kcpClient, runtimeID)
			if err != nil {
				errMsg := fmt.Sprintf("Failed to set the rutnime to be controlled by KIM: %v", err)
				b.results.ErrorOccurred(runtimeID, shootToBackup.Name, errMsg)
				slog.Error(errMsg, "runtimeID", runtimeID)
				continue
			}
		}

		slog.Info("Runtime backup created successfully", "runtimeID", runtimeID)
		b.results.OperationSucceeded(runtimeID, shootToBackup.Name, deprecatedCRBs, b.cfg.SetControlledByKim)
	}

	resultsFile, err := b.outputWriter.SaveBackupResults(b.results)
	if err != nil {
		return err
	}

	slog.Info(fmt.Sprintf("Backup completed. Successfully stored backups: %d, Failed backups: %d", b.results.Succeeded, b.results.Failed))
	slog.Info(fmt.Sprintf("Backup results saved in: %s", resultsFile))

	return nil
}

func labelDeprecatedCRBs(ctx context.Context, runtimeClient client.Client) ([]rbacv1.ClusterRoleBinding, error) {
	var crbList rbacv1.ClusterRoleBindingList

	listCtx, cancel := context.WithTimeout(ctx, timeoutK8sOperation)
	defer cancel()

	selector, err := labels.Parse("reconciler.kyma-project.io/managed-by=reconciler,app=kyma")
	if err != nil {
		return nil, err
	}

	err = runtimeClient.List(listCtx, &crbList, &client.ListOptions{
		LabelSelector: selector,
	})

	if err != nil {
		return nil, err
	}

	deprecatedCRBs := slices.DeleteFunc(crbList.Items, func(clusterRoleBinding rbacv1.ClusterRoleBinding) bool {
		if clusterRoleBinding.RoleRef.Kind != "ClusterRole" || clusterRoleBinding.RoleRef.Name != "cluster-admin" {
			return true
		}
		// leave only cluster-admin CRBs where at least one subject is of a user type
		if slices.ContainsFunc(clusterRoleBinding.Subjects, func(subject rbacv1.Subject) bool { return subject.Kind == rbacv1.UserKind }) {
			return false
		}
		return true
	})

	patchCRB := func(clusterRoleBinding rbacv1.ClusterRoleBinding) error {
		patchCtx, cancelPatch := context.WithTimeout(ctx, timeoutK8sOperation)
		defer cancelPatch()

		clusterRoleBinding.Kind = "ClusterRoleBinding"
		clusterRoleBinding.APIVersion = "rbac.authorization.k8s.io/v1"
		clusterRoleBinding.ManagedFields = nil

		return runtimeClient.Patch(patchCtx, &clusterRoleBinding, client.Apply, &client.PatchOptions{
			FieldManager: fieldManagerName,
		})
	}

	for _, clusterRoleBinding := range deprecatedCRBs {
		clusterRoleBinding.ObjectMeta.Labels["kyma-project.io/deprecation"] = "to-be-removed-soon"
		err := patchCRB(clusterRoleBinding)
		if err != nil {
			if err != nil {
				return nil, errors.Wrap(err, fmt.Sprintf("failed to update ClusterRoleBinding with deprecation label %s", clusterRoleBinding.Name))
			}
		}
	}

	return deprecatedCRBs, nil
}

func setControlledByKim(ctx context.Context, kcpClient client.Client, runtimeID string) error {
	getCtx, cancelGet := context.WithTimeout(ctx, timeoutK8sOperation)
	defer cancelGet()

	key := types.NamespacedName{
		Name:      runtimeID,
		Namespace: "kcp-system",
	}
	var runtime runtimev1.Runtime

	err := kcpClient.Get(getCtx, key, &runtime, &client.GetOptions{})
	if err != nil {
		return err
	}

	runtime.Labels["kyma-project.io/controlled-by-provisioner"] = "false"

	patchCtx, cancelPatch := context.WithTimeout(ctx, timeoutK8sOperation)
	defer cancelPatch()

	runtime.Kind = "Runtime"
	runtime.APIVersion = "infrastructuremanager.kyma-project.io/v1"
	runtime.ManagedFields = nil

	return kcpClient.Patch(patchCtx, &runtime, client.Apply, &client.PatchOptions{
		FieldManager: fieldManagerName,
	})
}
