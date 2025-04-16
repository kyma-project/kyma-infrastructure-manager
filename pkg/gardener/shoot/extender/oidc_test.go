package extender

import (
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/extender/testutils"
	"testing"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestOidcExtender(t *testing.T) {
	defaultOidc := config.OidcProvider{
		ClientID:       "client-id",
		GroupsClaim:    "groups",
		IssuerURL:      "https://my.cool.tokens.com",
		SigningAlgs:    []string{"RS256"},
		UsernameClaim:  "sub",
		UsernamePrefix: "-",
	}

	t.Run("OIDC should be added in create scenario", func(t *testing.T) {
		// given
		shoot := testutils.FixEmptyGardenerShoot("test", "kcp-system")
		runtimeShoot := imv1.Runtime{
			ObjectMeta: metav1.ObjectMeta{},
			Spec: imv1.RuntimeSpec{
				Shoot: imv1.RuntimeShoot{
					Name: "shoot",
					Kubernetes: imv1.Kubernetes{
						KubeAPIServer: imv1.APIServer{
							OidcConfig: gardener.OIDCConfig{
								ClientID:      &defaultOidc.ClientID,
								GroupsClaim:   &defaultOidc.GroupsClaim,
								IssuerURL:     &defaultOidc.IssuerURL,
								SigningAlgs:   defaultOidc.SigningAlgs,
								UsernameClaim: &defaultOidc.UsernameClaim,
							},
						},
					},
				},
			},
		}

		// when
		extender := NewOidcExtender()
		err := extender(runtimeShoot, &shoot)

		// then
		require.NoError(t, err)

		require.Nil(t, shoot.Spec.Kubernetes.KubeAPIServer.OIDCConfig) //nolint:staticcheck
		require.NotNil(t, shoot.Spec.Kubernetes.KubeAPIServer.StructuredAuthentication)
		assert.Equal(t, "structured-auth-config-shoot", shoot.Spec.Kubernetes.KubeAPIServer.StructuredAuthentication.ConfigMapName)
	})
}
