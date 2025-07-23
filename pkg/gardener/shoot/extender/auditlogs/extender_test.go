package auditlogs

import (
	"testing"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_AuditlogExtenderExperimentalCfg(t *testing.T) {
	defaultPolicyConfigmapName := "default"
	for _, tc := range []struct {
		rt                 imv1.Runtime
		shoot              gardener.Shoot
		data               AuditLogData
		expectedRefMapName string
	}{
		{
			shoot: gardener.Shoot{},
			rt: imv1.Runtime{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"operator.kyma-project.io/experimental-audit-policy": "xxx",
					},
				},
			},
			data: AuditLogData{
				TenantID:   "tenant-id",
				ServiceURL: "testme",
			},
			expectedRefMapName: defaultPolicyConfigmapName,
		},
		{
			shoot: gardener.Shoot{},
			rt: imv1.Runtime{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"operator.kyma-project.io/experimental-audit-policy": "false",
					},
				},
			},
			data: AuditLogData{
				TenantID:   "tenant-id",
				ServiceURL: "testme",
			},
			expectedRefMapName: defaultPolicyConfigmapName,
		},
		{
			shoot: gardener.Shoot{},
			rt: imv1.Runtime{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"operator.kyma-project.io/experimental-audit-policy": "true",
					},
				},
			},
			data: AuditLogData{
				TenantID:   "tenant-id",
				ServiceURL: "testme",
			},
			expectedRefMapName: "experimental-audit-policy",
		},
		{
			shoot: gardener.Shoot{},
			data: AuditLogData{
				TenantID:   "tenant-id",
				ServiceURL: "testme",
			},
			expectedRefMapName: defaultPolicyConfigmapName,
		},
	} {
		// given
		extendWithAuditlogs := NewAuditlogExtenderForCreate(defaultPolicyConfigmapName, tc.data)

		// when
		err := extendWithAuditlogs(tc.rt, &tc.shoot)

		// then
		require.NoError(t, err)

		// then
		require.Equal(t, tc.expectedRefMapName, tc.shoot.Spec.Kubernetes.KubeAPIServer.AuditConfig.AuditPolicy.ConfigMapRef.Name)
	}
}

func Test_AuditlogExtender(t *testing.T) {
	var zero imv1.Runtime
	for _, tc := range []struct {
		shoot               gardener.Shoot
		data                AuditLogData
		policyConfigmapName string
	}{
		{
			shoot: gardener.Shoot{},
			data: AuditLogData{
				TenantID:   "tenant-id",
				ServiceURL: "testme",
			},
		},
	} {
		// given
		extendWithAuditlogs := NewAuditlogExtenderForCreate(tc.policyConfigmapName, tc.data)

		// when
		err := extendWithAuditlogs(zero, &tc.shoot)

		// then
		require.NoError(t, err)
	}
}
