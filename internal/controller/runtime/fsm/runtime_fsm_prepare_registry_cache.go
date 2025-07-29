package fsm

import (
	"context"
	"fmt"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/internal/log_level"
	"github.com/kyma-project/infrastructure-manager/internal/registrycache"
	registrycacheapi "github.com/kyma-project/kim-snatch/api/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"
)

func sFnPrepareRegistryCache(ctx context.Context, m *fsm, s *systemState) (stateFn, *ctrl.Result, error) {
	if registryCacheExists(s.instance) {
		m.log.V(log_level.DEBUG).Info("Registry cache configuration exists", "instance", s.instance.Name)
		runtimeClient, err := m.RuntimeClientGetter.Get(ctx, s.instance)
		if err != nil {
			s.instance.UpdateStatePending(
				imv1.ConditionTypeRegistryCacheConfigured,
				imv1.ConditionReasonRegistryCacheGardenClusterConfigurationFailed,
				"False",
				err.Error(),
			)
			m.log.Error(err, "Failed to get runtime client")

			return updateStatusAndRequeue()
		}

		statusManager := registrycache.NewStatusManager(runtimeClient)
		secretSyncer := registrycache.NewSecretSyncer(m.GardenClient, runtimeClient, fmt.Sprintf("garden-%s", m.ConverterConfig.Gardener.ProjectName), s.instance.Name)

		m.log.V(log_level.DEBUG).Info("Registry cache CRs state set to Pending", "instance", s.instance.Name)
		err = statusManager.SetStatusPending(ctx, s.instance, registrycacheapi.ConditionTypeRegistryCacheConfigured, registrycacheapi.ConditionReasonRegistryCacheConfigured)
		if err != nil {
			m.log.Error(err, "Failed to set registry cache status to pending")

			return requeue()
		}

		m.log.V(log_level.DEBUG).Info("Registry cache secrets creation", "instance", s.instance.Name)
		err = secretSyncer.CreateOrUpdate(ctx, s.instance.Spec.Caching)
		if err != nil {
			s.instance.UpdateStatePending(
				imv1.ConditionTypeRegistryCacheConfigured,
				imv1.ConditionReasonRegistryCacheGardenClusterConfigurationFailed,
				"False",
				err.Error(),
			)
			m.log.Error(err, "Failed to sync registry cache secrets")

			err = statusManager.SetStatusFailed(ctx, s.instance, registrycacheapi.ConditionTypeRegistryCacheConfigured, registrycacheapi.ConditionReasonRegistryCacheCGardenClusterConfigurationFailed, err.Error())

			if err != nil {
				m.log.Error(err, "Failed to update registry cache status")
			}

			return updateStatusAndRequeue()
		}
	}

	return switchState(sFnPatchExistingShoot)
}
