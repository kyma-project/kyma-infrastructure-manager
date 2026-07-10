package auditlog

import (
	"context"
	"testing"

	auditlogv1 "github.com/kyma-project/infrastructure-manager/pkg/auditlog/v1beta1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func TestDefaultDataProvider_GetSharedAuditLogData(t *testing.T) {
	logger := zap.New(zap.UseDevMode(true))

	sharedConfig := Configuration{
		"aws": {
			"eu-central-1": AuditLogData{
				TenantID:   "shared-tenant-1",
				ServiceURL: "https://shared.example.com",
				SecretName: "shared-secret",
			},
		},
	}

	provider := NewDataProvider(nil, sharedConfig, logger, "kcp-system")

	t.Run("returns shared config for existing provider and region", func(t *testing.T) {
		data, err := provider.GetSharedAuditLogData(context.Background(), "aws", "eu-central-1")
		require.NoError(t, err)
		assert.Equal(t, "shared-tenant-1", data.TenantID)
		assert.Equal(t, "https://shared.example.com", data.ServiceURL)
		assert.Equal(t, "shared-secret", data.SecretName)
	})

	t.Run("returns error for missing provider", func(t *testing.T) {
		_, err := provider.GetSharedAuditLogData(context.Background(), "gcp", "eu-central-1")
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrConfigurationNotFound)
	})

	t.Run("returns error for missing region", func(t *testing.T) {
		_, err := provider.GetSharedAuditLogData(context.Background(), "aws", "us-east-1")
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrConfigurationNotFound)
	})
}

func TestDefaultDataProvider_ReserveAuditLog(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, auditlogv1.AddToScheme(scheme))
	logger := zap.New(zap.UseDevMode(true))

	t.Run("successfully reserves available CR", func(t *testing.T) {
		auditLog := createAuditLogCR("al-1", auditlogv1.StateSiemApproved, "", []string{"eu-central-1"}, nil)

		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithRuntimeObjects(&auditLog).
			Build()

		provider := NewDataProvider(fakeClient, nil, logger, "kcp-system")

		err := provider.ReserveAuditLog(context.Background(), "eu-central-1", "test-runtime")
		require.NoError(t, err)

		// Verify reservation labels were set
		var updated auditlogv1.AuditLog
		err = fakeClient.Get(context.Background(), namespacedName("al-1"), &updated)
		require.NoError(t, err)
		assert.Equal(t, "test-runtime", updated.Labels[LabelReservedForRuntimeID])
		assert.NotEmpty(t, updated.Labels[LabelReservedAt])
	})

	t.Run("succeeds when CR already reserved for same runtime (idempotent)", func(t *testing.T) {
		auditLog := createAuditLogCR("al-1", auditlogv1.StateSiemApproved, "", []string{"eu-central-1"}, map[string]string{
			LabelReservedForRuntimeID: "test-runtime",
			LabelReservedAt:           "1719238800",
		})

		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithRuntimeObjects(&auditLog).
			Build()

		provider := NewDataProvider(fakeClient, nil, logger, "kcp-system")

		err := provider.ReserveAuditLog(context.Background(), "eu-central-1", "test-runtime")
		require.NoError(t, err)
	})

	t.Run("returns error when no available CR for region", func(t *testing.T) {
		// CR exists but for different region
		auditLog := createAuditLogCR("al-1", auditlogv1.StateSiemApproved, "", []string{"us-east-1"}, nil)

		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithRuntimeObjects(&auditLog).
			Build()

		provider := NewDataProvider(fakeClient, nil, logger, "kcp-system")

		err := provider.ReserveAuditLog(context.Background(), "eu-central-1", "test-runtime")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no available AuditLogCR in the pool for region eu-central-1")
	})
}

