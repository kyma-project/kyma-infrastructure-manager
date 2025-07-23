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

type SecretSyncer struct {
	SeedClient    client.Client
	RuntimeClient client.Client
	RuntimeID     string
}

func NewSecretSyncer(seedClient, runtimeClient client.Client, runtimeID string) SecretSyncer {
	return SecretSyncer{
		SeedClient:    seedClient,
		RuntimeClient: runtimeClient,
		RuntimeID:     runtimeID,
	}
}

func (s SecretSyncer) CreateOrUpdate(registryCaches []imv1.ImageRegistryCache) error {

	cachesWithSecret := getRegistryCachesWithSecret(registryCaches)

	for _, cache := range cachesWithSecret {
		var secret v12.Secret
		err := s.SeedClient.Get(context.TODO(), client.ObjectKey{Name: getSeedSecretName(cache), Namespace: cache.Namespace}, &secret)

		if err != nil && errors.IsNotFound(err) {
			err = s.copySecret(cache)
			if err != nil {
				return fmt.Errorf("failed to copy secret for registry cache %s: %w", cache.Name, err)
			}

			continue
		}

		if err != nil {
			return err
		}

		err = s.updateSecret(cache, secret)
		if err != nil {
			return fmt.Errorf("failed to update secret for registry cache %s: %w", cache.Name, err)
		}
	}

	return nil
}

func (s SecretSyncer) copySecret(cacheConfig imv1.ImageRegistryCache) error {

	var secret v12.Secret
	err := s.RuntimeClient.Get(context.TODO(), client.ObjectKey{Name: *cacheConfig.Config.SecretReferenceName, Namespace: cacheConfig.Namespace}, &secret)

	if err != nil {
		return err
	}

	newSecret := v12.Secret{
		ObjectMeta: v1.ObjectMeta{
			Name:      getSeedSecretName(cacheConfig),
			Namespace: secret.Namespace,
			Labels: map[string]string{
				RegistryCacheSecretLabel: s.RuntimeID,
			},
		},
		Data: secret.Data,
	}

	return s.SeedClient.Create(context.TODO(), &newSecret)
}

func (s SecretSyncer) updateSecret(cacheConfig imv1.ImageRegistryCache, gardenerSecret v12.Secret) error {
	var runtimeSecret v12.Secret
	err := s.RuntimeClient.Get(context.TODO(), client.ObjectKey{Name: *cacheConfig.Config.SecretReferenceName, Namespace: cacheConfig.Namespace}, &runtimeSecret)

	if err != nil {
		return err
	}

	gardenerSecret.Data = runtimeSecret.Data

	return s.SeedClient.Update(context.TODO(), &gardenerSecret)
}

func (s SecretSyncer) DeleteNotUsed(registryCaches []imv1.ImageRegistryCache) error {
	return nil
}

func (s SecretSyncer) getSecretsFromRuntime() ([]v12.Secret, error) {
	return nil, nil
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

func getSeedSecretName(registryCaches imv1.ImageRegistryCache) string {
	return fmt.Sprintf(RegistryCacheSecretNameFmt, registryCaches.UID)
}
