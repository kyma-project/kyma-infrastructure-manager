package fsm

import (
	"context"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/internal/log_level"
	"github.com/kyma-project/infrastructure-manager/internal/rtbootstrapper"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	msgStatusCheckFailed      = "Runtime bootstrapper status check failed"
	msgInstallationFailed     = "Runtime bootstrapper installation failed"
	msgInstallationInProgress = "Runtime bootstrapper installation in progress"
	msgInstallationCompleted  = "Runtime bootstrapper installation completed"
)

func sFnInitializeRuntimeBootstrapper(ctx context.Context, m *fsm, s *systemState) (stateFn, *ctrl.Result, error) {
	if !m.RuntimeBootstrapperEnabled || m.RuntimeBootstrapperInstaller == nil {
		m.log.V(log_level.DEBUG).Info("Runtime bootstrapper installation is disabled")
		return switchState(sFnFinalizeRegistryCache)
	}

	status, err := m.RuntimeBootstrapperInstaller.Status(ctx, s.instance)
	if err != nil {
		m.log.Error(err, "Failed to get runtime bootstrapper installation status")
		s.instance.UpdateStatePending(
			imv1.ConditionTypeRuntimeBootstrapperReady,
			imv1.ConditionReasonRuntimeBootstrapperStatusUnknown,
			"False",
			msgStatusCheckFailed,
		)
		return updateStatusAndRequeue()
	}

	switch status {
	case rtbootstrapper.StatusNotStarted:
		{
			err := m.RuntimeBootstrapperInstaller.Install(ctx, s.instance)
			if err != nil {
				m.log.Error(err, "Failed to start runtime bootstrapper installation")
				s.instance.UpdateStatePending(
					imv1.ConditionTypeRuntimeBootstrapperReady,
					imv1.ConditionReasonRuntimeBootstrapperInstallationFailed,
					"False",
					msgInstallationFailed,
				)
			} else {
				s.instance.UpdateStatePending(
					imv1.ConditionTypeRuntimeBootstrapperReady,
					imv1.ConditionReasonRuntimeBootstrapperInstallationInProgress,
					"False",
					msgInstallationInProgress,
				)
			}

			return updateStatusAndRequeue()
		}
	case rtbootstrapper.StatusInProgress:
		{
			m.log.V(log_level.DEBUG).Info("Runtime bootstrapper installation in progress")
			s.instance.UpdateStatePending(
				imv1.ConditionTypeRuntimeBootstrapperReady,
				imv1.ConditionReasonRuntimeBootstrapperInstallationInProgress,
				"False",
				msgInstallationInProgress,
			)

			return updateStatusAndRequeue()
		}
	case rtbootstrapper.StatusFailed:
		{
			m.log.Error(err, "Runtime bootstrapper installation failed")
			s.instance.UpdateStatePending(
				imv1.ConditionTypeRuntimeBootstrapperReady,
				imv1.ConditionReasonRuntimeBootstrapperInstallationFailed,
				"False",
				msgInstallationFailed,
			)
			return updateStatusAndRequeue()
		}
	case rtbootstrapper.StatusReady:
		s.instance.UpdateStatePending(
			imv1.ConditionTypeRuntimeBootstrapperReady,
			imv1.ConditionReasonRuntimeBootstrapperConfigured,
			"True",
			msgInstallationCompleted,
		)
	}

	return switchState(sFnFinalizeRegistryCache)
}
