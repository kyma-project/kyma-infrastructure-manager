package registrycache

import (
	"context"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener"
	registrycache "github.com/kyma-project/kim-snatch/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ConfigExplorer struct {
	runtimeClient client.Client
	Context       context.Context
}

type GetSecretFunc func() (corev1.Secret, error)

func NewConfigExplorer(ctx context.Context, kubeconfigSecret corev1.Secret) (*ConfigExplorer, error) {

	runtimeClient, err := gardener.GetRuntimeClient(kubeconfigSecret)
	if err != nil {
		return nil, err
	}

	return &ConfigExplorer{
		runtimeClient: runtimeClient,
		Context:       ctx,
	}, nil
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
