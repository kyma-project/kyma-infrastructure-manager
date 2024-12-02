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
		if stateNoMatchingSeeds(s.shoot) {
			m.log.Info(fmt.Sprintf("Shoot %s has no matching seeds, setting error state", s.shoot.Name))
			s.instance.UpdateStatePending(
				imv1.ConditionTypeRuntimeProvisioned,
				imv1.ConditionReasonCreationError,
				"False",
				"Shoot creation failed, no matching seeds")
			return updateStatusAndStop()
		}

		m.log.Info(fmt.Sprintf("Shoot %s is in %s state, scheduling for retry", s.shoot.Name, s.shoot.Status.LastOperation.State))

		s.instance.UpdateStatePending(
			imv1.ConditionTypeRuntimeProvisioned,
			imv1.ConditionReasonShootCreationPending,
			"Unknown",
			"Shoot creation in progress")

		return updateStatusAndRequeueAfter(m.RCCfg.RequeueDurationShootCreate)

	case gardener.LastOperationStateFailed:
		lastErrors := s.shoot.Status.LastErrors
		reason := imgardenerhandler.ToErrReason(lastErrors...)

		if imgardenerhandler.IsRetryable(lastErrors) {
			m.log.Info(fmt.Sprintf("Retryable gardener errors during cluster provisioning for Shoot %s, reason: %s, scheduling for retry", s.shoot.Name, reason))
			s.instance.UpdateStatePending(
				imv1.ConditionTypeRuntimeProvisioned,
				imv1.ConditionReasonShootCreationPending,
				"Unknown",
				"Retryable gardener errors during cluster provisioning")
			return updateStatusAndRequeueAfter(m.RCCfg.RequeueDurationShootCreate)
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
			sFnHandleKubeconfig)

	default:
		m.log.Info("WaitForShootCreation - unknown shoot operation state, stopping state machine", "RuntimeCR", s.instance.Name, "shoot", s.shoot.Name)
		return stopWithMetrics()
	}
}

func stateNoMatchingSeeds(shoot *gardener.Shoot) bool {
	if shoot == nil {
		return false
	}

	var seedsCount int
	var provider string
	_, err := fmt.Sscanf(shoot.Status.LastOperation.Description, "Failed to schedule Shoot: none out of the %d seeds has a matching provider for %q", &seedsCount, &provider)
	return err == nil
}
