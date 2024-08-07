package extender

import (
	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v12 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func TestAuditLogExtender(t *testing.T) {
	t.Run("Should configure Audit Logs", func(t *testing.T) {
		// given
		auditLogPolicyName := "test-policy"
		auditLogTenantConfigPath := "test-tenant"
		auditLogConfig := &gardener.AuditConfig{
			AuditPolicy: &gardener.AuditPolicy{
				ConfigMapRef: &v12.ObjectReference{Name: auditLogPolicyName},
			},
		}
		runtimeShoot := imv1.Runtime{
			Spec: imv1.RuntimeSpec{
				Shoot: imv1.RuntimeShoot{
					Name: "myshoot",
				},
			},
		}
		extender := NewAuditLogExtender(auditLogPolicyName, auditLogTenantConfigPath)
		shoot := fixAuditLogGardenerShoot("test", "dev", "seedName")

		// when
		err := extender(runtimeShoot, &shoot)

		// then
		require.NoError(t, err)
		require.NotNil(t, shoot.Spec.Kubernetes.KubeAPIServer)
		require.NotNil(t, shoot.Spec.Kubernetes.KubeAPIServer.AuditConfig)
		assert.Equal(t, auditLogConfig, shoot.Spec.Kubernetes.KubeAPIServer.AuditConfig)
	})
	t.Run("Should not configure Audit Logs when shoot seed name is empty", func(t *testing.T) {
		// given
		auditLogPolicyName := ""
		auditLogTenantConfigPath := ""
		runtimeShoot := imv1.Runtime{
			Spec: imv1.RuntimeSpec{
				Shoot: imv1.RuntimeShoot{
					Name: "myshoot",
				},
			},
		}
		extender := NewAuditLogExtender(auditLogPolicyName, auditLogTenantConfigPath)
		expectedShoot := fixEmptyGardenerShoot("test", "dev")
		testShoot := expectedShoot.DeepCopy()

		// when
		err := extender(runtimeShoot, testShoot)

		// then
		require.NoError(t, err)
		require.Nil(t, testShoot.Spec.Kubernetes.KubeAPIServer)
	})
	t.Run("Should not create Audit Log config when policyConfigMapName and tenantConfigPath is not configured", func(t *testing.T) {
		// given
		auditLogPolicyName := ""
		auditLogTenantConfigPath := ""
		runtimeShoot := imv1.Runtime{
			Spec: imv1.RuntimeSpec{
				Shoot: imv1.RuntimeShoot{
					Name: "myshoot",
				},
			},
		}
		extender := NewAuditLogExtender(auditLogPolicyName, auditLogTenantConfigPath)
		expectedShoot := fixAuditLogGardenerShoot("test", "dev", "seedName")
		testShoot := expectedShoot.DeepCopy()

		// when
		err := extender(runtimeShoot, testShoot)

		// then
		require.NoError(t, err)
		assert.Equal(t, &expectedShoot, testShoot)
	})
}

func fixAuditLogGardenerShoot(name, namespace, shootName string) gardener.Shoot {
	return gardener.Shoot{
		ObjectMeta: v1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: gardener.ShootSpec{
			SeedName: &shootName,
		},
	}
}
