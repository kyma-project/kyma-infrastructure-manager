package fsm

import (
	"context"
	"fmt"
	"time"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/internal/log_level"
	imgardenerhandler "github.com/kyma-project/infrastructure-manager/pkg/gardener"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func ensureStatusConditionIsSetAndContinue(delay time.Duration, instance *imv1.Runtime, condType imv1.RuntimeConditionType, condReason imv1.RuntimeConditionReason, message string, next stateFn) (stateFn, *ctrl.Result, error) {
	if !instance.IsStateWithConditionAndStatusSet(imv1.RuntimeStatePending, condType, condReason, "True") {
		instance.UpdateStatePending(condType, condReason, metav1.ConditionTrue, message)
		return updateStatusAndRequeueAfter(delay)
	}
	return switchState(next)
}

func ensureTerminatingStatusConditionAndContinue(delay time.Duration, instance *imv1.Runtime, condType imv1.RuntimeConditionType, condReason imv1.RuntimeConditionReason, message string, next stateFn) (stateFn, *ctrl.Result, error) {
	if !instance.IsStateWithConditionAndStatusSet(imv1.RuntimeStateTerminating, condType, condReason, "True") {
		instance.UpdateStateDeletion(condType, condReason, metav1.ConditionTrue, message)
		return updateStatusAndRequeueAfter(delay)
	}
	return switchState(next)
}

func sFnWaitForShootCreation(ctx context.Context, m *fsm, s *systemState) (stateFn, *ctrl.Result, error) {
	m.log.V(log_level.DEBUG).Info("Waiting for shoot creation state")

	switch s.shoot.Status.LastOperation.State {
	case gardener.LastOperationStateProcessing, gardener.LastOperationStatePending, gardener.LastOperationStateAborted, gardener.LastOperationStateError:
		if stateNoMatchingSeeds(s.shoot) {
			m.log.Info(fmt.Sprintf("Shoot %s has no matching seeds, setting error state", s.shoot.Name))
			s.instance.UpdateStateFailed(
				imv1.ConditionTypeRuntimeProvisioned,
				imv1.ConditionReasonCreationError,
				"Shoot creation failed, no matching seeds")
			return updateStatusAndStop()
		}

		m.log.V(log_level.DEBUG).Info(fmt.Sprintf("Shoot %s is in %s state, scheduling for retry", s.shoot.Name, s.shoot.Status.LastOperation.State))

		s.instance.UpdateStatePending(
			imv1.ConditionTypeRuntimeProvisioned,
			imv1.ConditionReasonShootCreationPending,
			metav1.ConditionUnknown,
			"Shoot creation in progress")

		return updateStatusAndRequeueAfter(m.RequeueDurationShootCreate)

	case gardener.LastOperationStateFailed:
		lastErrors := s.shoot.Status.LastErrors
		reason := imgardenerhandler.ToErrReason(lastErrors...)

		if imgardenerhandler.IsRetryable(lastErrors) {
			if imgardenerhandler.IsInvalidSubnetError(lastErrors) {
				// Gardener classifies the transient AWS "InvalidSubnetID.NotFound" as a
				// non-retryable ERR_CONFIGURATION_PROBLEM, so it will not retry the failed
				// Shoot on its own. Stamping gardener.cloud/operation=retry is the only way
				// to start a new reconciliation loop on a failed Shoot
				if shootHasRetryOperationAnnotation(s.shoot) {
					m.log.Info("Retry annotation already set on shoot, waiting for Gardener to pick it up", "shoot", s.shoot.Name)
				} else {
					m.log.Info(fmt.Sprintf("Retryable InvalidSubnetID.NotFound error during cluster provisioning for Shoot %s, reason: %s, scheduling for Shoot reconciliation", s.shoot.Name, reason))
					patch := client.MergeFrom(s.shoot.DeepCopy())
					setRetryOperationAnnotation(s.shoot)
					if err := m.GardenClient.Patch(ctx, s.shoot, patch); err != nil {
						m.log.Error(err, "failed to set retry annotation on shoot", "shoot", s.shoot.Name)
					}
				}
				s.instance.UpdateStatePending(
					imv1.ConditionTypeRuntimeProvisioned,
					imv1.ConditionReasonShootCreationPending,
					metav1.ConditionUnknown,
					"Retryable gardener error InvalidSubnetID.NotFound during cluster provisioning")
				return updateStatusAndRequeueAfter(m.RequeueDurationShootCreate)
			}

			m.log.Info(fmt.Sprintf("Retryable gardener errors during cluster provisioning for Shoot %s, reason: %s, scheduling for retry", s.shoot.Name, reason))
			s.instance.UpdateStatePending(
				imv1.ConditionTypeRuntimeProvisioned,
				imv1.ConditionReasonShootCreationPending,
				metav1.ConditionUnknown,
				"Retryable gardener errors during cluster provisioning")
			return updateStatusAndRequeueAfter(m.RequeueDurationShootCreate)
		}

		msg := fmt.Sprintf("Provisioning failed for shoot: %s ! Last state: %s, Description: %s", s.shoot.Name, s.shoot.Status.LastOperation.State, s.shoot.Status.LastOperation.Description)
		m.log.Info(msg)

		s.instance.UpdateStateFailed(
			imv1.ConditionTypeRuntimeProvisioned,
			imv1.ConditionReasonCreationError,
			"Shoot creation failed")

		m.Metrics.IncRuntimeFSMStopCounter()
		return updateStatusAndStop()

	case gardener.LastOperationStateSucceeded:
		m.log.Info(fmt.Sprintf("Shoot %s successfully created", s.shoot.Name))
		return ensureStatusConditionIsSetAndContinue(
			m.StatusRequeueDelay,
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

func shootHasRetryOperationAnnotation(shoot *gardener.Shoot) bool {
	return shoot.GetAnnotations()[v1beta1constants.GardenerOperation] == v1beta1constants.ShootOperationRetry
}

func setRetryOperationAnnotation(shoot *gardener.Shoot) {
	annotations := shoot.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}
	annotations[v1beta1constants.GardenerOperation] = v1beta1constants.ShootOperationRetry
	shoot.SetAnnotations(annotations)
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
