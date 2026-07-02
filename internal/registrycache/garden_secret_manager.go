package registrycache

import (
	"context"
	"fmt"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	v12 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type GardenSecretManager struct {
	GardenClient    client.Client
	RuntimeID       string
	GardenNamespace string
}

func NewGardenSecretManager(gardenClient client.Client, gardenNamespace, runtimeID string) GardenSecretManager {
	return GardenSecretManager{
		GardenClient:    gardenClient,
		GardenNamespace: gardenNamespace,
		RuntimeID:       runtimeID,
	}
}

func (m GardenSecretManager) GetCacheUIDToSecretNameMap(ctx context.Context) (map[string]string, error) {
	var gardenSecrets v12.SecretList
	err := m.GardenClient.List(ctx, &gardenSecrets, client.MatchingLabels{RuntimeSecretLabel: m.RuntimeID, ManagedByLabel: ManagedByValue}, client.InNamespace(m.GardenNamespace))
	if err != nil {
		return nil, fmt.Errorf("failed to list garden secrets: %w", err)
	}

	result := make(map[string]string, len(gardenSecrets.Items))
	for _, secret := range gardenSecrets.Items {
		if secret.Labels[DirtyLabel] == "true" {
			continue
		}
		if cacheUID := secret.Labels[CacheIDLabel]; cacheUID != "" {
			result[cacheUID] = secret.Name
		}
	}
	return result, nil
}

func (m GardenSecretManager) Delete(ctx context.Context, registryCaches []imv1.ImageRegistryCache) error {
	cachesWithSecret := getRegistryCachesWithSecret(registryCaches)

	var gardenSecrets v12.SecretList
	err := m.GardenClient.List(ctx, &gardenSecrets, client.MatchingLabels{RuntimeSecretLabel: m.RuntimeID, ManagedByLabel: ManagedByValue}, client.InNamespace(m.GardenNamespace))
	if err != nil {
		return fmt.Errorf("failed to list garden secrets: %w", err)
	}

	for _, gardenSecret := range gardenSecrets.Items {
		registryCacheUID := gardenSecret.Labels[CacheIDLabel]

		if registryCacheUID != "" && !registryCacheUidExists(registryCacheUID, cachesWithSecret) {
			err = m.GardenClient.Delete(ctx, &gardenSecret)

			if err != nil && !errors.IsNotFound(err) {
				return fmt.Errorf("failed to delete unused garden secret %s: %w", gardenSecret.Name, err)
			}
		}
	}

	return nil
}

func (m GardenSecretManager) DeleteAll(ctx context.Context) error {
	var gardenSecrets v12.SecretList
	err := m.GardenClient.List(ctx, &gardenSecrets, client.MatchingLabels{RuntimeSecretLabel: m.RuntimeID, ManagedByLabel: ManagedByValue}, client.InNamespace(m.GardenNamespace))
	if err != nil {
		return fmt.Errorf("failed to list garden secrets: %w", err)
	}

	for _, gardenSecret := range gardenSecrets.Items {
		err = m.GardenClient.Delete(ctx, &gardenSecret)

		if err != nil && !errors.IsNotFound(err) {
			return fmt.Errorf("failed to delete garden secret %s: %w", gardenSecret.Name, err)
		}
	}

	return nil
}

func (m GardenSecretManager) DeleteDirty(ctx context.Context) error {
	var gardenSecrets v12.SecretList
	err := m.GardenClient.List(ctx, &gardenSecrets, client.MatchingLabels{RuntimeSecretLabel: m.RuntimeID, ManagedByLabel: ManagedByValue, DirtyLabel: "true"}, client.InNamespace(m.GardenNamespace))
	if err != nil {
		return fmt.Errorf("failed to list dirty garden secrets: %w", err)
	}

	for _, gardenSecret := range gardenSecrets.Items {
		err = m.GardenClient.Delete(ctx, &gardenSecret)

		if err != nil && !errors.IsNotFound(err) {
			return fmt.Errorf("failed to delete dirty garden secret %s: %w", gardenSecret.Name, err)
		}
	}

	return nil
}

func registryCacheUidExists(uid string, registryCaches []imv1.ImageRegistryCache) bool {
	for _, cache := range registryCaches {
		if cache.UID == uid {
			return true
		}
	}

	return false
}
