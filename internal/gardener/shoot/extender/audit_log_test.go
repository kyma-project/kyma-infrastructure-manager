package extender

import (
	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v12 "k8s.io/api/core/v1"
	"testing"
)

func TestAuditLogExtender(t *testing.T) {
	t.Run("Should create Audit Log config when policyConfigMapName is not empty", func(t *testing.T) {
		// given
		auditLogPolicyName := "test-policy"
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
		extender := NewAuditLogExtender(auditLogPolicyName)
		shoot := fixEmptyGardenerShoot("test", "dev")

		// when
		err := extender(runtimeShoot, &shoot)

		// then
		require.NoError(t, err)
		assert.Equal(t, auditLogConfig, shoot.Spec.Kubernetes.KubeAPIServer.AuditConfig)
	})
	t.Run("Should not create Audit Log config when policyConfigMapName is not configured", func(t *testing.T) {
		// given
		auditLogPolicyName := ""
		runtimeShoot := imv1.Runtime{
			Spec: imv1.RuntimeSpec{
				Shoot: imv1.RuntimeShoot{
					Name: "myshoot",
				},
			},
		}
		extender := NewAuditLogExtender(auditLogPolicyName)
		expectedShoot := fixEmptyGardenerShoot("test", "dev")
		testShoot := expectedShoot.DeepCopy()

		// when
		err := extender(runtimeShoot, testShoot)

		// then
		require.NoError(t, err)
		assert.Equal(t, &expectedShoot, testShoot)
	})
}