func TestDefaultDataProvider_GetDedicatedAuditLogData(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, auditlogv1.AddToScheme(scheme))
	logger := zap.New(zap.UseDevMode(true))

	t.Run("claims and returns data when claim=true", func(t *testing.T) {
		auditLog := createAuditLogCR("al-1", auditlogv1.StateSiemApproved, "", []string{"eu-central-1"}, map[string]string{
			LabelReservedForRuntimeID: "test-runtime",
		})
		auditLog.Spec.SubaccountID = "dedicated-tenant"
		auditLog.Spec.Config.ServiceURL = "https://dedicated.example.com"
		auditLog.Spec.Config.GardenerSecretName = "dedicated-secret"

		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithRuntimeObjects(&auditLog).
			Build()

		provider := NewDataProvider(fakeClient, nil, logger, "kcp-system")

		data, err := provider.GetDedicatedAuditLogData(context.Background(), "test-runtime", true)
		require.NoError(t, err)
		assert.Equal(t, "dedicated-tenant", data.TenantID)
		assert.Equal(t, "https://dedicated.example.com", data.ServiceURL)
		assert.Equal(t, "dedicated-secret", data.SecretName)

		// Verify claim was set
		var updated auditlogv1.AuditLog
		err = fakeClient.Get(context.Background(), namespacedName("al-1"), &updated)
		require.NoError(t, err)
		assert.Equal(t, "test-runtime", updated.Spec.AssignedToRuntimeID)
	})

	t.Run("claims and returns data from RegistrationReady CR when claim=true", func(t *testing.T) {
		// AuditLogCR in RegistrationReady state (not yet SiemApproved)
		auditLog := createAuditLogCR("al-1", auditlogv1.StateRegistrationReady, "", []string{"eu-central-1"}, map[string]string{
			LabelReservedForRuntimeID: "test-runtime",
		})
		auditLog.Spec.SubaccountID = "dedicated-tenant"
		auditLog.Spec.Config.ServiceURL = "https://dedicated.example.com"
		auditLog.Spec.Config.GardenerSecretName = "dedicated-secret"

		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithRuntimeObjects(&auditLog).
			Build()

		provider := NewDataProvider(fakeClient, nil, logger, "kcp-system")

		data, err := provider.GetDedicatedAuditLogData(context.Background(), "test-runtime", true)
		require.NoError(t, err)
		assert.Equal(t, "dedicated-tenant", data.TenantID)
		assert.Equal(t, "https://dedicated.example.com", data.ServiceURL)
		assert.Equal(t, "dedicated-secret", data.SecretName)

		// Verify claim was set
		var updated auditlogv1.AuditLog
		err = fakeClient.Get(context.Background(), namespacedName("al-1"), &updated)
		require.NoError(t, err)
		assert.Equal(t, "test-runtime", updated.Spec.AssignedToRuntimeID)
	})

	t.Run("returns data without claiming when claim=false", func(t *testing.T) {
		auditLog := createAuditLogCR("al-1", auditlogv1.StateSiemApproved, "test-runtime", []string{"eu-central-1"}, nil)
		auditLog.Spec.SubaccountID = "dedicated-tenant"
		auditLog.Spec.Config.ServiceURL = "https://dedicated.example.com"
		auditLog.Spec.Config.GardenerSecretName = "dedicated-secret"

		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithRuntimeObjects(&auditLog).
			Build()

		provider := NewDataProvider(fakeClient, nil, logger, "kcp-system")

		data, err := provider.GetDedicatedAuditLogData(context.Background(), "test-runtime", false)
		require.NoError(t, err)
		assert.Equal(t, "dedicated-tenant", data.TenantID)
		assert.Equal(t, "https://dedicated.example.com", data.ServiceURL)
		assert.Equal(t, "dedicated-secret", data.SecretName)

		// Verify AssignedToRuntimeID was not modified (no claim operation)
		var updated auditlogv1.AuditLog
		err = fakeClient.Get(context.Background(), namespacedName("al-1"), &updated)
		require.NoError(t, err)
		assert.Equal(t, "test-runtime", updated.Spec.AssignedToRuntimeID)
	})

	t.Run("returns data when claim=true and CR already claimed (idempotent)", func(t *testing.T) {
		// CR is already claimed for the same runtime
		auditLog := createAuditLogCR("al-1", auditlogv1.StateSiemApproved, "test-runtime", []string{"eu-central-1"}, map[string]string{
			LabelReservedForRuntimeID: "test-runtime",
		})
		auditLog.Spec.SubaccountID = "dedicated-tenant"
		auditLog.Spec.Config.ServiceURL = "https://dedicated.example.com"
		auditLog.Spec.Config.GardenerSecretName = "dedicated-secret"

		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithRuntimeObjects(&auditLog).
			Build()

		provider := NewDataProvider(fakeClient, nil, logger, "kcp-system")

		data, err := provider.GetDedicatedAuditLogData(context.Background(), "test-runtime", true)
		require.NoError(t, err)
		assert.Equal(t, "dedicated-tenant", data.TenantID)
		assert.Equal(t, "https://dedicated.example.com", data.ServiceURL)
		assert.Equal(t, "dedicated-secret", data.SecretName)
	})

	t.Run("returns error when no reservation found with claim=true", func(t *testing.T) {
		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			Build()

		provider := NewDataProvider(fakeClient, nil, logger, "kcp-system")

		_, err := provider.GetDedicatedAuditLogData(context.Background(), "test-runtime", true)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no reservation found")
	})
}

