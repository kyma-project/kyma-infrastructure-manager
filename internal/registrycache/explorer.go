package registrycache

import (
	"context"
	registrycache "github.com/kyma-project/kim-snatch/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ConfigExplorer struct {
	runtimeClient client.Client
	Context       context.Context
}

type GetSecretFunc func() (corev1.Secret, error)

func NewConfigExplorer(ctx context.Context, runtimeClient client.Client) *ConfigExplorer {
	return &ConfigExplorer{
		runtimeClient: runtimeClient,
		Context:       ctx,
	}
}

func (c *ConfigExplorer) RegistryCacheConfigExists() (bool, error) {
	var customConfigList registrycache.CustomConfigList
	err := c.runtimeClient.List(c.Context, &customConfigList)
	if err != nil {
		return false, err
	}

	for _, customConfig := range customConfigList.Items {
		if len(customConfig.Spec.RegistryCaches) > 0 {
			return true, nil
		}
	}

	return false, nil
}

func (c *ConfigExplorer) GetRegistryCacheConfig() ([]registrycache.RegistryCache, error) {
	var customConfigList registrycache.CustomConfigList
	err := c.runtimeClient.List(c.Context, &customConfigList)
	if err != nil {
		return nil, err
	}
	registryCacheConfigs := make([]registrycache.RegistryCache, 0)

	for _, customConfig := range customConfigList.Items {
		registryCacheConfigs = append(registryCacheConfigs, customConfig.Spec.RegistryCaches...)
	}

	return registryCacheConfigs, nil
}
