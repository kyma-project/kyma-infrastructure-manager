package fsm

import (
	"context"
	"fmt"
	"strconv"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/go-logr/logr"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/internal/log_level"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/extender"
	reconciler "github.com/kyma-project/infrastructure-manager/pkg/reconciler"
	ctrl "sigs.k8s.io/controller-runtime"
)

func sFnSelectShootProcessing(_ context.Context, m *fsm, s *systemState) (stateFn, *ctrl.Result, error) {
	if s.shoot.Spec.DNS == nil || s.shoot.Spec.DNS.Domain == nil {
		m.log.V(log_level.DEBUG).Info("DNS Domain is not set yet for shoot, scheduling for retry", "RuntimeCR", s.instance.Name, "shoot", s.shoot.Name)
		m.Metrics.SetRuntimeStates(s.instance)
		return requeueAfter(m.GardenerRequeueDuration)
	}

	lastOperation := s.shoot.Status.LastOperation
	if lastOperation == nil {
		m.log.V(log_level.DEBUG).Info("Last operation is nil for shoot, scheduling for retry", "RuntimeCR", s.instance.Name, "shoot", s.shoot.Name)
		m.Metrics.SetRuntimeStates(s.instance)
		return requeueAfter(m.GardenerRequeueDuration)
	}

	LogLastErrors(s, m)

	patchShoot, err := shouldPatchShoot(&s.instance, s.shoot, &m.log)
	if err != nil {
		m.log.Error(err, "Failed to get applied generation for shoot", "RuntimeCR", s.instance.Name, "shoot", s.shoot.Name)
		m.Metrics.SetRuntimeStates(s.instance)
		return requeueAfter(m.GardenerRequeueDuration)
	}

	if patchShoot {
		return switchState(sFnSyncRegistryCacheGardenSecrets)
	}

	if s.instance.Status.State == imv1.RuntimeStatePending || s.instance.Status.State == "" {
		if lastOperation.Type == gardener.LastOperationTypeCreate {
			return switchState(sFnWaitForShootCreation)
		}

		if lastOperation.Type == gardener.LastOperationTypeReconcile {
			return switchState(sFnWaitForShootReconcile)
		}
	}

	shootStatus := s.shoot.Status

	// Guard against premature stop() when Runtime state is stale in the informer cache
	// (e.g. after force-annotation removal triggers re-enqueue before the status-write
	// watch event propagates). We consult the Shoot directly: if Gardener still has
	// unprocessed work (generation drift or active operation), route to the appropriate
	// wait state instead of falling through to stop() with no retry.
	if s.instance.Status.State == imv1.RuntimeStateReady ||
		s.instance.Status.State == imv1.RuntimeStateFailed {
		shootMidReconcile := s.shoot.Generation > shootStatus.ObservedGeneration ||
			lastOperation.State == gardener.LastOperationStatePending ||
			lastOperation.State == gardener.LastOperationStateProcessing

		if shootMidReconcile {
			m.log.Info("Shoot indicates active reconciliation despite Runtime state Ready/Failed, switching to wait state",
				"RuntimeCR", s.instance.Name,
				"shoot", s.shoot.Name,
				"shootGeneration", s.shoot.Generation,
				"shootObservedGeneration", shootStatus.ObservedGeneration,
				"lastOperationState", lastOperation.State,
				"lastOperationType", lastOperation.Type)

			if lastOperation.Type == gardener.LastOperationTypeCreate {
				return switchState(sFnWaitForShootCreation)
			}
			return switchState(sFnWaitForShootReconcile)
		}
	}

	// All other runtimes in Ready and Failed state will be not processed to mitigate massive reconciliation during restart
	m.log.Info("Stopping processing reconcile, exiting with no retry", "RuntimeCR", s.instance.Name, "shoot", s.shoot.Name, "function", "sFnSelectShootProcessing")
	return stop()
}

func LogLastErrors(s *systemState, m *fsm) {
	if s.shoot == nil {
		return
	}

	status := s.shoot.Status
	state := status.LastOperation.State
	lastErrors := status.LastErrors
	stateRequiresErrorLogging := state == gardener.LastOperationStatePending ||
		state == gardener.LastOperationStateFailed ||
		state == gardener.LastOperationStateError

	if stateRequiresErrorLogging && len(lastErrors) > 0 {
		var errorsDescriptions string
		for i, lastError := range lastErrors {
			errorsDescriptions += fmt.Sprintf("%d: %s; ", i+1, lastError.Description)
		}
		logMessage := fmt.Sprintf("runtimeID: %s, Last operation state: %s, last errors: %s", s.instance.Name, state, errorsDescriptions)
		m.log.Info(logMessage)
	}
}

func shouldPatchShoot(runtime *imv1.Runtime, shoot *gardener.Shoot, logger *logr.Logger) (bool, error) {
	if reconciler.ShouldSuspendReconciliation(runtime.Annotations) {
		msg := fmt.Sprintf(`Reconciliation is suspended. Remove "%s" annotation to resume reconciliation`, reconciler.SuspendReconcileAnnotation)
		logger.Info(msg)
		return false, nil
	}

	if reconciler.ShouldForceReconciliation(runtime.Annotations) {
		return true, nil
	}

	runtimeGeneration := runtime.GetGeneration()
	appliedGenerationString, found := shoot.GetAnnotations()[extender.ShootRuntimeGenerationAnnotation]

	if !found {
		return true, nil
	}

	appliedGeneration, err := strconv.ParseInt(appliedGenerationString, 10, 64)
	if err != nil {
		return false, err
	}

	return appliedGeneration < runtimeGeneration, nil
}
