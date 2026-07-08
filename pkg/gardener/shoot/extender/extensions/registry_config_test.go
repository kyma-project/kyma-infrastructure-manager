package extensions

import (
	registrycacheext "github.com/gardener/gardener-extension-registry-cache/pkg/apis/registry/v1alpha3"
	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	registrycache "github.com/kyma-project/registry-cache/api/v1beta1"
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
		caches := []imv1.ImageRegistryCache{
			{
				Name:      "cache1",
				Namespace: "test",
				UID:       "id1",
				Config: registrycache.RegistryCacheConfigSpec{
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
			},
			{
				Config: registrycache.RegistryCacheConfigSpec{
					RemoteURL: ptr.To("http://my-registry.io:5000"),
					Proxy: &registrycache.Proxy{
						HTTPProxy:  ptr.To("http://proxy.io:5000"),
						HTTPSProxy: ptr.To("https://proxy.io:5000"),
					},
				},
			},
		}

		// when
		registryCacheExtension, err := NewRegistryCacheExtension(caches, map[string]string{"id1": "garden-name-1"}, nil)

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
		assert.Equal(t, "garden-name-1", *providerConfig.Caches[0].SecretReferenceName)
		assert.Nil(t, providerConfig.Caches[0].Proxy)
		assert.Equal(t, ptr.To("storageClass"), providerConfig.Caches[0].Volume.StorageClassName)

		assert.Equal(t, "http://my-registry.io:5000", *providerConfig.Caches[1].RemoteURL)
		assert.Equal(t, ptr.To("http://proxy.io:5000"), providerConfig.Caches[1].Proxy.HTTPProxy)
		assert.Equal(t, ptr.To("https://proxy.io:5000"), providerConfig.Caches[1].Proxy.HTTPSProxy)
	})

	t.Run("should return nil when no caches and no existing extension", func(t *testing.T) {

		// when
		registryCacheExtension, err := NewRegistryCacheExtension(nil, nil, nil)

		// then
		require.NoError(t, err)
		assert.Nil(t, registryCacheExtension)
	})

	t.Run("should return disabled extension when no caches but existing extension present", func(t *testing.T) {

		// given
		existingExt := &gardener.Extension{
			Type:     RegistryCacheExtensionType,
			Disabled: ptr.To(false),
		}

		// when
		registryCacheExtension, err := NewRegistryCacheExtension(nil, nil, existingExt)

		// then
		require.NoError(t, err)
		require.NotNil(t, registryCacheExtension)

		assert.Equal(t, RegistryCacheExtensionType, registryCacheExtension.Type)
		assert.Equal(t, ptr.To(true), registryCacheExtension.Disabled)
		assert.Nil(t, registryCacheExtension.ProviderConfig)
	})

	t.Run("should set nil volume when cache has no volume configured", func(t *testing.T) {

		// given
		caches := []imv1.ImageRegistryCache{
			{
				Config: registrycache.RegistryCacheConfigSpec{
					Upstream: "docker.io",
				},
			},
		}

		// when
		registryCacheExtension, err := NewRegistryCacheExtension(caches, nil, nil)

		// then
		require.NoError(t, err)
		require.NotNil(t, registryCacheExtension)

		var providerConfig registrycacheext.RegistryConfig
		err = yaml.Unmarshal(registryCacheExtension.ProviderConfig.Raw, &providerConfig)
		require.NoError(t, err)

		assert.Nil(t, providerConfig.Caches[0].Volume)
	})

	t.Run("should set nil garbage collection when cache has no garbage collection configured", func(t *testing.T) {

		// given
		caches := []imv1.ImageRegistryCache{
			{
				Config: registrycache.RegistryCacheConfigSpec{
					Upstream: "docker.io",
				},
			},
		}

		// when
		registryCacheExtension, err := NewRegistryCacheExtension(caches, nil, nil)

		// then
		require.NoError(t, err)
		require.NotNil(t, registryCacheExtension)

		var providerConfig registrycacheext.RegistryConfig
		err = yaml.Unmarshal(registryCacheExtension.ProviderConfig.Raw, &providerConfig)
		require.NoError(t, err)

		assert.Nil(t, providerConfig.Caches[0].GarbageCollection)
	})

	t.Run("should set nil secret reference when SecretReferenceName is nil", func(t *testing.T) {

		// given
		caches := []imv1.ImageRegistryCache{
			{
				UID: "id1",
				Config: registrycache.RegistryCacheConfigSpec{
					Upstream:            "docker.io",
					SecretReferenceName: nil,
				},
			},
		}

		// when
		registryCacheExtension, err := NewRegistryCacheExtension(caches, map[string]string{"id1": "garden-name-1"}, nil)

		// then
		require.NoError(t, err)
		require.NotNil(t, registryCacheExtension)

		var providerConfig registrycacheext.RegistryConfig
		err = yaml.Unmarshal(registryCacheExtension.ProviderConfig.Raw, &providerConfig)
		require.NoError(t, err)

		assert.Nil(t, providerConfig.Caches[0].SecretReferenceName)
	})

	t.Run("should set nil secret reference when SecretReferenceName is empty string", func(t *testing.T) {

		// given
		caches := []imv1.ImageRegistryCache{
			{
				UID: "id1",
				Config: registrycache.RegistryCacheConfigSpec{
					Upstream:            "docker.io",
					SecretReferenceName: ptr.To(""),
				},
			},
		}

		// when
		registryCacheExtension, err := NewRegistryCacheExtension(caches, map[string]string{"id1": "garden-name-1"}, nil)

		// then
		require.NoError(t, err)
		require.NotNil(t, registryCacheExtension)

		var providerConfig registrycacheext.RegistryConfig
		err = yaml.Unmarshal(registryCacheExtension.ProviderConfig.Raw, &providerConfig)
		require.NoError(t, err)

		assert.Nil(t, providerConfig.Caches[0].SecretReferenceName)
	})
}
