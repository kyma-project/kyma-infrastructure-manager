package registrycache

import (
	"encoding/json"
	"fmt"
	registrycacheext "github.com/gardener/gardener-extension-registry-cache/pkg/apis/registry/v1alpha3"
	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/extender/extensions"
)

func HasRegistryCacheCountChanged(currentExtensions []gardener.Extension, desiredRegistryCacheConfig []imv1.ImageRegistryCache) (bool, error) {
	var registryCacheExt *gardener.Extension

	for _, ext := range currentExtensions {
		if ext.Type == extensions.RegistryCacheExtensionType {
			if ext.Disabled != nil && !*ext.Disabled {
				registryCacheExt = &ext
			}
		}
	}

	if registryCacheExt == nil {
		return false, nil
	}

	var registryConfig registrycacheext.RegistryConfig

	err := json.Unmarshal(registryCacheExt.ProviderConfig.Raw, &registryConfig)
	if err != nil {
		return false, fmt.Errorf("failed to unmarshal registry cache config: %w", err)
	}

	return len(desiredRegistryCacheConfig) != len(registryConfig.Caches), nil
}
