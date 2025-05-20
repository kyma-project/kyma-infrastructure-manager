package extensions

import (
	"encoding/json"
	registrycacheext "github.com/gardener/gardener-extension-registry-cache/pkg/apis/registry/v1alpha3"
	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	registrycache "github.com/kyma-project/kim-snatch/api/v1beta1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
)

const RegistryCacheExtensionType = "registry-cache"

func NewRegistryCacheExtension(cache []registrycache.RegistryCache, enabled bool) (*gardener.Extension, error) {
	return extension(cache, enabled)
}

func extension(caches []registrycache.RegistryCache, enabled bool) (*gardener.Extension, error) {

	registryConfig := registrycacheext.RegistryConfig{
		TypeMeta: v1.TypeMeta{
			APIVersion: "registry.extensions.gardener.cloud/v1alpha3",
			Kind:       "RegistryConfig",
		},
		Caches: toRegistryCacheExtension(caches),
	}

	providerConfigBytes, err := json.Marshal(registryConfig)
	if err != nil {
		return nil, err
	}

	return &gardener.Extension{
		Type: RegistryCacheExtensionType,
		ProviderConfig: &runtime.RawExtension{
			Raw: providerConfigBytes,
		},
		Disabled: ptr.To(!enabled),
	}, nil
}

func toRegistryCacheExtension(cache []registrycache.RegistryCache) []registrycacheext.RegistryCache {

	volumeToCacheExtension := func(volume *registrycache.Volume) *registrycacheext.Volume {

		if volume == nil {
			return nil
		}

		return &registrycacheext.Volume{
			Size:             volume.Size,
			StorageClassName: volume.StorageClassName,
		}
	}

	garbageCollectionExtension := func(garbageCollection *registrycache.GarbageCollection) *registrycacheext.GarbageCollection {
		if garbageCollection == nil {
			return nil
		}

		return &registrycacheext.GarbageCollection{
			TTL: garbageCollection.TTL,
		}
	}

	proxyExtension := func(proxy *registrycache.Proxy) *registrycacheext.Proxy {
		if proxy == nil {
			return nil
		}

		return &registrycacheext.Proxy{
			HTTPProxy:  proxy.HTTPProxy,
			HTTPSProxy: proxy.HTTPSProxy,
		}
	}

	// Convert the registry cache to the internal format
	registryCaches := make([]registrycacheext.RegistryCache, 0)
	for _, c := range cache {
		registryCaches = append(registryCaches, registrycacheext.RegistryCache{
			Upstream:            c.Upstream,
			RemoteURL:           c.RemoteURL,
			Volume:              volumeToCacheExtension(c.Volume),
			GarbageCollection:   garbageCollectionExtension(c.GarbageCollection),
			SecretReferenceName: c.SecretReferenceName,
			Proxy:               proxyExtension(c.Proxy),
		})
	}
	return registryCaches
}
