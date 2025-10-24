package extensions

import (
	"encoding/json"
	"fmt"
	registrycacheext "github.com/gardener/gardener-extension-registry-cache/pkg/apis/registry/v1alpha3"
	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	registrycache "github.com/kyma-project/registry-cache/api/v1beta1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
)

const RegistryCacheExtensionType = "registry-cache"
const RegistryCacheSecretPrefix = "reg-cache-"
const RegistryCacheSecretNameFmt = RegistryCacheSecretPrefix + "%s"

func NewRegistryCacheExtension(caches []imv1.ImageRegistryCache, existingRegistryCacheExt *gardener.Extension) (*gardener.Extension, error) {
	if len(caches) > 0 {
		return extension(caches)
	}

	if existingRegistryCacheExt != nil {
		return disabledExtension(existingRegistryCacheExt)
	}

	return nil, nil
}

func extension(caches []imv1.ImageRegistryCache) (*gardener.Extension, error) {

	registryConfig := registrycacheext.RegistryConfig{
		TypeMeta: v1.TypeMeta{
			APIVersion: "registry.extensions.gardener.cloud/v1alpha3",
			Kind:       "RegistryConfig",
		},
		Caches: ToRegistryCacheExtension(caches),
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
		Disabled: ptr.To(false),
	}, nil
}

func disabledExtension(existingRegistryCacheExt *gardener.Extension) (*gardener.Extension, error) {
	var providerConfig registrycacheext.RegistryConfig

	err := json.Unmarshal(existingRegistryCacheExt.ProviderConfig.Raw, &providerConfig)

	if err != nil {
		return nil, err
	}

	for i := 0; i < len(providerConfig.Caches); i++ {
		providerConfig.Caches[i].SecretReferenceName = nil
	}

	providerConfigBytes, err := json.Marshal(providerConfig)
	if err != nil {
		return nil, err
	}

	// In case the extension is configured, and the user removes registry cache we disable the extension.
	// In order to be able to remove registry credentials secret the reference to the secret name needs to be removed.
	return &gardener.Extension{
		Type: RegistryCacheExtensionType,
		ProviderConfig: &runtime.RawExtension{
			Raw: providerConfigBytes,
		},
		Disabled: ptr.To(true),
	}, nil
}

func ToRegistryCacheExtension(caches []imv1.ImageRegistryCache) []registrycacheext.RegistryCache {

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
	for _, c := range caches {
		registryCaches = append(registryCaches, registrycacheext.RegistryCache{
			Upstream:            c.Config.Upstream,
			RemoteURL:           c.Config.RemoteURL,
			Volume:              volumeToCacheExtension(c.Config.Volume),
			GarbageCollection:   garbageCollectionExtension(c.Config.GarbageCollection),
			SecretReferenceName: ptr.To(fmt.Sprintf(RegistryCacheSecretNameFmt, c.UID)),
			Proxy:               proxyExtension(c.Config.Proxy),
		})
	}
	return registryCaches
}
