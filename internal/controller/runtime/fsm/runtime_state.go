package fsm

import (
	gardener_api "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
)

// the state of controlled system (k8s cluster)
type systemState struct {
	instance imv1.Runtime
	snapshot imv1.RuntimeStatus
	shoot    *gardener_api.Shoot
}

func (s *systemState) saveRuntimeStatus() {
	result := s.instance.Status.DeepCopy()
	if result == nil {
		result = &imv1.RuntimeStatus{}
	}
	s.snapshot = *result
}

func exposeShootStatusInfo(s *systemState) {
	s.instance.Status.ShootLastOperation = s.shoot.Status.LastOperation
	s.instance.Status.ShootLastErrors = s.shoot.Status.LastErrors
}
