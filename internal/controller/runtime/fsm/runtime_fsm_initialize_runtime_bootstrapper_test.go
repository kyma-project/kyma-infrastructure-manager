package fsm

import (
	"context"
	"errors"
	"testing"
	"time"

	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/internal/rtbootstrapper"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_sFnInitializeRuntimeBootstrapper_Disabled(t *testing.T) {
	// given
	f := &fsm{
		RCCfg: RCCfg{
			RuntimeBootstrapperEnabled:   false,
			RuntimeBootstrapperInstaller: nil,
		},
	}

	ss := &systemState{instance: minimalRuntime()}

	// when
	next, res, err := sFnInitializeRuntimeBootstrapper(context.Background(), f, ss)

	// then
	require.NoError(t, err)
	require.Nil(t, res)
	require.NotNil(t, next)
	require.Contains(t, next.name(), "sFnFinalizeRegistryCache")
}

func Test_sFnInitializeRuntimeBootstrapper_Ready(t *testing.T) {
	// given
	runtime := minimalRuntime()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	inst := NewMockRuntimeBootstrapperInstaller(t)
	inst.EXPECT().Configure(mock.Anything, runtime).Return(nil)
	inst.EXPECT().Status(mock.Anything, runtime).Return(rtbootstrapper.StatusReady, nil)
	inst.EXPECT().Cleanup(mock.Anything, runtime).Return(errors.New("cleanup error"))

	f := &fsm{
		RCCfg: RCCfg{
			RuntimeBootstrapperEnabled:   true,
			RuntimeBootstrapperInstaller: inst,
		},
	}

	ss := &systemState{instance: runtime}

	expectedRuntimeConditions := []metav1.Condition{
		{
			Type:    string(imv1.ConditionTypeRuntimeBootstrapperReady),
			Reason:  string(imv1.ConditionReasonRuntimeBootstrapperConfigured),
			Status:  "True",
			Message: msgInstallationCompleted,
		},
	}

	// when
	next, res, err := sFnInitializeRuntimeBootstrapper(ctx, f, ss)

	// then
	require.NoError(t, err)
	require.Nil(t, res)
	require.NotNil(t, next)
	require.Contains(t, next.name(), "sFnFinalizeRegistryCache")
	assertEqualConditions(t, expectedRuntimeConditions, ss.instance.Status.Conditions)
}

func Test_sFnInitializeRuntimeBootstrapper_Errors(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	cases := []struct {
		name                 string
		mockSetup            func(inst *MockRuntimeBootstrapperInstaller, runtime imv1.Runtime)
		expectedNextContains string
		expectedCondition    metav1.Condition
	}{
		{
			name: "ConfigError",
			mockSetup: func(inst *MockRuntimeBootstrapperInstaller, runtime imv1.Runtime) {
				inst.EXPECT().Configure(mock.Anything, runtime).Return(errors.New("config error"))
			},
			expectedCondition: metav1.Condition{
				Type:    string(imv1.ConditionTypeRuntimeBootstrapperReady),
				Reason:  string(imv1.ConditionReasonRuntimeBootstrapperConfigurationFailed),
				Status:  "False",
				Message: msgConfigurationFailed,
			},
		},
		{
			name: "StatusError",
			mockSetup: func(inst *MockRuntimeBootstrapperInstaller, runtime imv1.Runtime) {
				inst.EXPECT().Configure(mock.Anything, runtime).Return(nil)
				inst.EXPECT().Status(mock.Anything, runtime).Return(rtbootstrapper.StatusFailed, errors.New("status failed"))
			},
			expectedCondition: metav1.Condition{
				Type:    string(imv1.ConditionTypeRuntimeBootstrapperReady),
				Reason:  string(imv1.ConditionReasonRuntimeBootstrapperStatusUnknown),
				Status:  "False",
				Message: msgStatusCheckFailed,
			},
		},
		{
			name: "InstallError",
			mockSetup: func(inst *MockRuntimeBootstrapperInstaller, runtime imv1.Runtime) {
				inst.EXPECT().Configure(mock.Anything, runtime).Return(nil)
				inst.EXPECT().Status(mock.Anything, runtime).Return(rtbootstrapper.StatusNotStarted, nil)
				inst.EXPECT().Install(mock.Anything, runtime).Return(errors.New("install failed"))
			},
			expectedCondition: metav1.Condition{
				Type:    string(imv1.ConditionTypeRuntimeBootstrapperReady),
				Reason:  string(imv1.ConditionReasonRuntimeBootstrapperInstallationFailed),
				Status:  "False",
				Message: msgInstallationFailed,
			},
		},
		{
			name: "FailedStatus",
			mockSetup: func(inst *MockRuntimeBootstrapperInstaller, runtime imv1.Runtime) {
				inst.EXPECT().Configure(mock.Anything, runtime).Return(nil)
				inst.EXPECT().Status(mock.Anything, runtime).Return(rtbootstrapper.StatusFailed, nil)
			},
			expectedCondition: metav1.Condition{
				Type:    string(imv1.ConditionTypeRuntimeBootstrapperReady),
				Reason:  string(imv1.ConditionReasonRuntimeBootstrapperInstallationFailed),
				Status:  "False",
				Message: msgInstallationFailed,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			inst := NewMockRuntimeBootstrapperInstaller(t)
			runtime := minimalRuntime()

			tc.mockSetup(inst, runtime)

			f := &fsm{
				RCCfg: RCCfg{
					RuntimeBootstrapperEnabled:   true,
					RuntimeBootstrapperInstaller: inst,
				},
			}
			ss := &systemState{instance: runtime}

			next, res, err := sFnInitializeRuntimeBootstrapper(ctx, f, ss)

			require.NoError(t, err)
			require.Nil(t, res)
			require.NotNil(t, next)
			require.Contains(t, next.name(), "sFnUpdateStatus")
			assertEqualConditions(t, []metav1.Condition{tc.expectedCondition}, ss.instance.Status.Conditions)
		})
	}
}

