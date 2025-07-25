package fsm

import (
	"context"
	"fmt"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/internal/registrycache"
	ctrl "sigs.k8s.io/controller-runtime"
)

func sFnPrepareRegistryCache(ctx context.Context, m *fsm, s *systemState) (stateFn, *ctrl.Result, error) {
	if registryCacheExists(s.instance) {
		runtimeClient, err := m.RuntimeClientGetter.Get(ctx, s.instance)
		if err != nil {
			s.instance.UpdateStatePending(
				imv1.ConditionTypeRuntimeProvisioned,
				imv1.ConditionReasonRegistryCacheError,
				"False",
				err.Error(),
			)
			m.log.Error(err, "Failed to get runtime client")

			return updateStatusAndRequeue()
		}

		statusManager := registrycache.NewStatusManager(runtimeClient)
		secretSyncer := registrycache.NewSecretSyncer(m.GardenClient, runtimeClient, fmt.Sprintf("garden-%s", m.ConverterConfig.Gardener.ProjectName), s.instance.Name)

		err = statusManager.SetStatusPending(ctx, s.instance, string(registrycache.ConditionTypeCacheConfigured))
		if err != nil {
			m.log.Error(err, "Failed to set registry cache status to pending")

			return requeue()
		}

		err = secretSyncer.CreateOrUpdate(ctx, s.instance.Spec.Caching)
		if err != nil {
			s.instance.UpdateStatePending(
				imv1.ConditionTypeRuntimeKubeconfigReady,
				imv1.ConditionReasonRegistryCacheError,
				"False",
				err.Error(),
			)
			m.log.Error(err, "Failed to sync registry cache secrets")

			err = statusManager.SetStatusFailed(ctx, s.instance, string(registrycache.ConditionTypeCacheConfigured), err.Error())

			if err != nil {
				m.log.Error(err, "Failed to update registry cache status")
			}

			return updateStatusAndRequeue()
		}
	}

	return switchState(sFnPatchExistingShoot)
}
