package extensions

import (
	"encoding/json"
	registrycache "github.com/gardener/gardener-extension-registry-cache/pkg/apis/registry/v1alpha3"
	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
)

const RegistryCacheExtensionType = "registry-cache"

func NewRegistryCacheExtension(cache []registrycache.RegistryCache, enabled bool) (*gardener.Extension, error) {
	return extension(cache, enabled)
}

func extension(caches []registrycache.RegistryCache, enabled bool) (*gardener.Extension, error) {

	registryConfig := registrycache.RegistryConfig{
		TypeMeta: v1.TypeMeta{
			APIVersion: "registry.extensions.gardener.cloud/v1alpha3",
			Kind:       "RegistryConfig",
		},
		Caches: caches,
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
		Disabled: ptr.To(enabled),
	}, nil
}
