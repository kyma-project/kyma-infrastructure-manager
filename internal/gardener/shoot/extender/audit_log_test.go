package extender

import (
	"encoding/json"
	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v12 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"path/filepath"
	"testing"
)

func TestAuditLogExtender(t *testing.T) {
	t.Run("Should configure Audit Logs", func(t *testing.T) {
		// given
		auditLogPolicyName := "test-policy"
		seedName := "seedName"
		auditLogConfig := &gardener.AuditConfig{
			AuditPolicy: &gardener.AuditPolicy{
				ConfigMapRef: &v12.ObjectReference{Name: auditLogPolicyName},
			},
		}
		runtimeCR := imv1.Runtime{
			Spec: imv1.RuntimeSpec{
				Shoot: imv1.RuntimeShoot{
					Name: "myshoot",
					Provider: imv1.Provider{
						Type: "azure",
					},
				},
			},
		}

		shoot := gardener.Shoot{
			ObjectMeta: v1.ObjectMeta{
				Name:      "test",
				Namespace: "dev",
			},
			Spec: gardener.ShootSpec{
				SeedName: &seedName,
				Region:   "westus2",
			},
		}

		auditLogTenantConfigPath := filepath.Join("testdata", "config.json")
		extender := NewAuditLogExtender(auditLogPolicyName, auditLogTenantConfigPath)

		// when
		err := extender(runtimeCR, &shoot)

		expected := `
            {
                "providerConfig": {
                    "apiVersion": "service.auditlog.extensions.gardener.cloud/v1alpha1",
                    "kind": "AuditlogConfig",
                    "secretReferenceName": "auditlog-credentials",
                    "serviceURL": "https://auditlog.example.com:3000",
                    "tenantID": "a9be5aad-f855-4fd1-a8c8-e95683ec786b",
                    "type": "standard"
                },
                "type": "shoot-auditlog-service"
            }`

		// then
		require.NoError(t, err)
		require.NotNil(t, shoot.Spec.Kubernetes.KubeAPIServer)
		require.NotNil(t, shoot.Spec.Kubernetes.KubeAPIServer.AuditConfig)
		assert.Equal(t, auditLogConfig, shoot.Spec.Kubernetes.KubeAPIServer.AuditConfig)
		actual, _ := json.Marshal(shoot.Spec.Extensions[0])
		assert.JSONEq(t, expected, string(actual))
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
