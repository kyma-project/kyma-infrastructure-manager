package fsm

import (
	"context"
	"fmt"
	"strings"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	gardenerhelper "github.com/gardener/gardener/pkg/apis/core/v1beta1/helper"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

func ensureStatusConditionIsSetAndContinue(instance *imv1.Runtime, condType imv1.RuntimeConditionType, condReason imv1.RuntimeConditionReason, message string, next stateFn) (stateFn, *ctrl.Result, error) {
	if !instance.IsStateWithConditionAndStatusSet(imv1.RuntimeStatePending, condType, condReason, "True") {
		instance.UpdateStatePending(condType, condReason, "True", message)
		return updateStatusAndRequeue()
	}
	return switchState(next)
}

func ensurePendingStatusConditionIsSetAndRequeue(instance *imv1.Runtime, condType imv1.RuntimeConditionType, condReason imv1.RuntimeConditionReason, message string) (stateFn, *ctrl.Result, error) {
	if !instance.IsStateWithConditionAndStatusSet(imv1.RuntimeStatePending, condType, condReason, "Unknown") {
		instance.UpdateStatePending(condType, condReason, "Unknown", message)
		return updateStatusAndRequeue()
	}

	return updateStatusAndRequeueAfter(gardenerRequeueDuration)
}

func sFnWaitForShootCreation(_ context.Context, m *fsm, s *systemState) (stateFn, *ctrl.Result, error) {
	m.log.Info("Waiting for shoot creation state")

	if s.instance.HasTimeoutElapsed(m.ProvisionTimeout) {
		m.log.Info(fmt.Sprintf("Shoot creation timeout for %s", s.shoot.Name))
		s.instance.UpdateStatePending(imv1.ConditionTypeRuntimeProvisioned, imv1.ConditionReasonShootCreationTimeout, "False", "Shoot creation timeout")
		return updateStatusAndStop()
	}

	switch s.shoot.Status.LastOperation.State {
	case gardener.LastOperationStateProcessing, gardener.LastOperationStatePending, gardener.LastOperationStateAborted:

		m.log.Info(fmt.Sprintf("Shoot %s is in %s state, scheduling for retry", s.shoot.Name, s.shoot.Status.LastOperation.State))

		return ensurePendingStatusConditionIsSetAndRequeue(&s.instance, imv1.ConditionTypeRuntimeProvisioned, imv1.ConditionReasonShootCreationPending, "Shoot creation in progress")

	case gardener.LastOperationStateFailed:
		if gardenerhelper.HasErrorCode(s.shoot.Status.LastErrors, gardener.ErrorInfraRateLimitsExceeded) {
			m.log.Info(fmt.Sprintf("Error during cluster provisioning: Rate limits exceeded for Shoot %s, scheduling for retry", s.shoot.Name))
			return updateStatusAndRequeueAfter(gardenerRequeueDuration)
		}

		// also handle other retryable errors here
		// ErrorRetryableConfigurationProblem
		// ErrorRetryableInfraDependencies

		msg := fmt.Sprintf("Provisioning failed for shoot: %s ! Last state: %s, Description: %s", s.shoot.Name, s.shoot.Status.LastOperation.State, s.shoot.Status.LastOperation.Description)
		m.log.Info(msg)

		s.instance.UpdateStatePending(
			imv1.ConditionTypeRuntimeProvisioned,
			imv1.ConditionReasonCreationError,
			"False",
			"Shoot creation failed")

		return updateStatusAndStop()

	case gardener.LastOperationStateSucceeded:
		m.log.Info(fmt.Sprintf("Shoot %s successfully created", s.shoot.Name))
		return ensureStatusConditionIsSetAndContinue(&s.instance, imv1.ConditionTypeRuntimeProvisioned, imv1.ConditionReasonShootCreationCompleted, "Shoot creation completed", sFnCreateKubeconfig)

	default:
		m.log.Info("Unknown shoot operation state, exiting with no retry")
		return stop()
	}
}

func gardenerErrCodesToErrReason(lastErrors ...gardener.LastError) ErrReason {
	var codes []gardener.ErrorCode
	var vals []string

	for _, e := range lastErrors {
		if len(e.Codes) > 0 {
			codes = append(codes, e.Codes...)
		}
	}

	for _, code := range codes {
		vals = append(vals, string(code))
	}

	return ErrReason(strings.Join(vals, ", "))
}
