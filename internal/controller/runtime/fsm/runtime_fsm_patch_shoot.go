package fsm

import (
	"context"
	"fmt"
	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	gardener_shoot "github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot"
	"github.com/kyma-project/infrastructure-manager/pkg/reconciler"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/utils/ptr"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const fieldManagerName = "kim"

func sFnPatchExistingShoot(ctx context.Context, m *fsm, s *systemState) (stateFn, *ctrl.Result, error) {
	m.log.Info("Patch shoot state")

	data, err := m.AuditLogging.GetAuditLogData(
		s.instance.Spec.Shoot.Provider.Type,
		s.instance.Spec.Shoot.Region)

	if err != nil {
		m.log.Error(err, msgFailedToConfigureAuditlogs)
	}

	if err != nil && m.RCCfg.AuditLogMandatory {
		m.Metrics.IncRuntimeFSMStopCounter()
		return updateStatePendingWithErrorAndStop(
			&s.instance,
			imv1.ConditionTypeRuntimeProvisioned,
			imv1.ConditionReasonAuditLogError,
			msgFailedToConfigureAuditlogs)
	}

	// NOTE: In the future we want to pass the whole shoot object here
	updatedShoot, err := convertPatch(&s.instance, gardener_shoot.PatchOpts{
		ConverterConfig:      m.ConverterConfig,
		AuditLogData:         data,
		Workers:              s.shoot.Spec.Provider.Workers,
		ShootK8SVersion:      s.shoot.Spec.Kubernetes.Version,
		Extensions:           s.shoot.Spec.Extensions,
		Resources:            s.shoot.Spec.Resources,
		InfrastructureConfig: s.shoot.Spec.Provider.InfrastructureConfig,
		ControlPlaneConfig:   s.shoot.Spec.Provider.ControlPlaneConfig,
	})

	if err != nil {
		m.log.Error(err, "Failed to convert Runtime instance to shoot object, exiting with no retry")
		m.Metrics.IncRuntimeFSMStopCounter()
		return updateStatePendingWithErrorAndStop(&s.instance, imv1.ConditionTypeRuntimeProvisioned, imv1.ConditionReasonConversionError, "Runtime conversion error")
	}

	m.log.Info("Shoot converted successfully", "Name", updatedShoot.Name, "Namespace", updatedShoot.Namespace)

	// The additional Update function is required to fully replace shoot Workers collection with workers defined in updated runtime object.
	// This is a workaround for the sigs.k8s.io/controller-runtime/pkg/client, which does not support replacing the Workers collection with client.Patch
	// This could caused some workers to be not removed from the shoot object during update
	// More info: https://github.com/kyma-project/infrastructure-manager/issues/640

	if workersAreDifferent(s.shoot.Spec.Provider.Workers, updatedShoot.Spec.Provider.Workers) {
		copyShoot := s.shoot.DeepCopy()
		copyShoot.Spec.Provider.Workers = updatedShoot.Spec.Provider.Workers

		err = m.ShootClient.Update(ctx, copyShoot,
			&client.UpdateOptions{
				FieldManager: fieldManagerName,
			})

		if err != nil {
			if k8serrors.IsConflict(err) {
				m.log.Info("Gardener shoot for runtime is outdated, retrying", "Name", s.shoot.Name, "Namespace", s.shoot.Namespace)
				return updateStatusAndRequeueAfter(m.RCCfg.GardenerRequeueDuration)
			}

			m.log.Error(err, "Failed to update shoot worker list, exiting with no retry")
			m.Metrics.IncRuntimeFSMStopCounter()
			return updateStatePendingWithErrorAndStop(&s.instance, imv1.ConditionTypeRuntimeProvisioned, imv1.ConditionReasonProcessingErr, fmt.Sprintf("Gardener API shoot patch error: %v", err))
		}
	}

	err = m.ShootClient.Patch(ctx, &updatedShoot, client.Apply, &client.PatchOptions{
		FieldManager: fieldManagerName,
		Force:        ptr.To(true),
	})

	if err != nil {
		if k8serrors.IsConflict(err) {
			m.log.Info("Gardener shoot for runtime is outdated, retrying", "Name", s.shoot.Name, "Namespace", s.shoot.Namespace)
			return updateStatusAndRequeueAfter(m.RCCfg.GardenerRequeueDuration)
		}

		m.log.Error(err, "Failed to patch shoot object, exiting with no retry")
		m.Metrics.IncRuntimeFSMStopCounter()
		return updateStatePendingWithErrorAndStop(&s.instance, imv1.ConditionTypeRuntimeProvisioned, imv1.ConditionReasonProcessingErr, fmt.Sprintf("Gardener API shoot patch error: %v", err))
	}

	err = handleForceReconciliationAnnotation(&s.instance, m, ctx)
	if err != nil {
		m.log.Error(err, "could not handle force reconciliation annotation. Scheduling for retry.")
		return requeue()
	}

	if updatedShoot.Generation == s.shoot.Generation {
		m.log.Info("Gardener shoot for runtime did not change after patch, moving to processing", "Name", s.shoot.Name, "Namespace", s.shoot.Namespace)
		return switchState(sFnHandleKubeconfig)
	}

	m.log.Info("Gardener shoot for runtime patched successfully", "Name", s.shoot.Name, "Namespace", s.shoot.Namespace)

	s.instance.UpdateStatePending(
		imv1.ConditionTypeRuntimeProvisioned,
		imv1.ConditionReasonProcessing,
		"Unknown",
		"Shoot is pending for update",
	)

	return updateStatusAndRequeueAfter(m.RCCfg.GardenerRequeueDuration)
}

func workersAreDifferent(workers []gardener.Worker, workers2 []gardener.Worker) bool {
	if len(workers) != len(workers2) {
		return true
	}

	for i := range workers {
		if !reflect.DeepEqual(workers[i], workers2[i]) {
			return false
		}
	}
	return false
}

func handleForceReconciliationAnnotation(runtime *imv1.Runtime, fsm *fsm, ctx context.Context) error {
	annotations := runtime.Annotations
	if reconciler.ShouldForceReconciliation(annotations) {
		fsm.log.Info("Force reconciliation annotation found, removing the annotation and continuing the reconciliation")
		delete(annotations, reconciler.ForceReconcileAnnotation)
		runtime.SetAnnotations(annotations)

		err := fsm.Update(ctx, runtime)
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
