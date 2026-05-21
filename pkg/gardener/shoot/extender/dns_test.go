package extender

import (
	"testing"

	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/extender/testutils"

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

	t.Run("Set credentialsRef", func(t *testing.T) {
		extender := NewDNSExtender(secretName, domainPrefix, dnsProviderType)
		shoot := testutils.FixEmptyGardenerShoot("test", "dev")

		err := extender(runtimeShoot, &shoot)

		require.NoError(t, err)
		assert.Equal(t, "myshoot.dev.mydomain.com", *shoot.Spec.DNS.Domain)
		assert.Equal(t, []string{"myshoot.dev.mydomain.com"}, shoot.Spec.DNS.Providers[0].Domains.Include) //nolint:staticcheck
		assert.Equal(t, dnsProviderType, *shoot.Spec.DNS.Providers[0].Type)                                //nolint:staticcheck
		assert.Equal(t, true, *shoot.Spec.DNS.Providers[0].Primary)                                        //nolint:staticcheck

		require.NotNil(t, shoot.Spec.DNS.Providers[0].CredentialsRef)                //nolint:staticcheck
		assert.Equal(t, "v1", shoot.Spec.DNS.Providers[0].CredentialsRef.APIVersion) //nolint:staticcheck
		assert.Equal(t, "Secret", shoot.Spec.DNS.Providers[0].CredentialsRef.Kind)   //nolint:staticcheck
		assert.Equal(t, secretName, shoot.Spec.DNS.Providers[0].CredentialsRef.Name) //nolint:staticcheck
	})
}
