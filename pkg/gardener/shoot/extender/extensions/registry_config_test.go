package extensions

import (
	registrycacheext "github.com/gardener/gardener-extension-registry-cache/pkg/apis/registry/v1alpha3"
	registrycache "github.com/kyma-project/kim-snatch/api/v1beta1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/yaml"
	"testing"
	"time"
)

func TestNewRegistryCacheExtension(t *testing.T) {

	t.Run("should create registry cache extension", func(t *testing.T) {

		// given
		volumeQuantity := resource.MustParse("10Gi")
		caches := []registrycache.RegistryCache{
			{
				Upstream: "ghcr.io",
				GarbageCollection: &registrycache.GarbageCollection{
					TTL: metav1.Duration{Duration: time.Hour * 24},
				},
				SecretReferenceName: ptr.To("secret"),
				Volume: &registrycache.Volume{
					Size:             &volumeQuantity,
					StorageClassName: ptr.To("storageClass"),
				},
			},
			{
				RemoteURL: ptr.To("http://my-registry.io:5000"),
				Proxy: &registrycache.Proxy{
					HTTPProxy:  ptr.To("http://proxy.io:5000"),
					HTTPSProxy: ptr.To("https://proxy.io:5000"),
				},
			},
		}

		// when
		registryCacheExtension, err := NewRegistryCacheExtension(caches, true)

		// then
		require.NoError(t, err)
		require.NotNil(t, registryCacheExtension)

		require.Equal(t, RegistryCacheExtensionType, registryCacheExtension.Type)
		require.NotNil(t, registryCacheExtension.ProviderConfig)
		require.NotNil(t, registryCacheExtension.ProviderConfig.Raw)
		require.Equal(t, registryCacheExtension.Disabled, ptr.To(false))

		var providerConfig registrycacheext.RegistryConfig
		err = yaml.Unmarshal(registryCacheExtension.ProviderConfig.Raw, &providerConfig)
		require.NoError(t, err)

		assert.Equal(t, "registry.extensions.gardener.cloud/v1alpha3", providerConfig.APIVersion)
		assert.Equal(t, "RegistryConfig", providerConfig.Kind)
		assert.Equal(t, "ghcr.io", providerConfig.Caches[0].Upstream)
		assert.Equal(t, metav1.Duration{Duration: time.Hour * 24}, providerConfig.Caches[0].GarbageCollection.TTL)
		assert.Equal(t, ptr.To("secret"), providerConfig.Caches[0].SecretReferenceName)
		assert.Nil(t, providerConfig.Caches[0].Proxy)
		assert.Equal(t, ptr.To("storageClass"), providerConfig.Caches[0].Volume.StorageClassName)

		assert.Equal(t, "http://my-registry.io:5000", *providerConfig.Caches[1].RemoteURL)
		assert.Equal(t, ptr.To("http://proxy.io:5000"), providerConfig.Caches[1].Proxy.HTTPProxy)
		assert.Equal(t, ptr.To("https://proxy.io:5000"), providerConfig.Caches[1].Proxy.HTTPSProxy)
	})

	t.Run("should create registry cache extension", func(t *testing.T) {

	})
}
