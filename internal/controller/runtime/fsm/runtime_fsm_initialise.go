package fsm

import (
	"context"

	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/internal/controller/metrics"
	"k8s.io/apimachinery/pkg/api/meta"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// There is a decision made to not rely on state of the Runtime CR we have already set
// All the states we set in the operator are about to be read only by the external clients

func sFnInitialize(ctx context.Context, m *fsm, s *systemState) (stateFn, *ctrl.Result, error) {
	instanceIsNotBeingDeleted := s.instance.GetDeletionTimestamp().IsZero()
	instanceHasFinalizer := controllerutil.ContainsFinalizer(&s.instance, m.Finalizer)
	provisioningCondition := meta.FindStatusCondition(s.instance.Status.Conditions, string(imv1.ConditionTypeRuntimeProvisioned))
	dryRunProvisioningCondition := meta.FindStatusCondition(s.instance.Status.Conditions, string(imv1.ConditionTypeRuntimeProvisionedDryRun))
	dryRunMode := s.instance.IsControlledByProvisioner()

	if instanceIsNotBeingDeleted && !instanceHasFinalizer {
		return addFinalizerAndRequeue(ctx, m, s)
	}

	if instanceIsNotBeingDeleted && s.shoot == nil && provisioningCondition == nil && dryRunProvisioningCondition == nil {
		m.log.Info("Update Runtime state to Pending - initialised")

		getConditionType := func() imv1.RuntimeConditionType {
			if dryRunMode {
				return imv1.ConditionTypeRuntimeProvisionedDryRun
			}
			return imv1.ConditionTypeRuntimeProvisioned
		}

		s.instance.UpdateStatePending(
			getConditionType(),
			imv1.ConditionReasonInitialized,
			"Unknown",
			"Runtime initialized",
		)
		return updateStatusAndRequeue()
	}

	shootNeedsToBeCreated := func() bool {
		if dryRunMode {
			return instanceIsNotBeingDeleted && dryRunProvisioningCondition != nil &&
				dryRunProvisioningCondition.Status != "True"
		}

		return instanceIsNotBeingDeleted && s.shoot == nil
	}

	if shootNeedsToBeCreated() {
		if !dryRunMode {
			return switchState(sFnCreateShoot)
		}

		m.log.Info("Gardener shoot does not exist, creating new one")
		return switchState(sFnCreateShootDryRun)
	}

	if instanceIsNotBeingDeleted && !dryRunMode {
		m.log.Info("Gardener shoot exists, processing")
		return switchState(sFnSelectShootProcessing)
	}

	// instance is being deleted
	if !instanceIsNotBeingDeleted && instanceHasFinalizer {
		if s.shoot != nil && !dryRunMode {
			m.log.Info("Delete instance resources")
			return switchState(sFnDeleteKubeconfig)
		}
		return removeFinalizerAndStop(ctx, m, s) // resource cleanup completed
	}

	m.log.Info("noting to reconcile, stopping fsm")
	return stopWithMetrics()
}

func addFinalizerAndRequeue(ctx context.Context, m *fsm, s *systemState) (stateFn, *ctrl.Result, error) {
	m.log.Info("adding finalizer")
	controllerutil.AddFinalizer(&s.instance, m.Finalizer)

	err := m.Update(ctx, &s.instance)
	if err != nil {
		return updateStatusAndStopWithError(err)
	}
	return requeue()
}

func removeFinalizerAndStop(ctx context.Context, m *fsm, s *systemState) (stateFn, *ctrl.Result, error) {
	m.log.Info("removing finalizer")
	runtimeID := s.instance.GetLabels()[metrics.RuntimeIDLabel]
	controllerutil.RemoveFinalizer(&s.instance, m.Finalizer)
	err := m.Update(ctx, &s.instance)
	if err != nil {
		return updateStatusAndStopWithError(err)
	}

	// remove from metrics
	m.Metrics.CleanUpRuntimeGauge(runtimeID, s.instance.Name)
	return stop()
}
