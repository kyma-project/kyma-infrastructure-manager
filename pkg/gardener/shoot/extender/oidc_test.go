package extender

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
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
		shoot := fixEmptyGardenerShoot("test", "kcp-system")
		runtimeShoot := imv1.Runtime{
			ObjectMeta: metav1.ObjectMeta{},
			Spec: imv1.RuntimeSpec{
				Shoot: imv1.RuntimeShoot{
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
		extender := NewOidcExtenderForCreate(defaultOidc)
		err := extender(runtimeShoot, &shoot)

		// then
		require.NoError(t, err)

		assert.Equal(t, runtimeShoot.Spec.Shoot.Kubernetes.KubeAPIServer.OidcConfig, *shoot.Spec.Kubernetes.KubeAPIServer.OIDCConfig)
		assert.Equal(t, false, *shoot.Spec.Extensions[0].Disabled)
		assert.Equal(t, "shoot-oidc-service", shoot.Spec.Extensions[0].Type)
	})

	emptyOidcExtension := gardener.Extension{
		Type:           "shoot-oidc-service",
		ProviderConfig: &runtime.RawExtension{},
		Disabled:       ptr.To(false),
	}

	for _, testCase := range []struct {
		name              string
		expectedExtension *gardener.Extension
	}{
		{
			name:              "OIDC extension should be added",
			expectedExtension: &emptyOidcExtension,
		},
		{
			name: "OIDC extension should not be added",
		},
	} {
		runtimeShoot := imv1.Runtime{
			Spec: imv1.RuntimeSpec{
				Shoot: imv1.RuntimeShoot{
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

		shoot := fixEmptyGardenerShoot("test", "kcp-system")

		if testCase.expectedExtension != nil {
			shoot.Spec.Extensions = []gardener.Extension{
				*testCase.expectedExtension,
			}

			extender := NewOidcExtenderForPatch(defaultOidc, shoot.Spec.Extensions)
			err := extender(runtimeShoot, &shoot)

			require.NoError(t, err)
			assert.Equal(t, runtimeShoot.Spec.Shoot.Kubernetes.KubeAPIServer.OidcConfig, *shoot.Spec.Kubernetes.KubeAPIServer.OIDCConfig)

			if testCase.expectedExtension != nil {
				assert.Equal(t, emptyOidcExtension, shoot.Spec.Extensions[0])
				assert.Equal(t, "shoot-oidc-service", shoot.Spec.Extensions[0].Type)
			} else {
				assert.Equal(t, 0, len(shoot.Spec.Extensions))
			}
		}
	}

}
