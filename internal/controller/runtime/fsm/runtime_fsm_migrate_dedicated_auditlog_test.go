package fsm

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/internal/controller/metrics/mocks"
	"github.com/kyma-project/infrastructure-manager/pkg/auditlog"
	auditlogmocks "github.com/kyma-project/infrastructure-manager/pkg/auditlog/mocks"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/extender/extensions"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/autoscaling/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestSFnMigrateToDedicatedAuditLog(t *testing.T) {
	ctx := context.Background()
	runtimeID := "test-runtime-id"
	auditLogAccessEnabled := true

	t.Run("should complete provisioning when dedicated audit logging is disabled globally", func(t *testing.T) {
		// given
		instance := &imv1.Runtime{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-runtime",
				Labels: map[string]string{
					imv1.LabelKymaRuntimeID: runtimeID,
				},
			},
			Spec: imv1.RuntimeSpec{
				AuditLogAccessEnabled: &auditLogAccessEnabled,
			},
		}

		shoot := &gardener.Shoot{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-shoot",
				Namespace: "garden-test",
			},
		}

		mockMetrics := &mocks.Metrics{}
		scheme, _ := newCreateTestScheme()
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(shoot).Build()

		testFsm := &fsm{
			K8s: K8s{
				GardenClient: fakeClient,
			},
			RCCfg: RCCfg{
				DedicatedAuditLoggingEnabled: false, // feature disabled
				Metrics:                      mockMetrics,
			},
		}

		systemState := &systemState{
			instance: *instance,
			shoot:    shoot,
		}

		// when
		stateFn, result, err := sFnMigrateToDedicatedAuditLog(ctx, testFsm, systemState)

		// then
		require.NoError(t, err)
		require.Nil(t, result)
		require.Contains(t, stateFn.name(), "updateStatusAndStop")
		require.True(t, systemState.instance.IsProvisioningCompletedStatusSet())
	})

	t.Run("should complete provisioning when audit log access is not enabled for runtime", func(t *testing.T) {
		// given
		auditLogAccessDisabled := false
		instance := &imv1.Runtime{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-runtime",
				Labels: map[string]string{
					imv1.LabelKymaRuntimeID: runtimeID,
				},
			},
			Spec: imv1.RuntimeSpec{
				AuditLogAccessEnabled: &auditLogAccessDisabled,
			},
		}

		shoot := &gardener.Shoot{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-shoot",
				Namespace: "garden-test",
			},
		}

		mockMetrics := &mocks.Metrics{}
		scheme, _ := newCreateTestScheme()
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(shoot).Build()

		testFsm := &fsm{
			K8s: K8s{
				GardenClient: fakeClient,
			},
			RCCfg: RCCfg{
				DedicatedAuditLoggingEnabled: true,
				Metrics:                      mockMetrics,
			},
		}

		systemState := &systemState{
			instance: *instance,
			shoot:    shoot,
		}

		// when
		stateFn, result, err := sFnMigrateToDedicatedAuditLog(ctx, testFsm, systemState)

		// then
		require.NoError(t, err)
		require.Nil(t, result)
		require.Contains(t, stateFn.name(), "updateStatusAndStop")
		require.True(t, systemState.instance.IsProvisioningCompletedStatusSet())
	})

	t.Run("should fail when unable to claim dedicated audit log", func(t *testing.T) {
		// given
		instance := &imv1.Runtime{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-runtime",
				Labels: map[string]string{
					imv1.LabelKymaRuntimeID: runtimeID,
				},
			},
			Spec: imv1.RuntimeSpec{
				AuditLogAccessEnabled: &auditLogAccessEnabled,
			},
		}

		shoot := &gardener.Shoot{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-shoot",
				Namespace: "garden-test",
			},
		}

		mockAuditLogProvider := &auditlogmocks.DataProvider{}
		mockAuditLogProvider.On("ClaimDedicatedAuditLogData", ctx, runtimeID).
			Return(auditlog.AuditLogData{}, errors.New("failed to claim audit log"))

		mockMetrics := &mocks.Metrics{}
		mockMetrics.On("IncRuntimeFSMStopCounter").Return()

		scheme, _ := newCreateTestScheme()
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(shoot).Build()

		testFsm := &fsm{
			K8s: K8s{
				GardenClient: fakeClient,
			},
			RCCfg: RCCfg{
				DedicatedAuditLoggingEnabled: true,
				AuditLogDataProvider:         mockAuditLogProvider,
				Metrics:                      mockMetrics,
			},
		}

		systemState := &systemState{
			instance: *instance,
			shoot:    shoot,
		}

		// when
		stateFn, result, err := sFnMigrateToDedicatedAuditLog(ctx, testFsm, systemState)

		// then
		require.NoError(t, err)
		require.Nil(t, result)
		require.Contains(t, stateFn.name(), "updateStatusAndStop")

		// Verify instance status was updated with error
		condition := meta.FindStatusCondition(systemState.instance.Status.Conditions, string(imv1.ConditionTypeCustomAuditLogConfigured))
		require.NotNil(t, condition)
		require.Equal(t, metav1.ConditionFalse, condition.Status)
		require.Equal(t, string(imv1.ConditionReasonCustomAuditLogError), condition.Reason)

		mockMetrics.AssertExpectations(t)
		mockAuditLogProvider.AssertExpectations(t)
	})

	t.Run("should complete provisioning when shoot already has correct dedicated config", func(t *testing.T) {
		// given
		desiredAuditLogData := auditlog.AuditLogData{
			TenantID:   "test-tenant-id",
			ServiceURL: "https://test.auditlog.example.com",
			SecretName: "test-secret",
		}

		// Create shoot with matching audit log configuration
		config := extensions.AuditlogExtensionConfig{
			Type:                "standard",
			TenantID:            desiredAuditLogData.TenantID,
			ServiceURL:          desiredAuditLogData.ServiceURL,
			SecretReferenceName: dedicatedAuditlogSecretReference,
		}
		configJSON, _ := json.Marshal(config)

		shoot := &gardener.Shoot{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-shoot",
				Namespace: "garden-test",
			},
			Spec: gardener.ShootSpec{
				Extensions: []gardener.Extension{
					{
						Type: extensions.AuditlogExtensionType,
						ProviderConfig: &runtime.RawExtension{
							Raw: configJSON,
						},
					},
				},
				Resources: []gardener.NamedResourceReference{
					{
						Name: dedicatedAuditlogSecretReference,
						ResourceRef: v1.CrossVersionObjectReference{
							Name:       desiredAuditLogData.SecretName,
							Kind:       "Secret",
							APIVersion: "v1",
						},
					},
				},
			},
		}

		instance := &imv1.Runtime{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-runtime",
				Labels: map[string]string{
					imv1.LabelKymaRuntimeID: runtimeID,
				},
			},
			Spec: imv1.RuntimeSpec{
				AuditLogAccessEnabled: &auditLogAccessEnabled,
			},
		}

		mockAuditLogProvider := &auditlogmocks.DataProvider{}
		mockAuditLogProvider.On("ClaimDedicatedAuditLogData", ctx, runtimeID).
			Return(desiredAuditLogData, nil)

		mockMetrics := &mocks.Metrics{}
		scheme, _ := newCreateTestScheme()
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(shoot).Build()

		testFsm := &fsm{
			K8s: K8s{
				GardenClient: fakeClient,
			},
			RCCfg: RCCfg{
				DedicatedAuditLoggingEnabled: true,
				AuditLogDataProvider:         mockAuditLogProvider,
				Metrics:                      mockMetrics,
			},
		}

		systemState := &systemState{
			instance: *instance,
			shoot:    shoot,
		}

		// when
		stateFn, result, err := sFnMigrateToDedicatedAuditLog(ctx, testFsm, systemState)

		// then
		require.NoError(t, err)
		require.Nil(t, result)
		require.Contains(t, stateFn.name(), "updateStatusAndStop")
		require.True(t, systemState.instance.IsProvisioningCompletedStatusSet())

		// Verify condition is set to ready (true status)
		condition := meta.FindStatusCondition(systemState.instance.Status.Conditions, string(imv1.ConditionTypeCustomAuditLogConfigured))
		require.NotNil(t, condition)
		require.Equal(t, metav1.ConditionTrue, condition.Status)

		mockAuditLogProvider.AssertExpectations(t)
	})

	t.Run("should requeue when shoot patch fails", func(t *testing.T) {
		// given
		desiredAuditLogData := auditlog.AuditLogData{
			TenantID:   "new-tenant-id",
			ServiceURL: "https://new.auditlog.example.com",
			SecretName: "new-secret",
		}

		// Create shoot with different audit log configuration (will trigger patch)
		oldConfig := extensions.AuditlogExtensionConfig{
			Type:                "standard",
			TenantID:            "old-tenant-id",
			ServiceURL:          "https://old.auditlog.example.com",
			SecretReferenceName: "auditlog-credentials",
		}
		configJSON, _ := json.Marshal(oldConfig)

		shoot := &gardener.Shoot{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-shoot",
				Namespace: "garden-test",
			},
			Spec: gardener.ShootSpec{
				Extensions: []gardener.Extension{
					{
						Type: extensions.AuditlogExtensionType,
						ProviderConfig: &runtime.RawExtension{
							Raw: configJSON,
						},
					},
				},
				Resources: []gardener.NamedResourceReference{
					{
						Name: "auditlog-credentials",
						ResourceRef: v1.CrossVersionObjectReference{
							Name:       "old-secret",
							Kind:       "Secret",
							APIVersion: "v1",
						},
					},
				},
			},
		}

		instance := &imv1.Runtime{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-runtime",
				Labels: map[string]string{
					imv1.LabelKymaRuntimeID: runtimeID,
				},
			},
			Spec: imv1.RuntimeSpec{
				AuditLogAccessEnabled: &auditLogAccessEnabled,
			},
		}

		mockAuditLogProvider := &auditlogmocks.DataProvider{}
		mockAuditLogProvider.On("ClaimDedicatedAuditLogData", ctx, runtimeID).
			Return(desiredAuditLogData, nil)

		mockMetrics := &mocks.Metrics{}
		scheme, _ := newCreateTestScheme()
		// Use fake client without the shoot to simulate patch failure
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

		requeueDuration := 5 * time.Second
		testFsm := &fsm{
			K8s: K8s{
				GardenClient: fakeClient,
			},
			RCCfg: RCCfg{
				DedicatedAuditLoggingEnabled: true,
				AuditLogDataProvider:         mockAuditLogProvider,
				Metrics:                      mockMetrics,
				GardenerRequeueDuration:      requeueDuration,
			},
		}

		systemState := &systemState{
			instance: *instance,
			shoot:    shoot,
		}

		// when
		stateFn, result, err := sFnMigrateToDedicatedAuditLog(ctx, testFsm, systemState)

		// then
		require.NoError(t, err)
		require.Nil(t, result) // result is nil, returns sFnUpdateStatus
		require.Contains(t, stateFn.name(), "sFnUpdateStatus")

		mockAuditLogProvider.AssertExpectations(t)
	})

	t.Run("should requeue when shoot patch succeeds", func(t *testing.T) {
		// given
		desiredAuditLogData := auditlog.AuditLogData{
			TenantID:   "new-tenant-id",
			ServiceURL: "https://new.auditlog.example.com",
			SecretName: "new-secret",
		}

		// Create shoot with different audit log configuration
		oldConfig := extensions.AuditlogExtensionConfig{
			Type:                "standard",
			TenantID:            "old-tenant-id",
			ServiceURL:          "https://old.auditlog.example.com",
			SecretReferenceName: "auditlog-credentials",
		}
		configJSON, _ := json.Marshal(oldConfig)

		shoot := &gardener.Shoot{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-shoot",
				Namespace: "garden-test",
			},
			Spec: gardener.ShootSpec{
				Extensions: []gardener.Extension{
					{
						Type: extensions.AuditlogExtensionType,
						ProviderConfig: &runtime.RawExtension{
							Raw: configJSON,
						},
					},
				},
				Resources: []gardener.NamedResourceReference{
					{
						Name: "auditlog-credentials",
						ResourceRef: v1.CrossVersionObjectReference{
							Name:       "old-secret",
							Kind:       "Secret",
							APIVersion: "v1",
						},
					},
				},
			},
		}

		instance := &imv1.Runtime{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-runtime",
				Labels: map[string]string{
					imv1.LabelKymaRuntimeID: runtimeID,
				},
			},
			Spec: imv1.RuntimeSpec{
				AuditLogAccessEnabled: &auditLogAccessEnabled,
			},
		}

		mockAuditLogProvider := &auditlogmocks.DataProvider{}
		mockAuditLogProvider.On("ClaimDedicatedAuditLogData", ctx, runtimeID).
			Return(desiredAuditLogData, nil)

		mockMetrics := &mocks.Metrics{}
		scheme, _ := newCreateTestScheme()
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(shoot).Build()

		requeueDuration := 5 * time.Second
		testFsm := &fsm{
			K8s: K8s{
				GardenClient: fakeClient,
			},
			RCCfg: RCCfg{
				DedicatedAuditLoggingEnabled: true,
				AuditLogDataProvider:         mockAuditLogProvider,
				Metrics:                      mockMetrics,
				GardenerRequeueDuration:      requeueDuration,
			},
		}

		systemState := &systemState{
			instance: *instance,
			shoot:    shoot,
		}

		// when
		stateFn, result, err := sFnMigrateToDedicatedAuditLog(ctx, testFsm, systemState)

		// then
		require.NoError(t, err)
		require.Nil(t, result) // result is nil, returns sFnUpdateStatus
		require.Contains(t, stateFn.name(), "sFnUpdateStatus")

		mockAuditLogProvider.AssertExpectations(t)
	})

	t.Run("should handle nil AuditLogAccessEnabled gracefully", func(t *testing.T) {
		// given
		instance := &imv1.Runtime{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-runtime",
				Labels: map[string]string{
					imv1.LabelKymaRuntimeID: runtimeID,
				},
			},
			Spec: imv1.RuntimeSpec{
				AuditLogAccessEnabled: nil, // nil should be treated as disabled
			},
		}

		shoot := &gardener.Shoot{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-shoot",
				Namespace: "garden-test",
			},
		}

		mockMetrics := &mocks.Metrics{}
		scheme, _ := newCreateTestScheme()
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(shoot).Build()

		testFsm := &fsm{
			K8s: K8s{
				GardenClient: fakeClient,
			},
			RCCfg: RCCfg{
				DedicatedAuditLoggingEnabled: true,
				Metrics:                      mockMetrics,
			},
		}

		systemState := &systemState{
			instance: *instance,
			shoot:    shoot,
		}

		// when
		stateFn, result, err := sFnMigrateToDedicatedAuditLog(ctx, testFsm, systemState)

		// then
		require.NoError(t, err)
		require.Nil(t, result)
		require.Contains(t, stateFn.name(), "updateStatusAndStop")
		require.True(t, systemState.instance.IsProvisioningCompletedStatusSet())
	})
}

func TestUpdateStatusAndRequeueAfter(t *testing.T) {
	t.Run("should return update status state function", func(t *testing.T) {
		// given
		duration := 5 * time.Second

		// when
		stateFn, result, err := updateStatusAndRequeueAfter(duration)

		// then
		require.NoError(t, err)
		require.Nil(t, result) // result is nil, actual result handled by sFnUpdateStatus
		require.NotNil(t, stateFn)
		require.Contains(t, stateFn.name(), "sFnUpdateStatus")
	})
}
