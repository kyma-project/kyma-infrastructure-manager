package fsm

import (
	"context"
	"errors"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/internal/rtbootstrapper"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"testing"
	"time"
)

func Test_sFnInitializeRuntimeBootstrapper_Disabled(t *testing.T) {
	// given
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	scheme := runtime.NewScheme()
	utilruntime.Must(imv1.AddToScheme(scheme))

	f := &fsm{
		RCCfg: RCCfg{
			RuntimeBootstrapperEnabled:   false,
			RuntimeBootstrapperInstaller: nil,
		},
	}

	ss := &systemState{instance: minimalRuntime()}

	// when
	next, res, err := sFnInitializeRuntimeBootstrapper(ctx, f, ss)

	// then
	require.NoError(t, err)
	require.Nil(t, res)
	require.NotNil(t, next)
	require.Contains(t, next.name(), "sFnFinalizeRegistryCache")
}

func Test_sFnInitializeRuntimeBootstrapper_Ready(t *testing.T) {
	// given
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	scheme := runtime.NewScheme()
	utilruntime.Must(imv1.AddToScheme(scheme))

	inst := NewMockRuntimeBootstrapperInstaller(t)
	inst.EXPECT().Status(mockCtx(), "test-runtime").Return(rtbootstrapper.StatusReady, nil)

	f := &fsm{
		RCCfg: RCCfg{
			RuntimeBootstrapperEnabled:   true,
			RuntimeBootstrapperInstaller: inst,
		},
	}

	ss := &systemState{instance: minimalRuntime()}

	expectedRuntimeConditions := []metav1.Condition{
		{
			Type:    string(imv1.ConditionTypeRuntimeBootstrapperReady),
			Reason:  string(imv1.ConditionReasonRuntimeBootstrapperConfigured),
			Status:  "True",
			Message: "Runtime bootstrapper installation completed",
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

func Test_sFnInitializeRuntimeBootstrapper_StatusError(t *testing.T) {
	// given
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	scheme := runtime.NewScheme()
	utilruntime.Must(imv1.AddToScheme(scheme))

	inst := NewMockRuntimeBootstrapperInstaller(t)
	inst.EXPECT().Status(mockCtx(), "test-runtime").Return(rtbootstrapper.InstallationStatus(0), errors.New("status failed"))

	f := &fsm{
		RCCfg: RCCfg{
			RuntimeBootstrapperEnabled:   true,
			RuntimeBootstrapperInstaller: inst,
		},
	}

	ss := &systemState{instance: minimalRuntime()}

	// when
	next, res, err := sFnInitializeRuntimeBootstrapper(ctx, f, ss)

	// then
	require.NoError(t, err)
	require.Nil(t, res)
	require.NotNil(t, next)
	require.Contains(t, next.name(), "sFnUpdateStatus")
}

func Test_sFnInitializeRuntimeBootstrapper_InstallError(t *testing.T) {
	// given
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	scheme := runtime.NewScheme()
	utilruntime.Must(imv1.AddToScheme(scheme))

	inst := NewMockRuntimeBootstrapperInstaller(t)
	inst.EXPECT().Status(mockCtx(), "test-runtime").Return(rtbootstrapper.StatusNotStarted, nil)
	inst.EXPECT().Install(mockCtx(), "test-runtime").Return(errors.New("install failed"))

	f := &fsm{
		RCCfg: RCCfg{
			RuntimeBootstrapperEnabled:   true,
			RuntimeBootstrapperInstaller: inst,
		},
	}

	ss := &systemState{instance: minimalRuntime()}

	// when
	next, res, err := sFnInitializeRuntimeBootstrapper(ctx, f, ss)

	// then
	require.NoError(t, err)
	require.Nil(t, res)
	require.NotNil(t, next)
	require.Contains(t, next.name(), "sFnUpdateStatus")
}

func Test_sFnInitializeRuntimeBootstrapper_FailedStatus(t *testing.T) {
	// given
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	scheme := runtime.NewScheme()
	utilruntime.Must(imv1.AddToScheme(scheme))

	inst := NewMockRuntimeBootstrapperInstaller(t)
	inst.EXPECT().Status(mockCtx(), "test-runtime").Return(rtbootstrapper.StatusFailed, nil)

	f := &fsm{
		RCCfg: RCCfg{
			RuntimeBootstrapperEnabled:   true,
			RuntimeBootstrapperInstaller: inst,
		},
	}

	ss := &systemState{instance: minimalRuntime()}

	// when
	next, res, err := sFnInitializeRuntimeBootstrapper(ctx, f, ss)

	// then
	require.NoError(t, err)
	require.Nil(t, res)
	require.NotNil(t, next)
	require.Contains(t, next.name(), "sFnUpdateStatus")
}

func Test_sFnInitializeRuntimeBootstrapper_InProgress(t *testing.T) {
	// given
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	scheme := runtime.NewScheme()
	utilruntime.Must(imv1.AddToScheme(scheme))

	inst := NewMockRuntimeBootstrapperInstaller(t)
	inst.EXPECT().Status(mockCtx(), "test-runtime").Return(rtbootstrapper.StatusInProgress, nil)

	f := &fsm{
		RCCfg: RCCfg{
			RuntimeBootstrapperEnabled:   true,
			RuntimeBootstrapperInstaller: inst,
		},
	}

	ss := &systemState{instance: minimalRuntime()}

	// when
	next, res, err := sFnInitializeRuntimeBootstrapper(ctx, f, ss)

	// then
	require.NoError(t, err)
	require.Nil(t, res)
	require.NotNil(t, next)
	require.Contains(t, next.name(), "sFnUpdateStatus")
}

func minimalRuntime() imv1.Runtime {
	return imv1.Runtime{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-runtime",
			Namespace: "kcp-system",
		},
	}
}

// mockCtx is a loose matcher for context.Context used in mock expectations.
func mockCtx() interface{} { return mock.Anything }
