package fsm

import (
	"context"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/internal/registrycache"
	ctrl "sigs.k8s.io/controller-runtime"
)

func sFnSeedClusterPostProcessing(ctx context.Context, m *fsm, s *systemState) (stateFn, *ctrl.Result, error) {
	runtimeClient, err := m.RuntimeClientGetter.Get(ctx, s.instance)
	if err != nil {
		s.instance.UpdateStatePending(
			imv1.ConditionTypeRuntimeKubeconfigReady,
			imv1.ConditionReasonSeedClusterPostProcessingError,
			"False",
			err.Error(),
		)
		m.log.Error(err, "Failed to get runtime client")

		return updateStatusAndRequeue()
	}

	secretSyncer := registrycache.NewSecretSyncer(m.SeedClient, runtimeClient)
	registryCachesWitSecrets := getRegistryCachesWithSecrets(s.instance)

	if len(registryCachesWitSecrets) > 0 {
		err = secretSyncer.DeleteNotUsed(registryCachesWitSecrets)
		if err != nil {
			s.instance.UpdateStatePending(
				imv1.ConditionTypeRuntimeKubeconfigReady,
				imv1.ConditionReasonSeedClusterPostProcessingError,
				"False",
				err.Error(),
			)
			m.log.Error(err, "Failed to remove not used registry cache secrets")

			return updateStatusAndRequeue()
		}
	}

	return switchState(sFnPatchExistingShoot)
}
