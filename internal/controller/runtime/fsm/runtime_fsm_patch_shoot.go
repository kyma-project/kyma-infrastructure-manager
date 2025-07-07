package fsm

import (
	"context"
	"fmt"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/extender"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/structuredauth"
	"reflect"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/internal/log_level"
	"github.com/kyma-project/infrastructure-manager/internal/registrycache"
	gardener_shoot "github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot"
	"github.com/kyma-project/infrastructure-manager/pkg/reconciler"
	"github.com/kyma-project/kim-snatch/api/v1beta1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const fieldManagerName = "kim"

func sFnPatchExistingShoot(ctx context.Context, m *fsm, s *systemState) (stateFn, *ctrl.Result, error) {
	data, err := m.AuditLogging.GetAuditLogData(
		s.instance.Spec.Shoot.Provider.Type,
		s.instance.Spec.Shoot.Region)

	if err != nil {
		m.log.Error(err, msgFailedToConfigureAuditlogs)
	}

	if err != nil && m.AuditLogMandatory {
		m.Metrics.IncRuntimeFSMStopCounter()
		return updateStatePendingWithErrorAndStop(
			&s.instance,
			imv1.ConditionTypeRuntimeProvisioned,
			imv1.ConditionReasonAuditLogError,
			msgFailedToConfigureAuditlogs)
	}

	oidcConfig := structuredauth.GetOIDCConfigOrDefault(s.instance, m.ConverterConfig.Kubernetes.DefaultOperatorOidc.ToOIDCConfig())

	cmName := fmt.Sprintf(extender.StructuredAuthConfigFmt, s.instance.Spec.Shoot.Name)
	err = structuredauth.CreateOrUpdateStructuredAuthConfigMap(
		ctx,
		m.SeedClient,
		types.NamespacedName{Name: cmName, Namespace: m.ShootNamesapace},
		oidcConfig,
	)

	if err != nil {
		m.Metrics.IncRuntimeFSMStopCounter()
		return updateStatePendingWithErrorAndStop(
			&s.instance,
			imv1.ConditionTypeRuntimeProvisioned,
			imv1.ConditionReasonOidcError,
			msgFailedStructuredConfigMap)
	}

	var rc []v1beta1.RegistryCacheConfig
	if s.instance.Spec.Caching != nil && s.instance.Spec.Caching.Enabled {
		runtimeClient, err := m.RuntimeClientGetter.Get(ctx, s.instance)
		if err != nil {
			m.log.Error(err, "Failed to get Runtime Client")

			m.Metrics.IncRuntimeFSMStopCounter()
			return updateStatePendingWithErrorAndStop(
				&s.instance,
				imv1.ConditionTypeRuntimeProvisioned,
				imv1.ConditionReasonRegistryCacheError,
				msgFailedToConfigureRegistryCache)
		}

		rc, err = registrycache.NewConfigExplorer(ctx, runtimeClient).GetRegistryCacheConfig()

		if err != nil {
			m.log.Error(err, "Failed to get Registry Cache Config")

			m.Metrics.IncRuntimeFSMStopCounter()
			return updateStatePendingWithErrorAndStop(
				&s.instance,
				imv1.ConditionTypeRuntimeProvisioned,
				imv1.ConditionReasonRegistryCacheError,
				msgFailedToConfigureRegistryCache)
		}
	}

	// NOTE: In the future we want to pass the whole shoot object here
	updatedShoot, err := convertPatch(&s.instance, gardener_shoot.PatchOpts{
		ConverterConfig:       m.ConverterConfig,
		AuditLogData:          data,
		MaintenanceTimeWindow: getMaintenanceTimeWindow(s, m),
		Workers:               s.shoot.Spec.Provider.Workers,
		ShootK8SVersion:       s.shoot.Spec.Kubernetes.Version,
		Extensions:            s.shoot.Spec.Extensions,
		Resources:             s.shoot.Spec.Resources,
		InfrastructureConfig:  s.shoot.Spec.Provider.InfrastructureConfig,
		ControlPlaneConfig:    s.shoot.Spec.Provider.ControlPlaneConfig,
		Log:                   ptr.To(m.log),
		RegistryCache:         rc,
	})

	if err != nil {
		m.log.Error(err, "Failed to convert Runtime instance to shoot object, exiting with no retry")
		m.Metrics.IncRuntimeFSMStopCounter()
		return updateStatePendingWithErrorAndStop(&s.instance, imv1.ConditionTypeRuntimeProvisioned, imv1.ConditionReasonConversionError, fmt.Sprintf("Runtime conversion error %v", err))
	}

	m.log.V(log_level.DEBUG).Info("Shoot converted successfully", "Name", updatedShoot.Name, "Namespace", updatedShoot.Namespace)

	// The additional Update function is required to fully replace shoot Workers collection with workers defined in updated runtime object.
	// This is a workaround for the sigs.k8s.io/controller-runtime/pkg/client, which does not support replacing the Workers collection with client.Patch
	// This could caused some workers to be not removed from the shoot object during update
	// More info: https://github.com/kyma-project/infrastructure-manager/issues/640

	if !workersAreEqual(s.shoot.Spec.Provider.Workers, updatedShoot.Spec.Provider.Workers) {
		copyShoot := s.shoot.DeepCopy()
		copyShoot.Spec.Provider.Workers = updatedShoot.Spec.Provider.Workers
		copyShoot.Spec.Provider.ControlPlaneConfig = updatedShoot.Spec.Provider.ControlPlaneConfig
		copyShoot.Spec.Provider.InfrastructureConfig = updatedShoot.Spec.Provider.InfrastructureConfig

		updateErr := m.SeedClient.Update(ctx, copyShoot,
			&client.UpdateOptions{
				FieldManager: fieldManagerName,
			})

		nextState, res, err := handleUpdateError(updateErr, m, s, "Failed to update shoot object, exiting with no retry", "Gardener API shoot update error")
		if nextState != nil {
			return nextState, res, err
		}
	}

	patchErr := m.SeedClient.Patch(ctx, &updatedShoot, client.Apply, &client.PatchOptions{
		FieldManager: fieldManagerName,
		Force:        ptr.To(true),
	})
	nextState, res, err := handleUpdateError(patchErr, m, s, "Failed to patch shoot object, exiting with no retry", "Gardener API shoot patch error")

	if nextState != nil {
		return nextState, res, err
	}

	err = handleForceReconciliationAnnotation(&s.instance, m, ctx)
	if err != nil {
		m.log.Error(err, "could not handle force reconciliation annotation. Scheduling for retry.")
		return requeue()
	}

	if updatedShoot.Generation == s.shoot.Generation {
		m.log.V(log_level.DEBUG).Info("Gardener shoot for runtime did not change after patch, moving to processing", "Name", s.shoot.Name, "Namespace", s.shoot.Namespace)

		s.instance.UpdateStatePending(
			imv1.ConditionTypeRuntimeProvisioned,
			imv1.ConditionReasonProcessing,
			"True",
			"Shoot patched without changes",
		)

		return switchState(sFnHandleKubeconfig)
	}

	m.log.V(log_level.DEBUG).Info("Gardener shoot for runtime patched successfully", "Name", s.shoot.Name, "Namespace", s.shoot.Namespace)

	s.instance.UpdateStatePending(
		imv1.ConditionTypeRuntimeProvisioned,
		imv1.ConditionReasonProcessing,
		"Unknown",
		"Shoot is pending for update after patch",
	)

	return updateStatusAndRequeueAfter(m.GardenerRequeueDuration)
}

func handleUpdateError(err error, m *fsm, s *systemState, errMsg, statusMsg string) (stateFn, *ctrl.Result, error) {
	if err != nil {
		if k8serrors.IsConflict(err) {
			m.log.Info("Gardener shoot for runtime is outdated, retrying", "Name", s.shoot.Name, "Namespace", s.shoot.Namespace)

			s.instance.UpdateStatePending(
				imv1.ConditionTypeRuntimeProvisioned,
				imv1.ConditionReasonProcessing,
				"Unknown",
				"Shoot is pending for update after conflict error",
			)

			return updateStatusAndRequeueAfter(m.GardenerRequeueDuration)
		}

		// We're retrying on Forbidden error because Gardener returns them from time too time for operations that are properly authorized.
		if k8serrors.IsForbidden(err) {
			m.log.Info("Gardener shoot for runtime is forbidden, retrying")

			s.instance.UpdateStatePending(
				imv1.ConditionTypeRuntimeProvisioned,
				imv1.ConditionReasonProcessing,
				"Unknown",
				"Shoot is pending for update after forbidden error",
			)

			return updateStatusAndRequeueAfter(m.GardenerRequeueDuration)
		}

		m.log.Error(err, errMsg)
		m.Metrics.IncRuntimeFSMStopCounter()
		return updateStatePendingWithErrorAndStop(&s.instance, imv1.ConditionTypeRuntimeProvisioned, imv1.ConditionReasonProcessingErr, fmt.Sprintf("%s: %v", statusMsg, err))
	}

	return nil, nil, nil
}

func workersAreEqual(workers []gardener.Worker, workers2 []gardener.Worker) bool {
	if len(workers) != len(workers2) {
		return false
	}

	for i := range workers {
		if !reflect.DeepEqual(workers[i], workers2[i]) {
			return false
		}
	}
	return true
}

func handleForceReconciliationAnnotation(runtime *imv1.Runtime, fsm *fsm, ctx context.Context) error {
	annotations := runtime.Annotations
	if reconciler.ShouldForceReconciliation(annotations) {
		fsm.log.Info("Force reconciliation annotation found, removing the annotation and continuing the reconciliation")
		delete(annotations, reconciler.ForceReconcileAnnotation)
		runtime.SetAnnotations(annotations)

		err := fsm.KcpClient.Update(ctx, runtime)
		if err != nil {
			return err
		}

	}
	return nil
}

func convertPatch(instance *imv1.Runtime, opts gardener_shoot.PatchOpts) (gardener.Shoot, error) {
	if err := instance.ValidateRequiredLabels(); err != nil {
		return gardener.Shoot{}, err
	}

	converter := gardener_shoot.NewConverterPatch(opts)
	newShoot, err := converter.ToShoot(*instance)
	if err != nil {
		return newShoot, err
	}

	return newShoot, nil
}

func updateStatePendingWithErrorAndStop(instance *imv1.Runtime,
	//nolint:unparam
	c imv1.RuntimeConditionType, r imv1.RuntimeConditionReason, msg string) (stateFn, *ctrl.Result, error) {
	instance.UpdateStatePending(c, r, "False", msg)
	return updateStatusAndStop()
}
