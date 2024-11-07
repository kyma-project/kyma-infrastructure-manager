package fsm

import (
	"context"
	"fmt"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	imgardenerhandler "github.com/kyma-project/infrastructure-manager/pkg/gardener"
	ctrl "sigs.k8s.io/controller-runtime"
)

func ensureStatusConditionIsSetAndContinue(instance *imv1.Runtime, condType imv1.RuntimeConditionType, condReason imv1.RuntimeConditionReason, message string, next stateFn) (stateFn, *ctrl.Result, error) {
	if !instance.IsStateWithConditionAndStatusSet(imv1.RuntimeStatePending, condType, condReason, "True") {
		instance.UpdateStatePending(condType, condReason, "True", message)
		return updateStatusAndRequeue()
	}
	return switchState(next)
}

func ensureTerminatingStatusConditionAndContinue(instance *imv1.Runtime, condType imv1.RuntimeConditionType, condReason imv1.RuntimeConditionReason, message string, next stateFn) (stateFn, *ctrl.Result, error) {
	if !instance.IsStateWithConditionAndStatusSet(imv1.RuntimeStateTerminating, condType, condReason, "True") {
		instance.UpdateStateDeletion(condType, condReason, "True", message)
		return updateStatusAndRequeue()
	}
	return switchState(next)
}

func sFnWaitForShootCreation(_ context.Context, m *fsm, s *systemState) (stateFn, *ctrl.Result, error) {
	m.log.Info("Waiting for shoot creation state")

	switch s.shoot.Status.LastOperation.State {
	case gardener.LastOperationStateProcessing, gardener.LastOperationStatePending, gardener.LastOperationStateAborted, gardener.LastOperationStateError:
		m.log.Info(fmt.Sprintf("Shoot %s is in %s state, scheduling for retry", s.shoot.Name, s.shoot.Status.LastOperation.State))

		s.instance.UpdateStatePending(
			imv1.ConditionTypeRuntimeProvisioned,
			imv1.ConditionReasonShootCreationPending,
			"Unknown",
			"Shoot creation in progress")

		return updateStatusAndRequeueAfter(m.RCCfg.GardenerRequeueDuration)

	case gardener.LastOperationStateFailed:
		lastErrors := s.shoot.Status.LastErrors
		reason := imgardenerhandler.ToErrReason(lastErrors...)

		if imgardenerhandler.IsRetryable(lastErrors) {
			m.log.Info(fmt.Sprintf("Retryable gardener errors during cluster provisioning for Shoot %s, reason: %s, scheduling for retry", s.shoot.Name, reason))
			//TODO: should update status?
			return updateStatusAndRequeueAfter(m.RCCfg.GardenerRequeueDuration)
		}

		msg := fmt.Sprintf("Provisioning failed for shoot: %s ! Last state: %s, Description: %s", s.shoot.Name, s.shoot.Status.LastOperation.State, s.shoot.Status.LastOperation.Description)
		m.log.Info(msg)

		s.instance.UpdateStatePending(
			imv1.ConditionTypeRuntimeProvisioned,
			imv1.ConditionReasonCreationError,
			"False",
			"Shoot creation failed")

		m.Metrics.IncRuntimeFSMStopCounter()
		return updateStatusAndStop()

	case gardener.LastOperationStateSucceeded:
		m.log.Info(fmt.Sprintf("Shoot %s successfully created", s.shoot.Name))
		return ensureStatusConditionIsSetAndContinue(
			&s.instance,
			imv1.ConditionTypeRuntimeProvisioned,
			imv1.ConditionReasonShootCreationCompleted,
			"Shoot creation completed",
			sFnCreateKubeconfig)

	default:
		m.log.Info("WaitForShootCreation - unknown shoot operation state, stopping state machine", "RuntimeCR", s.instance.Name, "shoot", s.shoot.Name)
		return stopWithMetrics()
	}
}
