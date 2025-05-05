package extender

import (
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/extender/testutils"
	"testing"

	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDNSExtender(t *testing.T) {
	t.Run("Create DNS config for create scenario", func(t *testing.T) {
		// given
		secretName := "my-secret"
		domainPrefix := "dev.mydomain.com"
		dnsProviderType := "aws-route53"
		runtimeShoot := imv1.Runtime{
			Spec: imv1.RuntimeSpec{
				Shoot: imv1.RuntimeShoot{
					Name: "myshoot",
				},
			},
		}
		extender := NewDNSExtender(secretName, domainPrefix, dnsProviderType)
		shoot := testutils.FixEmptyGardenerShoot("test", "dev")

		// when
		err := extender(runtimeShoot, &shoot)

		// then
		require.NoError(t, err)
		assert.Equal(t, "myshoot.dev.mydomain.com", *shoot.Spec.DNS.Domain)
		assert.Equal(t, []string{"myshoot.dev.mydomain.com"}, shoot.Spec.DNS.Providers[0].Domains.Include) //nolint:staticcheck
		assert.Equal(t, dnsProviderType, *shoot.Spec.DNS.Providers[0].Type) //nolint:staticcheck
		assert.Equal(t, secretName, *shoot.Spec.DNS.Providers[0].SecretName) //nolint:staticcheck
		assert.Equal(t, true, *shoot.Spec.DNS.Providers[0].Primary) //nolint:staticcheck
	})
}