func TestDefaultDataProvider_ClaimAuditLog(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, auditlogv1.AddToScheme(scheme))
	logger := zap.New(zap.UseDevMode(true))

	t.Run("successfully claims available CR", func(t *testing.T) {
		auditLog := createAuditLogCR("al-1", auditlogv1.StateSiemApproved, "", []string{"eu-central-1"}, nil)
		auditLog.Spec.SubaccountID = "dedicated-tenant"
		auditLog.Spec.Config.ServiceURL = "https://dedicated.example.com"
		auditLog.Spec.Config.GardenerSecretName = "dedicated-secret"
		auditLog.Spec.Config.ReadCredsSecretName = "read-secret"

		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithRuntimeObjects(&auditLog).
			Build()

		provider := NewDataProvider(fakeClient, nil, logger, "kcp-system")

		data, err := provider.ClaimAuditLog(context.Background(), "eu-central-1", "test-runtime")
		require.NoError(t, err)
		assert.Equal(t, "dedicated-tenant", data.TenantID)
		assert.Equal(t, "https://dedicated.example.com", data.ServiceURL)
		assert.Equal(t, "dedicated-secret", data.SecretName)
		assert.Equal(t, "read-secret", data.ReadCredsSecretName)

		// Verify CR was claimed
		var updated auditlogv1.AuditLog
		err = fakeClient.Get(context.Background(), namespacedName("al-1"), &updated)
		require.NoError(t, err)
		assert.Equal(t, "test-runtime", updated.Spec.AssignedToRuntimeID)
		// Verify reservation labels were also set (workaround for sFnMigrateToDedicatedAuditLog)
		assert.Equal(t, "test-runtime", updated.Labels[LabelReservedForRuntimeID])
		assert.NotEmpty(t, updated.Labels[LabelReservedAt])
	})

	t.Run("returns existing data when already claimed (idempotent)", func(t *testing.T) {
		auditLog := createAuditLogCR("al-1", auditlogv1.StateSiemApproved, "test-runtime", []string{"eu-central-1"}, nil)
		auditLog.Spec.SubaccountID = "dedicated-tenant"
		auditLog.Spec.Config.ServiceURL = "https://dedicated.example.com"
		auditLog.Spec.Config.GardenerSecretName = "dedicated-secret"
		auditLog.Spec.Config.ReadCredsSecretName = "read-secret"

		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithRuntimeObjects(&auditLog).
			Build()

		provider := NewDataProvider(fakeClient, nil, logger, "kcp-system")

		data, err := provider.ClaimAuditLog(context.Background(), "eu-central-1", "test-runtime")
		require.NoError(t, err)
		assert.Equal(t, "dedicated-tenant", data.TenantID)
		assert.Equal(t, "https://dedicated.example.com", data.ServiceURL)
		assert.Equal(t, "dedicated-secret", data.SecretName)
		assert.Equal(t, "read-secret", data.ReadCredsSecretName)

		// Verify AssignedToRuntimeID is still set (not modified)
		var updated auditlogv1.AuditLog
		err = fakeClient.Get(context.Background(), namespacedName("al-1"), &updated)
		require.NoError(t, err)
		assert.Equal(t, "test-runtime", updated.Spec.AssignedToRuntimeID)
	})

	t.Run("returns existing data when CR is reserved (idempotent)", func(t *testing.T) {
		// CR is reserved but not claimed yet - getDedicatedAuditLogDataWithoutClaim finds it by reservation
		auditLog := createAuditLogCR("al-1", auditlogv1.StateSiemApproved, "", []string{"eu-central-1"}, map[string]string{
			LabelReservedForRuntimeID: "test-runtime",
		})
		auditLog.Spec.SubaccountID = "dedicated-tenant"
		auditLog.Spec.Config.ServiceURL = "https://dedicated.example.com"
		auditLog.Spec.Config.GardenerSecretName = "dedicated-secret"

		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithRuntimeObjects(&auditLog).
			Build()

		provider := NewDataProvider(fakeClient, nil, logger, "kcp-system")

		data, err := provider.ClaimAuditLog(context.Background(), "eu-central-1", "test-runtime")
		require.NoError(t, err)
		assert.Equal(t, "dedicated-tenant", data.TenantID)

		// getDedicatedAuditLogDataWithoutClaim finds reserved CR, so no update happens
		var updated auditlogv1.AuditLog
		err = fakeClient.Get(context.Background(), namespacedName("al-1"), &updated)
		require.NoError(t, err)
		assert.Equal(t, "", updated.Spec.AssignedToRuntimeID) // Still not claimed, but that's OK
	})

	t.Run("returns error when no available CR for region", func(t *testing.T) {
		// CR exists but for different region
		auditLog := createAuditLogCR("al-1", auditlogv1.StateSiemApproved, "", []string{"us-east-1"}, nil)

		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithRuntimeObjects(&auditLog).
			Build()

		provider := NewDataProvider(fakeClient, nil, logger, "kcp-system")

		_, err := provider.ClaimAuditLog(context.Background(), "eu-central-1", "test-runtime")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no available AuditLogCR for region eu-central-1")
	})

	t.Run("returns error when all CRs in region are claimed", func(t *testing.T) {
		// CR exists for the region but already claimed by different runtime
		auditLog := createAuditLogCR("al-1", auditlogv1.StateSiemApproved, "other-runtime", []string{"eu-central-1"}, nil)

		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithRuntimeObjects(&auditLog).
			Build()

		provider := NewDataProvider(fakeClient, nil, logger, "kcp-system")

		_, err := provider.ClaimAuditLog(context.Background(), "eu-central-1", "test-runtime")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no available AuditLogCR for region eu-central-1")
	})

	t.Run("claims from RegistrationReady CR", func(t *testing.T) {
		// AuditLogCR in RegistrationReady state (not yet SiemApproved)
		auditLog := createAuditLogCR("al-1", auditlogv1.StateRegistrationReady, "", []string{"eu-central-1"}, nil)
		auditLog.Spec.SubaccountID = "dedicated-tenant"
		auditLog.Spec.Config.ServiceURL = "https://dedicated.example.com"
		auditLog.Spec.Config.GardenerSecretName = "dedicated-secret"

		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithRuntimeObjects(&auditLog).
			Build()

		provider := NewDataProvider(fakeClient, nil, logger, "kcp-system")

		data, err := provider.ClaimAuditLog(context.Background(), "eu-central-1", "test-runtime")
		require.NoError(t, err)
		assert.Equal(t, "dedicated-tenant", data.TenantID)

		// Verify CR was claimed
		var updated auditlogv1.AuditLog
		err = fakeClient.Get(context.Background(), namespacedName("al-1"), &updated)
		require.NoError(t, err)
		assert.Equal(t, "test-runtime", updated.Spec.AssignedToRuntimeID)
		// Verify reservation labels were also set (workaround for sFnMigrateToDedicatedAuditLog)
		assert.Equal(t, "test-runtime", updated.Labels[LabelReservedForRuntimeID])
		assert.NotEmpty(t, updated.Labels[LabelReservedAt])
	})
}

