package registrycache

import (
	"context"
	"encoding/json"
	"fmt"
	registrycacheext "github.com/gardener/gardener-extension-registry-cache/pkg/apis/registry/v1alpha3"
	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	v12 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"slices"
)

const SecretNameFmt = "reg-cache-%s"
const RuntimeSecretLabel = "kyma-project.io/runtime-id"
const CacheIDAnnotation = "kyma-project.io/registry-cache-id"
const CacheNameAnnotation = "kyma-project.io/registry-cache-name"
const CacheNamespaceAnnotation = "kyma-project.io/registry-cache-namespace"

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

func (s SecretSyncer) CreateOrUpdate(ctx context.Context, registryCaches []imv1.ImageRegistryCache) error {

	cachesWithSecret := getRegistryCachesWithSecret(registryCaches)

	for _, cache := range cachesWithSecret {
		var gardenerSecret v12.Secret
		err := s.GardenClient.Get(ctx, client.ObjectKey{Name: GetGardenSecretName(cache.UID), Namespace: s.GardenNamespace}, &gardenerSecret)

		if err != nil && !errors.IsNotFound(err) {
			return err
		}

		if err == nil {
			err = s.updateSecretInGardenCluster(ctx, cache, gardenerSecret)
			if err != nil {
				return fmt.Errorf("failed to update secret for registry cache %s: %w", cache.Name, err)
			}
		} else {
			err = s.copySecretFromRuntimeToGardenCluster(ctx, cache)
			if err != nil {
				return fmt.Errorf("failed to copy secret for registry cache %s: %w", cache.Name, err)
			}
		}
	}

	return nil
}

func (s SecretSyncer) copySecretFromRuntimeToGardenCluster(ctx context.Context, cacheConfig imv1.ImageRegistryCache) error {

	var secret v12.Secret
	err := s.RuntimeClient.Get(ctx, client.ObjectKey{Name: *cacheConfig.Config.SecretReferenceName, Namespace: cacheConfig.Namespace}, &secret)

	if err != nil {
		return err
	}

	newSecret := v12.Secret{
		ObjectMeta: v1.ObjectMeta{
			Name:      GetGardenSecretName(cacheConfig.UID),
			Namespace: s.GardenNamespace,
			Labels: map[string]string{
				RuntimeSecretLabel: s.RuntimeID,
			},
			Annotations: map[string]string{
				CacheIDAnnotation:        cacheConfig.UID,
				CacheNameAnnotation:      cacheConfig.Name,
				CacheNamespaceAnnotation: cacheConfig.Namespace,
			},
		},
		Immutable: ptr.To(true),
		Data:      secret.Data,
	}

	return s.GardenClient.Create(ctx, &newSecret)
}

func (s SecretSyncer) updateSecretInGardenCluster(ctx context.Context, cacheConfig imv1.ImageRegistryCache, gardenerSecret v12.Secret) error {
	var runtimeSecret v12.Secret
	err := s.RuntimeClient.Get(ctx, client.ObjectKey{Name: *cacheConfig.Config.SecretReferenceName, Namespace: cacheConfig.Namespace}, &runtimeSecret)

	if err != nil {
		return err
	}

	gardenerSecret.Data = runtimeSecret.Data

	return s.GardenClient.Update(ctx, &gardenerSecret)
}

func (s SecretSyncer) Delete(ctx context.Context, registryCaches []imv1.ImageRegistryCache) error {

	cachesWithSecret := getRegistryCachesWithSecret(registryCaches)

	var gardenSecrets v12.SecretList
	err := s.GardenClient.List(ctx, &gardenSecrets, client.MatchingLabels{RuntimeSecretLabel: s.RuntimeID}, client.InNamespace(s.GardenNamespace))

	if err != nil {
		return fmt.Errorf("failed to list garden secrets: %w", err)
	}

	for _, gardenSecret := range gardenSecrets.Items {
		registryCacheUID := gardenSecret.Annotations[CacheIDAnnotation]

		if registryCacheUID != "" && !registryCacheUidExists(registryCacheUID, cachesWithSecret) {
			err = s.GardenClient.Delete(ctx, &gardenSecret)

			if err != nil && !errors.IsNotFound(err) {
				return fmt.Errorf("failed to delete unused garden secret %s: %w", gardenSecret.Name, err)
			}
		}
	}

	return nil
}

func (s SecretSyncer) DeleteAll(ctx context.Context) error {

	var gardenSecrets v12.SecretList
	err := s.GardenClient.List(ctx, &gardenSecrets, client.MatchingLabels{RuntimeSecretLabel: s.RuntimeID}, client.InNamespace(s.GardenNamespace))

	if err != nil {
		return fmt.Errorf("failed to list garden secrets: %w", err)
	}

	for _, gardenSecret := range gardenSecrets.Items {
		err = s.GardenClient.Delete(ctx, &gardenSecret)

		if err != nil && !errors.IsNotFound(err) {
			return fmt.Errorf("failed to delete unused garden secret %s: %w", gardenSecret.Name, err)
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

func getRegistryCachesWithSecret(caches []imv1.ImageRegistryCache) []imv1.ImageRegistryCache {
	var cachesWithSecret []imv1.ImageRegistryCache
	for _, cache := range caches {
		if cache.Config.SecretReferenceName != nil && *cache.Config.SecretReferenceName != "" {
			cachesWithSecret = append(cachesWithSecret, cache)
		}
	}
	return cachesWithSecret
}

func GetGardenSecretName(uid string) string {
	return fmt.Sprintf(SecretNameFmt, uid)
}

func SecretMustBeRemoved(currentShoot *gardener.Shoot, runtime imv1.Runtime) (bool, error) {
	var registryCacheExt *gardener.Extension

	for _, ext := range currentShoot.Spec.Extensions {
		if ext.Type == "registry-cache" {
			if ext.Disabled != nil && *ext.Disabled {
				registryCacheExt = &ext
			}
		}
	}

	if registryCacheExt == nil {
		return false, nil
	}

	imageRegistryConfigWithSecrets := getRegistryCachesWithSecret(runtime.Spec.Caching)

	var registryConfig registrycacheext.RegistryConfig

	err := json.Unmarshal(registryCacheExt.ProviderConfig.Raw, &registryConfig)
	if err != nil {
		return false, fmt.Errorf("failed to unmarshal registry cache config: %w", err)
	}

	for _, cache := range registryConfig.Caches {
		secretNotReferencedInRuntimeCR := slices.ContainsFunc(imageRegistryConfigWithSecrets, func(c imv1.ImageRegistryCache) bool {
			return c.Config.SecretReferenceName == cache.SecretReferenceName
		})

		if !secretNotReferencedInRuntimeCR {
			return true, nil
		}
	}

	return false, nil
}
