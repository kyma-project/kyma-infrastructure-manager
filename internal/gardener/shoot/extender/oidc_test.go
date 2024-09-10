package extender

import (
	"testing"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOidcExtender(t *testing.T) {
	t.Run("Create kubernetes config", func(t *testing.T) {
		// given
		clientID := "client-id"
		groupsClaim := "groups"
		issuerURL := "https://my.cool.tokens.com"
		usernameClaim := "sub"

		shoot := fixEmptyGardenerShoot("test", "kcp-system")
		runtimeShoot := imv1.Runtime{
			Spec: imv1.RuntimeSpec{
				Shoot: imv1.RuntimeShoot{
					Kubernetes: imv1.Kubernetes{
						KubeAPIServer: imv1.APIServer{
							OidcConfig: gardener.OIDCConfig{
								ClientID:    &clientID,
								GroupsClaim: &groupsClaim,
								IssuerURL:   &issuerURL,
								SigningAlgs: []string{
									"RS256",
								},
								UsernameClaim: &usernameClaim,
							},
						},
					},
				},
			},
		}

		// when
		err := ExtendWithOIDC(runtimeShoot, &shoot)

		// then
		require.NoError(t, err)

		assert.Equal(t, runtimeShoot.Spec.Shoot.Kubernetes.KubeAPIServer.OidcConfig, *shoot.Spec.Kubernetes.KubeAPIServer.OIDCConfig)
		assert.Equal(t, false, *shoot.Spec.Extensions[0].Disabled)
		assert.Equal(t, "shoot-oidc-service", shoot.Spec.Extensions[0].Type)
	})
}
