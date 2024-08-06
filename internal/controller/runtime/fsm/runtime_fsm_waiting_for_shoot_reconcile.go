package fsm

import (
	"context"
	"fmt"
	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

func sFnWaitForShootReconcile(ctx context.Context, m *fsm, s *systemState) (stateFn, *ctrl.Result, error) {
	m.log.Info("Waiting for shoot reconcile state")

	if s.instance.HasTimeoutElapsed(m.UpdateTimeout) {
		m.log.Info(fmt.Sprintf("Shoot creation timeout for %s", s.shoot.Name))
		s.instance.UpdateStatePending(imv1.ConditionTypeRuntimeProvisioned, imv1.ConditionReasonShootProcessingTimeout, "False", "Shoot reconcile timeout")
		return updateStatusAndRequeue() // Requeue to clear annotation
	}

	switch s.shoot.Status.LastOperation.State {
	case gardener.LastOperationStateProcessing, gardener.LastOperationStatePending, gardener.LastOperationStateAborted:
		m.log.Info(fmt.Sprintf("Shoot %s is in %s state, scheduling for retry", s.shoot.Name, s.shoot.Status.LastOperation.State))

		s.instance.UpdateStatePending(
			imv1.ConditionTypeRuntimeProvisioned,
			imv1.ConditionReasonProcessing,
			"Unknown",
			"Shoot update is in progress")

		return updateStatusAndRequeueAfter(gardenerRequeueDuration)

	case gardener.LastOperationStateFailed:
		var reason ErrReason

		if len(s.shoot.Status.LastErrors) > 0 {
			reason = gardenerErrCodesToErrReason(s.shoot.Status.LastErrors...)
		}

		msg := fmt.Sprintf("error during cluster processing: reconcilation failed for shoot %s, reason: %s, exiting with no retry", s.shoot.Name, reason)
		m.log.Info(msg)

		s.instance.UpdateStatePending(
			imv1.ConditionTypeRuntimeProvisioned,
			imv1.ConditionReasonProcessingErr,
			"False",
			string(reason))

		return updateStatusAndStop()

	case gardener.LastOperationStateSucceeded:
		m.log.Info(fmt.Sprintf("Shoot %s successfully updated, moving to processing", s.shoot.Name))
		return ensureStatusConditionIsSetAndContinue(&s.instance, imv1.ConditionTypeRuntimeProvisioned, imv1.ConditionReasonProcessing, "Shoot update is completed", sFnProcessShoot)
	}

	m.log.Info("Update did not processed, exiting with no retry")
	return stop()
}
