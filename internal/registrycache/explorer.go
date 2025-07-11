package registrycache

import (
	"context"
	registrycache "github.com/kyma-project/kim-snatch/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// RuntimeConfigurationManager is responsible for managing runtime configurations (RegistryCacheConfig and corresponding Secrets).
type RuntimeConfigurationManager struct {
	runtimeClient client.Client
	Context       context.Context
}

type GetSecretFunc func() (corev1.Secret, error)

func NewRuntimeConfigurationManager(ctx context.Context, runtimeClient client.Client) *RuntimeConfigurationManager {
	return &RuntimeConfigurationManager{
		runtimeClient: runtimeClient,
		Context:       ctx,
	}
}

func (c *RuntimeConfigurationManager) RegistryCacheConfigExists() (bool, error) {
	registryCaches, err := c.GetRegistryCacheConfig()

	if err != nil {
		return false, err
	}

	return len(registryCaches) > 0, nil
}

func (c *RuntimeConfigurationManager) GetRegistryCacheConfig() ([]registrycache.RegistryCacheConfig, error) {
	var registryCacheConfigList registrycache.RegistryCacheConfigList
	err := c.runtimeClient.List(c.Context, &registryCacheConfigList)
	if err != nil {
		return nil, err
	}
	registryCacheConfigs := make([]registrycache.RegistryCacheConfig, 0)

	return append(registryCacheConfigs, registryCacheConfigList.Items...), nil
}

func (c *RuntimeConfigurationManager) CreateSecrets(secrets []corev1.Secret) error {
	for _, secret := range secrets {
		if err := c.runtimeClient.Create(c.Context, &secret); err != nil {
			return err
		}
	}
	return nil
}

func (c *RuntimeConfigurationManager) MarkSecretForDeletionUsingLabel(toBeMarked corev1.Secret) error {
	toBeMarked.Labels["kyma-project.io/registry-cache-config-marked-for-deletion"] = "true"

	if err := c.runtimeClient.Update(c.Context, &toBeMarked); err != nil {
		return err
	}
	return nil
}
