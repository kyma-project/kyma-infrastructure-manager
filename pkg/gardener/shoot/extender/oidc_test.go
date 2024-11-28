package extender

import (
	"testing"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestOidcExtender(t *testing.T) {
	const migratorLabel = "operator.kyma-project.io/created-by-migrator"
	for _, testCase := range []struct {
		name                         string
		migratorLabel                map[string]string
		expectedOidcExtensionEnabled bool
	}{
		{
			name:                         "label created-by-migrator=true should not configure OIDC",
			migratorLabel:                map[string]string{migratorLabel: "true"},
			expectedOidcExtensionEnabled: false,
		},
		{
			name:                         "label created-by-migrator=false should configure OIDC",
			migratorLabel:                map[string]string{migratorLabel: "false"},
			expectedOidcExtensionEnabled: true,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			// given
			defaultOidc := config.OidcProvider{
				ClientID:       "client-id",
				GroupsClaim:    "groups",
				IssuerURL:      "https://my.cool.tokens.com",
				SigningAlgs:    []string{"RS256"},
				UsernameClaim:  "sub",
				UsernamePrefix: "-",
			}

			shoot := fixEmptyGardenerShoot("test", "kcp-system")
			runtimeShoot := imv1.Runtime{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						migratorLabel: testCase.migratorLabel[migratorLabel],
					},
				},
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
			extender := NewOidcExtender(defaultOidc)
			err := extender(runtimeShoot, &shoot)

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
