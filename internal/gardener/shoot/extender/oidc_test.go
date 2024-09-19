package extender

import (
	"testing"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestOidcExtender(t *testing.T) {
	const MigratorLabel = "operator.kyma-project.io/created-by-migrator"
	for _, testCase := range []struct {
		name                         string
		migratorLabel                map[string]string
		expectedOidcExtensionEnabled bool
	}{
		{
			name:                         "label created-by-migrator=true should not configure OIDC",
			migratorLabel:                map[string]string{MigratorLabel: "true"},
			expectedOidcExtensionEnabled: false,
		},
		{
			name:                         "label created-by-migrator=false should configure OIDC",
			migratorLabel:                map[string]string{MigratorLabel: "false"},
			expectedOidcExtensionEnabled: true,
		},
		{
			name:                         "label created-by-migrator unset should configure OIDC",
			migratorLabel:                nil,
			expectedOidcExtensionEnabled: true,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			// given
			clientID := "client-id"
			groupsClaim := "groups"
			issuerURL := "https://my.cool.tokens.com"
			usernameClaim := "sub"

			shoot := fixEmptyGardenerShoot("test", "kcp-system")
			runtimeShoot := imv1.Runtime{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						MigratorLabel: testCase.migratorLabel[MigratorLabel],
					},
				},
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
			if testCase.expectedOidcExtensionEnabled {
				assert.Equal(t, testCase.expectedOidcExtensionEnabled, !*shoot.Spec.Extensions[0].Disabled)
				assert.Equal(t, "shoot-oidc-service", shoot.Spec.Extensions[0].Type)
			} else {
				assert.Equal(t, 0, len(shoot.Spec.Extensions))
			}
		})
	}
}
