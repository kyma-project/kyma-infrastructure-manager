package fsm

import (
	"context"
	"fmt"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/internal/registrycache"
	ctrl "sigs.k8s.io/controller-runtime"
)

func sFnGardenClusterPreProcessing(ctx context.Context, m *fsm, s *systemState) (stateFn, *ctrl.Result, error) { //nolint:unused
	runtimeClient, err := m.RuntimeClientGetter.Get(ctx, s.instance)
	if err != nil {
		s.instance.UpdateStatePending(
			imv1.ConditionTypeRuntimeKubeconfigReady,
			imv1.ConditionReasonSeedClusterPreProcessingError,
			"False",
			err.Error(),
		)
		m.log.Error(err, "Failed to get runtime client")

		return updateStatusAndRequeue()
	}

	// TODO: pass Garden namespace name
	secretSyncer := registrycache.NewSecretSyncer(m.GardenClient, runtimeClient, fmt.Sprintf("garden-%s", m.ConverterConfig.Gardener.ProjectName), s.instance.Name)
	registryCachesWitSecrets := getRegistryCachesWithSecrets(s.instance)

	if len(registryCachesWitSecrets) > 0 {
		err = secretSyncer.CreateOrUpdate(ctx, registryCachesWitSecrets)
		if err != nil {
			s.instance.UpdateStatePending(
				imv1.ConditionTypeRuntimeKubeconfigReady,
				imv1.ConditionReasonSeedClusterPreProcessingError,
				"False",
				err.Error(),
			)
			m.log.Error(err, "Failed to sync registry cache secrets")

			return updateStatusAndRequeue()
		}
	}

	if s.shoot == nil {
		m.log.Info("Gardener shoot does not exist, creating new one")
		return switchState(sFnCreateShoot)
	}

	return switchState(sFnSelectShootProcessing)
}

func getRegistryCachesWithSecrets(instance imv1.Runtime) []imv1.ImageRegistryCache { //nolint:unused
	var caches []imv1.ImageRegistryCache
	for _, cache := range instance.Spec.Caching {
		if cache.Config.SecretReferenceName != nil && *cache.Config.SecretReferenceName != "" {
			caches = append(caches, cache)
		}
	}
	return caches
}