func TestDefaultDataProvider_ReleaseDedicated(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, auditlogv1.AddToScheme(scheme))
	logger := zap.New(zap.UseDevMode(true))

	t.Run("releases assigned CR", func(t *testing.T) {
		auditLog := createAuditLogCR("al-1", auditlogv1.StateAssigned, "test-runtime", []string{"eu-central-1"}, nil)

		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithRuntimeObjects(&auditLog).
			Build()

		provider := NewDataProvider(fakeClient, nil, logger, "kcp-system")

		err := provider.ReleaseDedicated(context.Background(), "test-runtime")
		require.NoError(t, err)

		// Verify orphaned flag was set
		var updated auditlogv1.AuditLog
		err = fakeClient.Get(context.Background(), namespacedName("al-1"), &updated)
		require.NoError(t, err)
		assert.True(t, updated.Spec.Orphaned)
	})

	t.Run("succeeds when no CR found", func(t *testing.T) {
		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			Build()

		provider := NewDataProvider(fakeClient, nil, logger, "kcp-system")

		err := provider.ReleaseDedicated(context.Background(), "test-runtime")
		require.NoError(t, err)
	})

	t.Run("returns error when List fails", func(t *testing.T) {
		// Create a client without proper scheme setup to simulate List failure
		fakeClient := fake.NewClientBuilder().Build()

		provider := NewDataProvider(fakeClient, nil, logger, "kcp-system")

		err := provider.ReleaseDedicated(context.Background(), "test-runtime")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to find assigned AuditLogCR for runtime")
	})
}
