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
	registryCaches, err := c.GetRegistryCacheConfig()

	if err != nil {
		return false, err
	}

	return len(registryCaches) > 0, nil
}

func (c *ConfigExplorer) GetRegistryCacheConfig() ([]registrycache.RegistryCacheConfig, error) {
	var customConfigList registrycache.RegistryCacheConfigList
	err := c.runtimeClient.List(c.Context, &customConfigList)
	if err != nil {
		return nil, err
	}
	registryCacheConfigs := make([]registrycache.RegistryCacheConfig, 0)

	return append(registryCacheConfigs, customConfigList.Items...), nil
}
