package fsm

import (
	"context"
	"errors"
	"testing"
	"time"

	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/internal/controller/metrics/mocks"
	fsm_mocks "github.com/kyma-project/infrastructure-manager/internal/controller/runtime/fsm/mocks"
	"github.com/kyma-project/infrastructure-manager/pkg/auditlog"
	auditlogmocks "github.com/kyma-project/infrastructure-manager/pkg/auditlog/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestSFnCopyAuditLogReadCredentials(t *testing.T) {
	ctx := context.Background()
	runtimeID := "test-runtime-id"
	namespace := "kcp-system"

	t.Run("should copy credentials and complete provisioning", func(t *testing.T) {
		// given
		sourceSecretName := "test-read-credentials"
		sourceSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      sourceSecretName,
				Namespace: namespace,
			},
			Data: map[string][]byte{
				"clientid":     []byte("test-client-id"),
				"clientsecret": []byte("test-client-secret"),
			},
			Type: corev1.SecretTypeOpaque,
		}

		instance := &imv1.Runtime{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-runtime",
				Namespace: namespace,
				Labels: map[string]string{
					imv1.LabelKymaRuntimeID: runtimeID,
				},
			},
		}

		// Create kyma-system namespace for target secret
		kymaSystemNs := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "kyma-system",
			},
		}

		mockAuditLogProvider := &auditlogmocks.DataProvider{}
		mockAuditLogProvider.On("GetDedicatedAuditLogData", ctx, runtimeID, false).
			Return(auditlog.AuditLogData{
				TenantID:            "test-tenant",
				ServiceURL:          "https://auditlog.example.com",
				SecretName:          "gardener-secret",
				ReadCredsSecretName: sourceSecretName,
			}, nil)

		mockMetrics := &mocks.Metrics{}
		scheme, _ := newCreateTestScheme()

		// KCP client with the source secret
		kcpClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(sourceSecret).
			Build()

		// SKR client (runtime client)
		skrClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(kymaSystemNs).
			Build()

		runtimeClientGetter := &fsm_mocks.RuntimeClientGetter{}
		runtimeClientGetter.On("Get", mock.Anything, mock.Anything).Return(skrClient, nil)

		testFsm := &fsm{
			K8s: K8s{
				KcpClient:           kcpClient,
				RuntimeClientGetter: runtimeClientGetter,
			},
			RCCfg: RCCfg{
				AuditLogDataProvider: mockAuditLogProvider,
				Metrics:              mockMetrics,
			},
		}

		systemState := &systemState{
			instance: *instance,
		}

		// when
		stateFn, result, err := sFnCopyAuditLogReadCredentials(ctx, testFsm, systemState)

		// then
		require.NoError(t, err)
		require.Nil(t, result)
		require.Contains(t, stateFn.name(), "updateStatusAndStop")
		require.True(t, systemState.instance.IsProvisioningCompletedStatusSet())

		// Verify condition is set
		condition := meta.FindStatusCondition(systemState.instance.Status.Conditions, string(imv1.ConditionTypeAuditLogCredentialsCopied))
		require.NotNil(t, condition)
		require.Equal(t, metav1.ConditionTrue, condition.Status)
		require.Equal(t, string(imv1.ConditionReasonCredentialsCopied), condition.Reason)

		// Verify secret was created in SKR
		var targetSecret corev1.Secret
		err = skrClient.Get(ctx, client.ObjectKey{Name: auditLogReadCredentialsSecretName, Namespace: "kyma-system"}, &targetSecret)
		require.NoError(t, err)
		require.Equal(t, sourceSecret.Data, targetSecret.Data)
		require.Equal(t, "infrastructure-manager", targetSecret.Labels[imv1.LabelKymaManagedBy])

		mockAuditLogProvider.AssertExpectations(t)
		runtimeClientGetter.AssertExpectations(t)
	})

	t.Run("should skip copy and complete when ReadCredsSecretName is empty", func(t *testing.T) {
		// given
		instance := &imv1.Runtime{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-runtime",
				Namespace: namespace,
				Labels: map[string]string{
					imv1.LabelKymaRuntimeID: runtimeID,
				},
			},
		}

		mockAuditLogProvider := &auditlogmocks.DataProvider{}
		mockAuditLogProvider.On("GetDedicatedAuditLogData", ctx, runtimeID, false).
			Return(auditlog.AuditLogData{
				TenantID:            "test-tenant",
				ServiceURL:          "https://auditlog.example.com",
				SecretName:          "gardener-secret",
				ReadCredsSecretName: "", // empty - no read credentials
			}, nil)

		mockMetrics := &mocks.Metrics{}
		scheme, _ := newCreateTestScheme()
		kcpClient := fake.NewClientBuilder().WithScheme(scheme).Build()

		testFsm := &fsm{
			K8s: K8s{
				KcpClient: kcpClient,
			},
			RCCfg: RCCfg{
				AuditLogDataProvider: mockAuditLogProvider,
				Metrics:              mockMetrics,
			},
		}

		systemState := &systemState{
			instance: *instance,
		}

		// when
		stateFn, result, err := sFnCopyAuditLogReadCredentials(ctx, testFsm, systemState)

		// then
		require.NoError(t, err)
		require.Nil(t, result)
		require.Contains(t, stateFn.name(), "updateStatusAndStop")
		require.True(t, systemState.instance.IsProvisioningCompletedStatusSet())

		// No condition should be set for credentials copy (skipped)
		condition := meta.FindStatusCondition(systemState.instance.Status.Conditions, string(imv1.ConditionTypeAuditLogCredentialsCopied))
		require.Nil(t, condition)

		mockAuditLogProvider.AssertExpectations(t)
	})

	t.Run("should requeue on GetDedicatedAuditLogData error", func(t *testing.T) {
		// given
		instance := &imv1.Runtime{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-runtime",
				Namespace: namespace,
				Labels: map[string]string{
					imv1.LabelKymaRuntimeID: runtimeID,
				},
			},
		}

		mockAuditLogProvider := &auditlogmocks.DataProvider{}
		mockAuditLogProvider.On("GetDedicatedAuditLogData", ctx, runtimeID, false).
			Return(auditlog.AuditLogData{}, errors.New("failed to get audit log data"))

		mockMetrics := &mocks.Metrics{}
		scheme, _ := newCreateTestScheme()
		kcpClient := fake.NewClientBuilder().WithScheme(scheme).Build()

		requeueDuration := 5 * time.Second
		testFsm := &fsm{
			K8s: K8s{
				KcpClient: kcpClient,
			},
			RCCfg: RCCfg{
				AuditLogDataProvider:        mockAuditLogProvider,
				Metrics:                     mockMetrics,
				ControlPlaneRequeueDuration: requeueDuration,
			},
		}

		systemState := &systemState{
			instance: *instance,
		}

		// when
		stateFn, result, err := sFnCopyAuditLogReadCredentials(ctx, testFsm, systemState)

		// then
		require.NoError(t, err)
		require.Nil(t, result)
		require.Contains(t, stateFn.name(), "sFnUpdateStatus")

		// Verify error condition is set
		condition := meta.FindStatusCondition(systemState.instance.Status.Conditions, string(imv1.ConditionTypeAuditLogCredentialsCopied))
		require.NotNil(t, condition)
		require.Equal(t, metav1.ConditionFalse, condition.Status)
		require.Equal(t, string(imv1.ConditionReasonCredentialsCopyError), condition.Reason)

		mockAuditLogProvider.AssertExpectations(t)
	})

	t.Run("should requeue on runtime client error", func(t *testing.T) {
		// given
		sourceSecretName := "test-read-credentials"
		instance := &imv1.Runtime{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-runtime",
				Namespace: namespace,
				Labels: map[string]string{
					imv1.LabelKymaRuntimeID: runtimeID,
				},
			},
		}

		mockAuditLogProvider := &auditlogmocks.DataProvider{}
		mockAuditLogProvider.On("GetDedicatedAuditLogData", ctx, runtimeID, false).
			Return(auditlog.AuditLogData{
				TenantID:            "test-tenant",
				ServiceURL:          "https://auditlog.example.com",
				SecretName:          "gardener-secret",
				ReadCredsSecretName: sourceSecretName,
			}, nil)

		mockMetrics := &mocks.Metrics{}
		scheme, _ := newCreateTestScheme()
		kcpClient := fake.NewClientBuilder().WithScheme(scheme).Build()

		runtimeClientGetter := &fsm_mocks.RuntimeClientGetter{}
		runtimeClientGetter.On("Get", mock.Anything, mock.Anything).Return(nil, errors.New("failed to get runtime client"))

		requeueDuration := 5 * time.Second
		testFsm := &fsm{
			K8s: K8s{
				KcpClient:           kcpClient,
				RuntimeClientGetter: runtimeClientGetter,
			},
			RCCfg: RCCfg{
				AuditLogDataProvider:        mockAuditLogProvider,
				Metrics:                     mockMetrics,
				ControlPlaneRequeueDuration: requeueDuration,
			},
		}

		systemState := &systemState{
			instance: *instance,
		}

		// when
		stateFn, result, err := sFnCopyAuditLogReadCredentials(ctx, testFsm, systemState)

		// then
		require.NoError(t, err)
		require.Nil(t, result)
		require.Contains(t, stateFn.name(), "sFnUpdateStatus")

		// Verify error condition is set
		condition := meta.FindStatusCondition(systemState.instance.Status.Conditions, string(imv1.ConditionTypeAuditLogCredentialsCopied))
		require.NotNil(t, condition)
		require.Equal(t, metav1.ConditionFalse, condition.Status)
		require.Equal(t, string(imv1.ConditionReasonCredentialsCopyError), condition.Reason)

		mockAuditLogProvider.AssertExpectations(t)
		runtimeClientGetter.AssertExpectations(t)
	})

	t.Run("should requeue when source secret not found", func(t *testing.T) {
		// given
		sourceSecretName := "non-existent-secret"
		instance := &imv1.Runtime{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-runtime",
				Namespace: namespace,
				Labels: map[string]string{
					imv1.LabelKymaRuntimeID: runtimeID,
				},
			},
		}

		mockAuditLogProvider := &auditlogmocks.DataProvider{}
		mockAuditLogProvider.On("GetDedicatedAuditLogData", ctx, runtimeID, false).
			Return(auditlog.AuditLogData{
				TenantID:            "test-tenant",
				ServiceURL:          "https://auditlog.example.com",
				SecretName:          "gardener-secret",
				ReadCredsSecretName: sourceSecretName,
			}, nil)

		mockMetrics := &mocks.Metrics{}
		scheme, _ := newCreateTestScheme()

		// KCP client without the source secret
		kcpClient := fake.NewClientBuilder().WithScheme(scheme).Build()

		// SKR client
		skrClient := fake.NewClientBuilder().WithScheme(scheme).Build()

		runtimeClientGetter := &fsm_mocks.RuntimeClientGetter{}
		runtimeClientGetter.On("Get", mock.Anything, mock.Anything).Return(skrClient, nil)

		requeueDuration := 5 * time.Second
		testFsm := &fsm{
			K8s: K8s{
				KcpClient:           kcpClient,
				RuntimeClientGetter: runtimeClientGetter,
			},
			RCCfg: RCCfg{
				AuditLogDataProvider:        mockAuditLogProvider,
				Metrics:                     mockMetrics,
				ControlPlaneRequeueDuration: requeueDuration,
			},
		}

		systemState := &systemState{
			instance: *instance,
		}

		// when
		stateFn, result, err := sFnCopyAuditLogReadCredentials(ctx, testFsm, systemState)

		// then
		require.NoError(t, err)
		require.Nil(t, result)
		require.Contains(t, stateFn.name(), "sFnUpdateStatus")

		// Verify error condition is set
		condition := meta.FindStatusCondition(systemState.instance.Status.Conditions, string(imv1.ConditionTypeAuditLogCredentialsCopied))
		require.NotNil(t, condition)
		require.Equal(t, metav1.ConditionFalse, condition.Status)
		require.Equal(t, string(imv1.ConditionReasonCredentialsCopyError), condition.Reason)

		mockAuditLogProvider.AssertExpectations(t)
		runtimeClientGetter.AssertExpectations(t)
	})

	t.Run("should handle idempotent re-runs when secret already exists", func(t *testing.T) {
		// given
		sourceSecretName := "test-read-credentials"
		sourceSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      sourceSecretName,
				Namespace: namespace,
			},
			Data: map[string][]byte{
				"clientid":     []byte("updated-client-id"),
				"clientsecret": []byte("updated-client-secret"),
			},
			Type: corev1.SecretTypeOpaque,
		}

		// Existing secret in SKR with different data
		existingTargetSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      auditLogReadCredentialsSecretName,
				Namespace: "kyma-system",
				Labels: map[string]string{
					imv1.LabelKymaManagedBy: "infrastructure-manager",
				},
			},
			Data: map[string][]byte{
				"clientid":     []byte("old-client-id"),
				"clientsecret": []byte("old-client-secret"),
			},
			Type: corev1.SecretTypeOpaque,
		}

		instance := &imv1.Runtime{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-runtime",
				Namespace: namespace,
				Labels: map[string]string{
					imv1.LabelKymaRuntimeID: runtimeID,
				},
			},
		}

		kymaSystemNs := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "kyma-system",
			},
		}

		mockAuditLogProvider := &auditlogmocks.DataProvider{}
		mockAuditLogProvider.On("GetDedicatedAuditLogData", ctx, runtimeID, false).
			Return(auditlog.AuditLogData{
				TenantID:            "test-tenant",
				ServiceURL:          "https://auditlog.example.com",
				SecretName:          "gardener-secret",
				ReadCredsSecretName: sourceSecretName,
			}, nil)

		mockMetrics := &mocks.Metrics{}
		scheme, _ := newCreateTestScheme()

		kcpClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(sourceSecret).
			Build()

		skrClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(kymaSystemNs, existingTargetSecret).
			Build()

		runtimeClientGetter := &fsm_mocks.RuntimeClientGetter{}
		runtimeClientGetter.On("Get", mock.Anything, mock.Anything).Return(skrClient, nil)

		testFsm := &fsm{
			K8s: K8s{
				KcpClient:           kcpClient,
				RuntimeClientGetter: runtimeClientGetter,
			},
			RCCfg: RCCfg{
				AuditLogDataProvider: mockAuditLogProvider,
				Metrics:              mockMetrics,
			},
		}

		systemState := &systemState{
			instance: *instance,
		}

		// when
		stateFn, result, err := sFnCopyAuditLogReadCredentials(ctx, testFsm, systemState)

		// then
		require.NoError(t, err)
		require.Nil(t, result)
		require.Contains(t, stateFn.name(), "updateStatusAndStop")
		require.True(t, systemState.instance.IsProvisioningCompletedStatusSet())

		// Verify secret was updated with new data
		var targetSecret corev1.Secret
		err = skrClient.Get(ctx, client.ObjectKey{Name: auditLogReadCredentialsSecretName, Namespace: "kyma-system"}, &targetSecret)
		require.NoError(t, err)
		require.Equal(t, sourceSecret.Data, targetSecret.Data)

		mockAuditLogProvider.AssertExpectations(t)
		runtimeClientGetter.AssertExpectations(t)
	})
}
