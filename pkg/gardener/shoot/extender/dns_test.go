package extender

import (
	"encoding/json"
	"testing"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestDNSExtender(t *testing.T) {
	t.Run("Create DNS config", func(t *testing.T) {
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
		shoot := fixEmptyGardenerShoot("test", "dev")

		// when
		err := extender(runtimeShoot, &shoot)

		// then
		require.NoError(t, err)
		assert.Equal(t, "myshoot.dev.mydomain.com", *shoot.Spec.DNS.Domain)
		assert.Equal(t, []string{"myshoot.dev.mydomain.com"}, shoot.Spec.DNS.Providers[0].Domains.Include)
		assert.Equal(t, dnsProviderType, *shoot.Spec.DNS.Providers[0].Type)
		assert.Equal(t, secretName, *shoot.Spec.DNS.Providers[0].SecretName)
		assert.Equal(t, true, *shoot.Spec.DNS.Providers[0].Primary)
		assert.NotEmpty(t, shoot.Spec.Extensions[0].ProviderConfig)
		assertExtensionConfig(t, shoot.Spec.Extensions[0].ProviderConfig)
	})
}

func assertExtensionConfig(t *testing.T, rawExtension *runtime.RawExtension) {
	var extension DNSExtensionProviderConfig
	err := json.Unmarshal(rawExtension.Raw, &extension)

	require.NoError(t, err)
	assert.Equal(t, "DNSConfig", extension.Kind)
	assert.Equal(t, "service.dns.extensions.gardener.cloud/v1alpha1", extension.APIVersion)
	assert.Equal(t, true, extension.DNSProviderReplication.Enabled)
	assert.Equal(t, true, *extension.SyncProvidersFromShootSpecDNS)
	assert.Equal(t, 1, len(extension.Providers))
	assert.Equal(t, "myshoot.dev.mydomain.com", extension.Providers[0].Domains.Include[0])
	assert.Equal(t, "my-secret", *extension.Providers[0].SecretName)
	assert.Equal(t, "aws-route53", *extension.Providers[0].Type)
}

func fixEmptyGardenerShoot(name, namespace string) gardener.Shoot {
	return gardener.Shoot{
		ObjectMeta: v1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    map[string]string{},
		},
		Spec: gardener.ShootSpec{},
	}
}
