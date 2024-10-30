package fsm

import (
	"context"
	"fmt"
	"strconv"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/extender"
	ctrl "sigs.k8s.io/controller-runtime"
)

type ErrReason string

func sFnSelectShootProcessing(_ context.Context, m *fsm, s *systemState) (stateFn, *ctrl.Result, error) {
	m.log.Info("Select shoot processing state")

	if s.shoot.Spec.DNS == nil || s.shoot.Spec.DNS.Domain == nil {
		msg := fmt.Sprintf("DNS Domain is not set yet for shoot: %s, scheduling for retry", s.shoot.Name)
		m.log.Info(msg)
		return requeueAfter(m.RCCfg.GardenerRequeueDuration)
	}

	lastOperation := s.shoot.Status.LastOperation
	if lastOperation == nil {
		msg := fmt.Sprintf("Last operation is nil for shoot: %s, scheduling for retry", s.shoot.Name)
		m.log.Info(msg)
		return requeueAfter(m.RCCfg.GardenerRequeueDuration)
	}

	patchShoot, err := shouldPatchShoot(&s.instance, s.shoot)
	if err != nil {
		msg := fmt.Sprintf("Failed to get applied generation for shoot: %s, scheduling for retry", s.shoot.Name)
		m.log.Error(err, msg)
		return requeueAfter(m.RCCfg.GardenerRequeueDuration)
	}

	if patchShoot {
		m.log.Info("Gardener shoot already exists, updating")
		return switchState(sFnPatchExistingShoot)
	}

	if s.instance.Status.State == imv1.RuntimeStatePending || s.instance.Status.State == "" {
		if lastOperation.Type == gardener.LastOperationTypeCreate {
			return switchState(sFnWaitForShootCreation)
		}

		if lastOperation.Type == gardener.LastOperationTypeReconcile {
			return switchState(sFnWaitForShootReconcile)
		}
	}

	// All other runtimes in Ready and Failed state will be not processed to mitigate massive reconciliation during restart
	msg := fmt.Sprintf("Stopping processing reconcile for runtime %s and shoot %s, exiting with no retry:", s.instance.Name, s.shoot.Name)
	m.log.Info(msg)
	return stop()
}

func shouldPatchShoot(runtime *imv1.Runtime, shoot *gardener.Shoot) (bool, error) {
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
