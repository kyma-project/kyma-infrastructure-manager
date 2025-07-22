package registrycache

import (
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	v12 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type SecretSyncer struct {
	SeedClient    client.Client
	RuntimeClient client.Client
}

func NewSecretSyncer(seedClient, runtimeClient client.Client) SecretSyncer {
	return SecretSyncer{
		SeedClient:    seedClient,
		RuntimeClient: runtimeClient,
	}
}

func (s SecretSyncer) CreateOrUpdate(registryCaches []imv1.ImageRegistryCache) error {
	return nil
}

func (s SecretSyncer) DeleteNotUsed(registryCaches []imv1.ImageRegistryCache) error {
	return nil
}

func (s SecretSyncer) getSecretsFromRuntime() ([]v12.Secret, error) {
	return nil, nil
}
