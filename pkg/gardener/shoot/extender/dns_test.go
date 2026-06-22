package extender

import (
	"testing"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
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
		extender := NewDNSExtenderForCreate(secretName, domainPrefix, dnsProviderType)
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

func TestDNSExtenderForPatch(t *testing.T) {
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

	t.Run("Preserve existing DNS when providers list is empty", func(t *testing.T) {
		existingDomain := "existing.domain.com"
		existingDNS := &gardener.DNS{
			Domain:    &existingDomain,
			Providers: []gardener.DNSProvider{},
		}
		extender := NewDNSExtenderForPatch(secretName, domainPrefix, dnsProviderType, existingDNS)
		shoot := testutils.FixEmptyGardenerShoot("test", "dev")

		err := extender(runtimeShoot, &shoot)

		require.NoError(t, err)
		require.NotNil(t, shoot.Spec.DNS)
		assert.Equal(t, existingDNS, shoot.Spec.DNS)
		//nolint:staticcheck // SA1019: Needs to be removed at some point
		assert.Empty(t, shoot.Spec.DNS.Providers)
	})

	t.Run("Create new DNS config when existing DNS has providers", func(t *testing.T) {
		existingProviderType := "gcp-clouddns"
		existingDNS := &gardener.DNS{
			Providers: []gardener.DNSProvider{
				{Type: &existingProviderType},
			},
		}
		extender := NewDNSExtenderForPatch(secretName, domainPrefix, dnsProviderType, existingDNS)
		shoot := testutils.FixEmptyGardenerShoot("test", "dev")

		err := extender(runtimeShoot, &shoot)

		require.NoError(t, err)
		require.NotNil(t, shoot.Spec.DNS)
		assert.Equal(t, "myshoot.dev.mydomain.com", *shoot.Spec.DNS.Domain)
		assert.Equal(t, []string{"myshoot.dev.mydomain.com"}, shoot.Spec.DNS.Providers[0].Domains.Include) //nolint:staticcheck
		assert.Equal(t, dnsProviderType, *shoot.Spec.DNS.Providers[0].Type)                                //nolint:staticcheck
		assert.Equal(t, true, *shoot.Spec.DNS.Providers[0].Primary)                                        //nolint:staticcheck

		require.NotNil(t, shoot.Spec.DNS.Providers[0].CredentialsRef)                //nolint:staticcheck
		assert.Equal(t, "v1", shoot.Spec.DNS.Providers[0].CredentialsRef.APIVersion) //nolint:staticcheck
		assert.Equal(t, "Secret", shoot.Spec.DNS.Providers[0].CredentialsRef.Kind)   //nolint:staticcheck
		assert.Equal(t, secretName, shoot.Spec.DNS.Providers[0].CredentialsRef.Name) //nolint:staticcheck
	})

	t.Run("Create new DNS config when existingDNS is nil", func(t *testing.T) {
		extender := NewDNSExtenderForPatch(secretName, domainPrefix, dnsProviderType, nil)
		shoot := testutils.FixEmptyGardenerShoot("test", "dev")

		err := extender(runtimeShoot, &shoot)

		require.NoError(t, err)
		assert.NotNil(t, shoot.Spec.DNS)
	})
}
