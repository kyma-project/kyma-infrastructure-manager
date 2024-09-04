package tests

import (
	"context"
	"fmt"
	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/kyma-project/infrastructure-manager/internal/auditlogging"
	"github.com/kyma-project/infrastructure-manager/internal/auditlogging/mocks"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"testing"
)

var auditLogMandatory = true

func TestEnable(t *testing.T) {
	t.Run("Should successfully enable Audit Log for Shoot", func(t *testing.T) {
		// given
		ctx := context.Background()
		configurator := &mocks.AuditLogConfigurator{}
		shoot := shootForTest()
		shoot.Spec.SeedName = ptr.To("seed-name")
		configFromFile := fileConfigData()
		seedKey := types.NamespacedName{Name: "seed-name", Namespace: ""}

		configurator.On("CanEnableAuditLogsForShoot", "seed-name").Return(true).Once()
		configurator.On("GetConfigFromFile").Return(configFromFile, nil).Once()
		configurator.On("GetPolicyConfigMapName").Return("policyConfigMapName").Once()
		configurator.On("GetSeedObj", ctx, seedKey).Return(seedForTest(), nil).Once()
		configurator.On("UpdateShoot", ctx, shoot).Return(nil).Once()

		// when
		auditLog := &auditlogging.AuditLog{AuditLogConfigurator: configurator}
		enable, err := auditLog.Enable(ctx, shoot, auditLogMandatory)

		// then
		configurator.AssertExpectations(t)
		require.True(t, enable)
		require.NoError(t, err)
	})

	t.Run("Should return false and error if was problem during enabling Audit Logs", func(t *testing.T) {
		// given
		ctx := context.Background()
		configurator := &mocks.AuditLogConfigurator{}
		shoot := shootForTest()
		shoot.Spec.SeedName = ptr.To("seed-name")
		configFromFile := fileConfigData()
		seedKey := types.NamespacedName{Name: "seed-name", Namespace: ""}
		seed := seedForTest()
		// delete shoot region to simulate error
		shoot.Spec.Region = ""

		configurator.On("CanEnableAuditLogsForShoot", "seed-name").Return(true).Once()
		configurator.On("GetConfigFromFile").Return(configFromFile, nil).Once()
		configurator.On("GetPolicyConfigMapName").Return("policyConfigMapName").Once()
		configurator.On("GetSeedObj", ctx, seedKey).Return(seed, nil).Once()

		// when
		auditLog := &auditlogging.AuditLog{AuditLogConfigurator: configurator}
		enable, err := auditLog.Enable(ctx, shoot, auditLogMandatory)

		// then
		configurator.AssertExpectations(t)
		require.False(t, enable)
		require.Error(t, err)
	})

	t.Run("Should return false and error if seed is empty", func(t *testing.T) {
		// given
		ctx := context.Background()
		configurator := &mocks.AuditLogConfigurator{}
		shoot := shootForTest()
		shoot.Spec.SeedName = nil

		configurator.On("CanEnableAuditLogsForShoot", "").Return(false).Once()

		// when

		auditLog := &auditlogging.AuditLog{AuditLogConfigurator: configurator}
		enable, err := auditLog.Enable(ctx, shoot, auditLogMandatory)

		// then
		configurator.AssertExpectations(t)
		require.False(t, enable)
		require.Error(t, err)
	})
}

func TestApplyAuditLogConfig(t *testing.T) {
	t.Run("Should apply Audit Log config to Shoot", func(t *testing.T) {
		// given
		shoot := shootForTest()
		configFromFile := fileConfigData()

		// when
		annotated, err := auditlogging.ApplyAuditLogConfig(shoot, configFromFile, "aws", auditLogMandatory)

		// then
		require.True(t, annotated)
		require.NoError(t, err)
	})

	t.Run("Should return false and error if can not find config for provider", func(t *testing.T) {
		// given
		shoot := shootForTest()

		// when
		annotated, err := auditlogging.ApplyAuditLogConfig(shoot, map[string]map[string]auditlogging.AuditLogData{}, "aws", auditLogMandatory)

		// then
		require.False(t, annotated)
		require.Equal(t, err, fmt.Errorf("cannot find config for provider aws"))
	})

	t.Run("Should return false and error if Shoot region is empty", func(t *testing.T) {
		// given
		shoot := shootForTest()
		configFromFile := fileConfigData()
		shoot.Spec.Region = ""

		// when
		annotated, err := auditlogging.ApplyAuditLogConfig(shoot, configFromFile, "aws", auditLogMandatory)

		// then
		require.False(t, annotated)
		require.Equal(t, err, fmt.Errorf("shoot has no region set"))
	})

	t.Run("Should return false and error if provider for region is empty", func(t *testing.T) {
		// given
		shoot := shootForTest()
		configFromFile := map[string]map[string]auditlogging.AuditLogData{
			"aws": {},
		}

		// when
		annotated, err := auditlogging.ApplyAuditLogConfig(shoot, configFromFile, "aws", auditLogMandatory)

		// then
		require.False(t, annotated)
		require.Equal(t, err, fmt.Errorf("auditlog config for region region, provider aws is empty"))
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

func seedForTest() gardener.Seed {
	return gardener.Seed{
		ObjectMeta: metav1.ObjectMeta{
			Name: "seed-name",
		},
		Spec: gardener.SeedSpec{
			Provider: gardener.SeedProvider{
				Type: "aws",
			},
		},
	}
}

func fileConfigData() map[string]map[string]auditlogging.AuditLogData {
	return map[string]map[string]auditlogging.AuditLogData{
		"aws": {
			"region": {
				TenantID:   "tenantID",
				ServiceURL: "serviceURL",
				SecretName: "secretName",
			},
		},
	}
}
