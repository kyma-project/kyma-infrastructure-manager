package fsm

import (
	"context"
	"fmt"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/internal/log_level"
	"github.com/kyma-project/infrastructure-manager/internal/registrycache"
	registrycacheapi "github.com/kyma-project/registry-cache/api/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

func sFnFinalizeRegistryCache(ctx context.Context, m *fsm, s *systemState) (stateFn, *ctrl.Result, error) {

	if !m.RegistryCacheConfigControllerEnabled {
		return switchState(sFnConfigureSKR)
	}

	runtimeClient, err := m.RuntimeClientGetter.Get(ctx, s.instance)
	if err != nil {
		s.instance.UpdateStatePending(
			imv1.ConditionTypeRegistryCacheConfigured,
			imv1.ConditionReasonRegistryCacheGardenClusterConfigurationFailed,
			metav1.ConditionFalse,
			err.Error(),
		)
		m.log.Error(err, "Failed to get runtime client")

		return updateStatusAndRequeue()
	}

	secretSyncer := registrycache.NewGardenSecretSyncer(m.GardenClient, runtimeClient, fmt.Sprintf("garden-%s", m.ConverterConfig.Gardener.ProjectName), s.instance.Name)

	m.log.V(log_level.DEBUG).Info("Registry cache secrets deletion", "instance", s.instance.Name)
	err = secretSyncer.Delete(ctx, s.instance.Spec.Caching)
	if err != nil {
		s.instance.UpdateStatePending(
			imv1.ConditionTypeRegistryCacheConfigured,
			imv1.ConditionReasonRegistryCacheGardenClusterCleanupFailed,
			metav1.ConditionFalse,
			err.Error(),
		)
		m.log.Error(err, "Failed to delete not used registry cache secrets")

		return updateStatusAndRequeue()
	}

	if registryCacheExists(s.instance) {
		m.log.V(log_level.DEBUG).Info("Registry cache configuration exists", "instance", s.instance.Name)
		statusManager := registrycache.NewStatusManager(runtimeClient)

		m.log.V(log_level.DEBUG).Info("Registry cache CRs state set to Ready", "instance", s.instance.Name)
		err = statusManager.SetStatusReady(ctx, s.instance, registrycacheapi.ConditionReasonRegistryCacheConfigured)
		if err != nil {
			m.log.Error(err, "Failed to set registry cache status to ready")

			return requeue()
		}

		return ensureStatusConditionIsSetAndContinue(
			&s.instance,
			imv1.ConditionTypeRegistryCacheConfigured,
			imv1.ConditionReasonRegistryCacheConfigured,
			"Registry cache configured successfully",
			sFnConfigureSKR)
	}

	return switchState(sFnConfigureSKR)
}