func Test_sFnInitializeRuntimeBootstrapper_InProgress(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	newFSMWith := func(inst *MockRuntimeBootstrapperInstaller) *fsm {
		return &fsm{
			RCCfg: RCCfg{
				RuntimeBootstrapperEnabled:   true,
				RuntimeBootstrapperInstaller: inst,
			},
		}
	}

	expectedRuntimeConditionsInstallation := []metav1.Condition{
		{
			Type:    string(imv1.ConditionTypeRuntimeBootstrapperReady),
			Reason:  string(imv1.ConditionReasonRuntimeBootstrapperInstallationInProgress),
			Status:  "False",
			Message: msgInstallationInProgress,
		},
	}

	expectedRuntimeConditionsUpgrade := []metav1.Condition{
		{
			Type:    string(imv1.ConditionTypeRuntimeBootstrapperReady),
			Reason:  string(imv1.ConditionReasonRuntimeBootstrapperUpgradeInProgress),
			Status:  "False",
			Message: msgUpgradeInProgress,
		},
	}

	t.Run("StatusNotStarted", func(t *testing.T) {
		inst := NewMockRuntimeBootstrapperInstaller(t)
		runtime := minimalRuntime()

		inst.EXPECT().Configure(mock.Anything, runtime).Return(nil)
		inst.EXPECT().Status(mock.Anything, runtime).Return(rtbootstrapper.StatusNotStarted, nil)
		inst.EXPECT().Install(mock.Anything, runtime).Return(nil)

		f := newFSMWith(inst)
		ss := &systemState{instance: runtime}

		next, res, err := sFnInitializeRuntimeBootstrapper(ctx, f, ss)

		require.NoError(t, err)
		require.Nil(t, res)
		require.NotNil(t, next)
		require.Contains(t, next.name(), "sFnUpdateStatus")

		assertEqualConditions(t, expectedRuntimeConditionsInstallation, ss.instance.Status.Conditions)
	})

	t.Run("StatusUpgradeNeeded", func(t *testing.T) {
		inst := NewMockRuntimeBootstrapperInstaller(t)
		runtime := minimalRuntime()

		inst.EXPECT().Configure(mock.Anything, runtime).Return(nil)
		inst.EXPECT().Status(mock.Anything, runtime).Return(rtbootstrapper.StatusUpgradeNeeded, nil)
		inst.EXPECT().Install(mock.Anything, runtime).Return(nil)

		f := newFSMWith(inst)
		ss := &systemState{instance: runtime}

		next, res, err := sFnInitializeRuntimeBootstrapper(ctx, f, ss)

		require.NoError(t, err)
		require.Nil(t, res)
		require.NotNil(t, next)
		require.Contains(t, next.name(), "sFnUpdateStatus")

		assertEqualConditions(t, expectedRuntimeConditionsUpgrade, ss.instance.Status.Conditions)
	})

	t.Run("StatusInProgress", func(t *testing.T) {
		inst := NewMockRuntimeBootstrapperInstaller(t)
		runtime := minimalRuntime()

		inst.EXPECT().Configure(mock.Anything, runtime).Return(nil)
		inst.EXPECT().Status(mock.Anything, runtime).Return(rtbootstrapper.StatusInProgress, nil)

		f := newFSMWith(inst)
		ss := &systemState{instance: minimalRuntime()}

		next, res, err := sFnInitializeRuntimeBootstrapper(ctx, f, ss)

		require.NoError(t, err)
		require.Nil(t, res)
		require.NotNil(t, next)
		require.Contains(t, next.name(), "sFnUpdateStatus")

		assertEqualConditions(t, expectedRuntimeConditionsInstallation, ss.instance.Status.Conditions)
	})
}

func minimalRuntime() imv1.Runtime {
	return imv1.Runtime{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-runtime",
			Namespace: "kcp-system",
		},
	}
}
