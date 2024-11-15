package auditlogs

import (
	"testing"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
)

func Test_oSetPolicyConfigmap(t *testing.T) {
	for _, testCase := range []struct {
		shoot               gardener.Shoot
		policyConfigmapName string
	}{
		{
			shoot:               gardener.Shoot{},
			policyConfigmapName: "test-me-plz",
		},
		{
			shoot: gardener.Shoot{
				Spec: gardener.ShootSpec{
					Kubernetes: gardener.Kubernetes{
						KubeAPIServer: &gardener.KubeAPIServerConfig{
							AuditConfig: &gardener.AuditConfig{
								AuditPolicy: &gardener.AuditPolicy{
									ConfigMapRef: &v1.ObjectReference{
										Name: "test",
									},
								},
							},
						},
					},
				},
			},
			policyConfigmapName: "test-me-plz",
		},
	} {
		// given
		operate := oSetPolicyConfigmap(testCase.policyConfigmapName)

		// when
		err := operate(&testCase.shoot)

		// then
		require.NoError(t, err)
		assert.Equal(t,
			testCase.policyConfigmapName,
			testCase.shoot.Spec.Kubernetes.KubeAPIServer.AuditConfig.AuditPolicy.ConfigMapRef.Name)
	}
}
