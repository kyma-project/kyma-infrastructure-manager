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
	instanceIsBeingDeleted := !s.instance.GetDeletionTimestamp().IsZero()
	instanceHasFinalizer := controllerutil.ContainsFinalizer(&s.instance, m.Finalizer)
	provisioningCondition := meta.FindStatusCondition(s.instance.Status.Conditions, string(imv1.ConditionTypeRuntimeProvisioned))

	if !instanceIsBeingDeleted && !instanceHasFinalizer {
		return addFinalizerAndRequeue(ctx, m, s)
	}

	// instance is being deleted
	if instanceIsBeingDeleted {
		if s.shoot != nil {
			m.log.Info("Shoot is being deleted, removing kubeconfig", "Name", s.shoot.Name)
			return switchState(sFnDeleteKubeconfig)
		}

		m.log.Info("Shoot deleted", "Name", s.shoot.Name)

		if instanceHasFinalizer {
			return removeFinalizerAndStop(ctx, m, s) // resource cleanup completed
		}
		return stopWithMetrics()
	}

	if s.shoot == nil && provisioningCondition == nil {
		s.instance.UpdateStatePending(
			imv1.ConditionTypeRuntimeProvisioned,
			imv1.ConditionReasonInitialized,
			"Unknown",
			"Runtime initialized",
		)
		return updateStatusAndRequeue()
	}

	if s.shoot == nil {
		m.log.Info("Gardener shoot does not exist, creating new one")
		return switchState(sFnCreateShoot)
	}

	return switchState(sFnSelectShootProcessing)
}

func addFinalizerAndRequeue(ctx context.Context, m *fsm, s *systemState) (stateFn, *ctrl.Result, error) {
	controllerutil.AddFinalizer(&s.instance, m.Finalizer)

	err := m.Update(ctx, &s.instance)
	if err != nil {
		return updateStatusAndStopWithError(err)
	}
	return requeue()
}

func removeFinalizerAndStop(ctx context.Context, m *fsm, s *systemState) (stateFn, *ctrl.Result, error) {
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
