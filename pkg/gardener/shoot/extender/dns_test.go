package extender

import (
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/extender/testutils"
	"testing"

	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDNSExtender(t *testing.T) {
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

	t.Run("Set credentialsRef when useCredentialsRef is true", func(t *testing.T) {
		extender := NewDNSExtender(secretName, domainPrefix, dnsProviderType, true)
		shoot := testutils.FixEmptyGardenerShoot("test", "dev")

		err := extender(runtimeShoot, &shoot)

		require.NoError(t, err)
		assert.Equal(t, "myshoot.dev.mydomain.com", *shoot.Spec.DNS.Domain)
		assert.Equal(t, []string{"myshoot.dev.mydomain.com"}, shoot.Spec.DNS.Providers[0].Domains.Include) //nolint:staticcheck
		assert.Equal(t, dnsProviderType, *shoot.Spec.DNS.Providers[0].Type)                                //nolint:staticcheck
		assert.Equal(t, true, *shoot.Spec.DNS.Providers[0].Primary)                                        //nolint:staticcheck

		require.Nil(t, shoot.Spec.DNS.Providers[0].SecretName) //nolint:staticcheck
		require.NotNil(t, shoot.Spec.DNS.Providers[0].CredentialsRef)
		assert.Equal(t, "v1", shoot.Spec.DNS.Providers[0].CredentialsRef.APIVersion)
		assert.Equal(t, "Secret", shoot.Spec.DNS.Providers[0].CredentialsRef.Kind)
		assert.Equal(t, secretName, shoot.Spec.DNS.Providers[0].CredentialsRef.Name)
	})

	t.Run("Set deprecated secretName when useCredentialsRef is false", func(t *testing.T) {
		extender := NewDNSExtender(secretName, domainPrefix, dnsProviderType, false)
		shoot := testutils.FixEmptyGardenerShoot("test", "dev")

		err := extender(runtimeShoot, &shoot)

		require.NoError(t, err)
		assert.Equal(t, secretName, *shoot.Spec.DNS.Providers[0].SecretName) //nolint:staticcheck
		assert.Nil(t, shoot.Spec.DNS.Providers[0].CredentialsRef)
	})
}
