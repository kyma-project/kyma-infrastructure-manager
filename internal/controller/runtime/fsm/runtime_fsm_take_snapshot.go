package fsm

import (
	"context"

	gardener_api "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

// to save the runtime status at the begining of the reconciliation
func sFnTakeSnapshot(ctx context.Context, m *fsm, s *systemState) (stateFn, *ctrl.Result, error) {
	s.saveRuntimeStatus()

	var shoot gardener_api.Shoot
	err := m.ShootClient.Get(ctx, types.NamespacedName{
		Name:      s.instance.Spec.Shoot.Name,
		Namespace: m.ShootNamesapace,
	}, &shoot)

	if err != nil && !apierrors.IsNotFound(err) {
		m.log.Info("Failed to get Gardener shoot", "error", err)
		return updateStatusAndRequeueAfter(m.RCCfg.GardenerRequeueDuration)
	}

	if err == nil {
		s.shoot = &shoot
	}

	return switchState(sFnInitialize)
}
