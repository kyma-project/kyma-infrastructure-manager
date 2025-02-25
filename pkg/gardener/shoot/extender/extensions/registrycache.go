package extensions

import (
	registrycache "github.com/gardener/gardener-extension-registry-cache/pkg/apis/registry/v1alpha3"
	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/yaml"
)

const RegistryCacheExtensionType = "registry-cache"

func NewRegistryCacheExtension(cache []registrycache.RegistryCache) (*gardener.Extension, error) {

	if len(cache) > 0 {
		return extensionEnabled(cache)
	}

	return extensionDisabled()
}

func extensionEnabled(cache []registrycache.RegistryCache) (*gardener.Extension, error) {
	providerConfigBytes, err := yaml.Marshal(cache)
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

func extensionDisabled() (*gardener.Extension, error) {
	return &gardener.Extension{
		Type:     RegistryCacheExtensionType,
		Disabled: ptr.To(true),
	}, nil
}
