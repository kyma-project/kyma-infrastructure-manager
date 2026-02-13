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

func exposeShootStatusInfo(s *systemState, m *fsm) {
	if s.shoot != nil {
		s.instance.Status.ShootLastOperation = s.shoot.Status.LastOperation
		s.instance.Status.ShootLastErrors = s.shoot.Status.LastErrors
		if s.shoot.Status.LastErrors != nil && len(s.shoot.Status.LastErrors) > 0 {
			m.log.Info("Shoot last errors", "shootName", s.shoot.Name, "runtimeId", s.instance.Name, "errors", s.shoot.Status.LastErrors)
		}
	}
}
