package fsm

import (
	"context"
	"github.com/kyma-project/infrastructure-manager/internal/auditlogging"
	"testing"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	v1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/internal/auditlogging/mocks"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAuditLogState(t *testing.T) {
	t.Run("Should set status on Runtime CR when Audit Log configuration was changed and Shoot enters into reconciliation", func(t *testing.T) {
		// given
		ctx := context.Background()
		auditLog := &mocks.AuditLogging{}
		shoot := shootForTest()
		instance := runtimeForTest()
		systemState := &systemState{
			instance: instance,
			shoot:    shoot,
		}
		expectedRuntimeConditions := []metav1.Condition{
			{
				Type:    string(v1.ConditionTypeAuditLogConfigured),
				Status:  "Unknown",
				Reason:  string(v1.ConditionReasonAuditLogConfigured),
				Message: "Waiting for Gardener shoot to be Ready state after configuration of the Audit Logs",
			},
		}

		fsm := &fsm{AuditLogging: auditLog}
		fsm.RCCfg.AuditLogMandatory = true

		auditLog.On("Enable", ctx, shoot).Return(true, nil).Once()

		// when
		stateFn, _, _ := sFnConfigureAuditLog(ctx, fsm, systemState)

		// set the time to its zero value for comparison purposes
		systemState.instance.Status.Conditions[0].LastTransitionTime = metav1.Time{}

		// then
		auditLog.AssertExpectations(t)
		require.Contains(t, stateFn.name(), "sFnUpdateStatus")
		assert.Equal(t, v1.RuntimeStatePending, string(systemState.instance.Status.State))
		assert.Equal(t, expectedRuntimeConditions, systemState.instance.Status.Conditions)
	})

	t.Run("Should set status on Runtime CR when Shoot is in Succeeded state after configuring Audit Log", func(t *testing.T) {
		// given
		ctx := context.Background()
		auditLog := &mocks.AuditLogging{}
		shoot := shootForTest()
		instance := runtimeForTest()
		systemState := &systemState{
			instance: instance,
			shoot:    shoot,
		}
		expectedRuntimeConditions := []metav1.Condition{
			{
				Type:    string(v1.ConditionTypeAuditLogConfigured),
				Status:  "True",
				Reason:  string(v1.ConditionReasonAuditLogConfigured),
				Message: "Audit Log state completed successfully",
			},
		}

		fsm := &fsm{AuditLogging: auditLog}
		fsm.RCCfg.AuditLogMandatory = true

		auditLog.On("Enable", ctx, shoot).Return(false, nil).Once()

		// when
		stateFn, _, _ := sFnConfigureAuditLog(ctx, fsm, systemState)

		// set the time to its zero value for comparison purposes
		systemState.instance.Status.Conditions[0].LastTransitionTime = metav1.Time{}

		// then
		auditLog.AssertExpectations(t)
		require.Contains(t, stateFn.name(), "sFnUpdateStatus")
		assert.Equal(t, v1.RuntimeStateReady, string(systemState.instance.Status.State))
		assert.Equal(t, expectedRuntimeConditions, systemState.instance.Status.Conditions)
	})

	t.Run("Should set status on Runtime CR when region mapping is missing and Audit Log is mandatory (hard error)", func(t *testing.T) {
		// given
		ctx := context.Background()
		auditLog := &mocks.AuditLogging{}
		shoot := shootForTest()
		instance := runtimeForTest()
		systemState := &systemState{
			instance: instance,
			shoot:    shoot,
		}
		expectedRuntimeConditions := []metav1.Condition{
			{
				Type:    string(v1.ConditionTypeAuditLogConfigured),
				Status:  "False",
				Reason:  string(v1.ConditionReasonAuditLogMissingRegionMapping),
				Message: auditlogging.ErrMissingMapping.Error(),
			},
		}

		fsm := &fsm{AuditLogging: auditLog}
		fsm.RCCfg.AuditLogMandatory = true

		auditLog.On("Enable", ctx, shoot).Return(false, auditlogging.ErrMissingMapping).Once()

		// when
		stateFn, _, _ := sFnConfigureAuditLog(ctx, fsm, systemState)

		// set the time to its zero value for comparison purposes
		systemState.instance.Status.Conditions[0].LastTransitionTime = metav1.Time{}

		// then
		auditLog.AssertExpectations(t)
		require.Contains(t, stateFn.name(), "sFnUpdateStatus")
		assert.Equal(t, v1.RuntimeStateFailed, string(systemState.instance.Status.State))
		assert.Equal(t, expectedRuntimeConditions, systemState.instance.Status.Conditions)
	})

	t.Run("Should set status on Runtime CR when region mapping is missing and Audit Log is not mandatory (proceed to next step)", func(t *testing.T) {
		// given
		ctx := context.Background()
		auditLog := &mocks.AuditLogging{}
		shoot := shootForTest()
		instance := runtimeForTest()
		systemState := &systemState{
			instance: instance,
			shoot:    shoot,
		}
		expectedRuntimeConditions := []metav1.Condition{
			{
				Type:    string(v1.ConditionTypeAuditLogConfigured),
				Status:  "True",
				Reason:  string(v1.ConditionReasonAuditLogMissingRegionMapping),
				Message: "Missing region mapping for this shoot. Audit Log is not mandatory. Skipping configuration",
			},
		}

		fsm := &fsm{AuditLogging: auditLog}
		fsm.RCCfg.AuditLogMandatory = false

		auditLog.On("Enable", ctx, shoot).Return(false, auditlogging.ErrMissingMapping).Once()

		// when
		stateFn, _, _ := sFnConfigureAuditLog(ctx, fsm, systemState)

		// set the time to its zero value for comparison purposes
		systemState.instance.Status.Conditions[0].LastTransitionTime = metav1.Time{}

		// then
		auditLog.AssertExpectations(t)
		require.Contains(t, stateFn.name(), "sFnUpdateStatus")
		assert.Equal(t, v1.RuntimeStateReady, string(systemState.instance.Status.State))
		assert.Equal(t, expectedRuntimeConditions, systemState.instance.Status.Conditions)
	})

	t.Run("Should stop in case of error during configuration if Audit Log is mandatory, and set status on Runtime CR (hard error)", func(t *testing.T) {
		// given
		ctx := context.Background()
		auditLog := &mocks.AuditLogging{}
		shoot := shootForTest()
		instance := runtimeForTest()
		systemState := &systemState{
			instance: instance,
			shoot:    shoot,
		}
		expectedRuntimeConditions := []metav1.Condition{
			{
				Type:    string(v1.ConditionTypeAuditLogConfigured),
				Status:  "False",
				Reason:  string(v1.ConditionReasonAuditLogError),
				Message: "some error during configuration",
			},
		}

		fsm := &fsm{AuditLogging: auditLog}
		fsm.RCCfg.AuditLogMandatory = true

		auditLog.On("Enable", ctx, shoot).Return(false, errors.New("some error during configuration")).Once()

		// when
		stateFn, _, _ := sFnConfigureAuditLog(ctx, fsm, systemState)

		// set the time to its zero value for comparison purposes
		systemState.instance.Status.Conditions[0].LastTransitionTime = metav1.Time{}

		// then
		auditLog.AssertExpectations(t)
		require.Contains(t, stateFn.name(), "sFnUpdateStatus")
		assert.Equal(t, v1.RuntimeStateFailed, string(systemState.instance.Status.State))
		assert.Equal(t, expectedRuntimeConditions, systemState.instance.Status.Conditions)
	})

	t.Run("Should stop in case of error during configuration if Audit Log is not mandatory, and set status on Runtime CR (proceed to next step)", func(t *testing.T) {
		// given
		ctx := context.Background()
		auditLog := &mocks.AuditLogging{}
		shoot := shootForTest()
		instance := runtimeForTest()
		systemState := &systemState{
			instance: instance,
			shoot:    shoot,
		}
		expectedRuntimeConditions := []metav1.Condition{
			{
				Type:    string(v1.ConditionTypeAuditLogConfigured),
				Status:  "True",
				Reason:  string(v1.ConditionReasonAuditLogError),
				Message: "Configuration of Audit Log is not mandatory, error for context: some error during configuration",
			},
		}

		fsm := &fsm{AuditLogging: auditLog}
		fsm.RCCfg.AuditLogMandatory = false

		auditLog.On("Enable", ctx, shoot).Return(false, errors.New("some error during configuration")).Once()

		// when
		stateFn, _, _ := sFnConfigureAuditLog(ctx, fsm, systemState)

		// set the time to its zero value for comparison purposes
		systemState.instance.Status.Conditions[0].LastTransitionTime = metav1.Time{}

		// then
		auditLog.AssertExpectations(t)
		require.Contains(t, stateFn.name(), "sFnUpdateStatus")
		assert.Equal(t, v1.RuntimeStateReady, string(systemState.instance.Status.State))
		assert.Equal(t, expectedRuntimeConditions, systemState.instance.Status.Conditions)
	})
}

func shootForTest() *gardener.Shoot {
	return &gardener.Shoot{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-shoot",
			Namespace: "namespace",
		},
		Spec: gardener.ShootSpec{
			Region:   "region",
			Provider: gardener.Provider{Type: "aws"}},
	}
}

func runtimeForTest() v1.Runtime {
	return v1.Runtime{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-runtime",
			Namespace: "namespace",
		},
		Spec: v1.RuntimeSpec{
			Shoot: v1.RuntimeShoot{
				Name:     "test-shoot",
				Region:   "region",
				Provider: v1.Provider{Type: "aws"},
			},
		},
	}
}
