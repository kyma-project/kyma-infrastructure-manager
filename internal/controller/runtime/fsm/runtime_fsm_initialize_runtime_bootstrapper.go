package fsm

import (
	"context"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/internal/log_level"
	"github.com/kyma-project/infrastructure-manager/internal/rtbootstrapper"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"time"
)

const (
	msgConfigurationFailed    = "Failed to copy configuration to the runtime cluster"
	msgStatusCheckFailed      = "Runtime bootstrapper status check failed"
	msgInstallationFailed     = "Runtime bootstrapper installation failed"
	msgUpgradeFailed          = "Runtime bootstrapper upgrade failed"
	msgInstallationInProgress = "Runtime bootstrapper installation in progress"
	msgUpgradeInProgress      = "Runtime bootstrapper upgrade in progress"
	msgInstallationCompleted  = "Runtime bootstrapper installation completed"
	timeout                   = time.Second * 30
)

func sFnInitializeRuntimeBootstrapper(ctx context.Context, m *fsm, s *systemState) (stateFn, *ctrl.Result, error) {
	if !m.RuntimeBootstrapperEnabled || m.RuntimeBootstrapperInstaller == nil {
		m.log.V(log_level.DEBUG).Info("Runtime bootstrapper installation is disabled")
		return switchState(sFnFinalizeRegistryCache)
	}

	err := m.RuntimeBootstrapperInstaller.Configure(ctx, s.instance)
	if err != nil {
		m.log.Error(err, "Failed to configure runtime bootstrapper")
		s.instance.UpdateStatePending(
			imv1.ConditionTypeRuntimeBootstrapperReady,
			imv1.ConditionReasonRuntimeBootstrapperConfigurationFailed,
			metav1.ConditionFalse,
			msgConfigurationFailed,
		)
		return updateStatusAndRequeueAfter(timeout)
	}

	status, err := m.RuntimeBootstrapperInstaller.Status(ctx, s.instance)
	if err != nil {
		m.log.Error(err, "Failed to get runtime bootstrapper installation status")
		s.instance.UpdateStatePending(
			imv1.ConditionTypeRuntimeBootstrapperReady,
			imv1.ConditionReasonRuntimeBootstrapperStatusUnknown,
			metav1.ConditionFalse,
			msgStatusCheckFailed,
		)
		return updateStatusAndRequeueAfter(timeout)
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
					metav1.ConditionFalse,
					msgInstallationFailed,
				)
			} else {
				s.instance.UpdateStatePending(
					imv1.ConditionTypeRuntimeBootstrapperReady,
					imv1.ConditionReasonRuntimeBootstrapperInstallationInProgress,
					metav1.ConditionFalse,
					msgInstallationInProgress,
				)
			}

			return updateStatusAndRequeueAfter(timeout)
		}
	case rtbootstrapper.StatusUpgradeNeeded:
		{
			err := m.RuntimeBootstrapperInstaller.Install(ctx, s.instance)
			if err != nil {
				m.log.Error(err, "Failed to start runtime bootstrapper upgrade")
				s.instance.UpdateStatePending(
					imv1.ConditionTypeRuntimeBootstrapperReady,
					imv1.ConditionReasonRuntimeBootstrapperUpgradeFailed,
					metav1.ConditionFalse,
					msgUpgradeFailed,
				)
			} else {
				s.instance.UpdateStatePending(
					imv1.ConditionTypeRuntimeBootstrapperReady,
					imv1.ConditionReasonRuntimeBootstrapperUpgradeInProgress,
					metav1.ConditionFalse,
					msgUpgradeInProgress,
				)
			}

			return updateStatusAndRequeueAfter(timeout)
		}
	case rtbootstrapper.StatusInProgress:
		{
			m.log.V(log_level.DEBUG).Info("Runtime bootstrapper installation in progress")
			s.instance.UpdateStatePending(
				imv1.ConditionTypeRuntimeBootstrapperReady,
				imv1.ConditionReasonRuntimeBootstrapperInstallationInProgress,
				metav1.ConditionFalse,
				msgInstallationInProgress,
			)

			return updateStatusAndRequeueAfter(timeout)
		}
	case rtbootstrapper.StatusFailed:
		{
			m.log.Error(err, "Runtime bootstrapper installation failed")
			return updateStateFailedWithErrorAndStop(
				&s.instance,
				imv1.ConditionTypeRuntimeBootstrapperReady,
				imv1.ConditionReasonRuntimeBootstrapperInstallationFailed,
				msgInstallationFailed)
		}
	case rtbootstrapper.StatusReady:
		err := m.RuntimeBootstrapperInstaller.Cleanup(ctx, s.instance)
		if err != nil {
			m.log.Error(err, "Failed to cleanup after runtime bootstrapper installation")
		}

		s.instance.UpdateStatePending(
			imv1.ConditionTypeRuntimeBootstrapperReady,
			imv1.ConditionReasonRuntimeBootstrapperConfigured,
			metav1.ConditionTrue,
			msgInstallationCompleted,
		)
	}

	return switchState(sFnFinalizeRegistryCache)
}
