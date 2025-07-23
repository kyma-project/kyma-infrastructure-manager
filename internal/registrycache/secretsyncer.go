package registrycache

import (
	"context"
	"fmt"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	v12 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const RegistryCacheSecretNameFmt = "reg-cache-%s"
const RegistryCacheSecretLabel = "kyma-project.io/runtime-id"
const RegistryCacheIDLabel = "kyma-project.io/registry-cache-id"

type SecretSyncer struct {
	GardenClient    client.Client
	RuntimeClient   client.Client
	RuntimeID       string
	GardenNamespace string
}

func NewSecretSyncer(gardenClient, runtimeClient client.Client, gardenNamespace, runtimeID string) SecretSyncer {
	return SecretSyncer{
		GardenClient:    gardenClient,
		RuntimeClient:   runtimeClient,
		RuntimeID:       runtimeID,
		GardenNamespace: gardenNamespace,
	}
}

func (s SecretSyncer) CreateOrUpdate(registryCaches []imv1.ImageRegistryCache) error {

	cachesWithSecret := getRegistryCachesWithSecret(registryCaches)

	for _, cache := range cachesWithSecret {
		var gardenerSecret v12.Secret
		err := s.GardenClient.Get(context.TODO(), client.ObjectKey{Name: getGardenSecretName(cache), Namespace: s.GardenNamespace}, &gardenerSecret)

		if err != nil && !errors.IsNotFound(err) {
			return err
		}

		if err == nil {
			err = s.updateSecretInGardenCluster(cache, gardenerSecret)
			if err != nil {
				return fmt.Errorf("failed to update secret for registry cache %s: %w", cache.Name, err)
			}
		} else {
			err = s.copySecretFromRuntimeToGardenCluster(cache)
			if err != nil {
				return fmt.Errorf("failed to copy secret for registry cache %s: %w", cache.Name, err)
			}
		}
	}

	return nil
}

func (s SecretSyncer) copySecretFromRuntimeToGardenCluster(cacheConfig imv1.ImageRegistryCache) error {

	var secret v12.Secret
	err := s.RuntimeClient.Get(context.TODO(), client.ObjectKey{Name: *cacheConfig.Config.SecretReferenceName, Namespace: cacheConfig.Namespace}, &secret)

	if err != nil {
		return err
	}

	newSecret := v12.Secret{
		ObjectMeta: v1.ObjectMeta{
			Name:      getGardenSecretName(cacheConfig),
			Namespace: s.GardenNamespace,
			Labels: map[string]string{
				RegistryCacheSecretLabel: s.RuntimeID,
				RegistryCacheIDLabel:     cacheConfig.UID,
			},
		},
		Data: secret.Data,
	}

	return s.GardenClient.Create(context.TODO(), &newSecret)
}

func (s SecretSyncer) updateSecretInGardenCluster(cacheConfig imv1.ImageRegistryCache, gardenerSecret v12.Secret) error {
	var runtimeSecret v12.Secret
	err := s.RuntimeClient.Get(context.TODO(), client.ObjectKey{Name: *cacheConfig.Config.SecretReferenceName, Namespace: cacheConfig.Namespace}, &runtimeSecret)

	if err != nil {
		return err
	}

	gardenerSecret.Data = runtimeSecret.Data

	return s.GardenClient.Update(context.TODO(), &gardenerSecret)
}

func (s SecretSyncer) DeleteNotUsed(registryCaches []imv1.ImageRegistryCache) error {

	cachesWithSecret := getRegistryCachesWithSecret(registryCaches)

	var gardenSecrets v12.SecretList
	err := s.GardenClient.List(context.TODO(), &gardenSecrets, client.MatchingLabels{RegistryCacheSecretLabel: s.RuntimeID})

	if err != nil {
		return fmt.Errorf("failed to list garden secrets: %w", err)
	}

	for _, gardenSecret := range gardenSecrets.Items {
		id := gardenSecret.Labels[RegistryCacheIDLabel]

		if id == "" {
			continue
		}

		found := false
		for _, cache := range cachesWithSecret {
			if cache.UID == id {
				found = true
			}
		}

		if !found {
			err = s.GardenClient.Delete(context.TODO(), &gardenSecret)

			if err != nil {
				return fmt.Errorf("failed to delete unused garden secret %s: %w", gardenSecret.Name, err)
			}
		}
	}

	return nil
}

func getRegistryCachesWithSecret(caches []imv1.ImageRegistryCache) []imv1.ImageRegistryCache {
	var cachesWithSecret []imv1.ImageRegistryCache
	for _, cache := range caches {
		if cache.Config.SecretReferenceName != nil && *cache.Config.SecretReferenceName != "" {
			cachesWithSecret = append(cachesWithSecret, cache)
		}
	}
	return cachesWithSecret
}

func getGardenSecretName(registryCaches imv1.ImageRegistryCache) string {
	return fmt.Sprintf(RegistryCacheSecretNameFmt, registryCaches.UID)
}
