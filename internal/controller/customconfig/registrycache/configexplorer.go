package registrycache

import (
	"context"
	registrycache "github.com/gardener/gardener-extension-registry-cache/pkg/apis/registry/v1alpha3"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ConfigExplorer struct {
	shootClient client.Client
	Context     context.Context
}

func NewConfigExplorer(ctx context.Context, kcpClient client.Client, runtime imv1.Runtime) (ConfigExplorer, error) {
	shootClient, err := gardener.GetShootClient(ctx, kcpClient, runtime)
	if err != nil {
		return ConfigExplorer{}, err
	}

	return ConfigExplorer{
		shootClient: shootClient,
		Context:     ctx,
	}, nil
}

func (c *ConfigExplorer) RegistryCacheConfigExists() (bool, error) {
	var customConfigList imv1.CustomConfigList
	err := c.shootClient.List(c.Context, &customConfigList)
	if err != nil {
		return false, err
	}

	for _, customConfig := range customConfigList.Items {
		if len(customConfig.Spec.RegistryCache) > 0 {
			return true, nil
		}
	}

	return false, nil
}

func (c *ConfigExplorer) GetRegistryCacheConfig() ([]registrycache.RegistryCache, error) {
	var customConfigList imv1.CustomConfigList
	err := c.shootClient.List(c.Context, &customConfigList)
	if err != nil {
		return nil, err
	}
	registryCacheConfigs := make([]registrycache.RegistryCache, 0)

	for _, customConfig := range customConfigList.Items {
		registryCacheConfigs = append(registryCacheConfigs, customConfig.Spec.RegistryCache...)
	}

	return registryCacheConfigs, nil
}
