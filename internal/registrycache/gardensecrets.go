package registrycache

import (
	"context"
	"encoding/json"
	"fmt"
	registrycacheext "github.com/gardener/gardener-extension-registry-cache/pkg/apis/registry/v1alpha3"
	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/extender/extensions"
	v12 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"slices"
)

const RuntimeSecretLabel = "kyma-project.io/runtime-id"
const CacheIDAnnotation = "kyma-project.io/registry-cache-id"
const CacheNameAnnotation = "kyma-project.io/registry-cache-name"
const CacheNamespaceAnnotation = "kyma-project.io/registry-cache-namespace"

type GardenSecretSyncer struct {
	GardenClient        client.Client
	RuntimeClient       client.Client
	RuntimeID           string
	GardenNamespace     string
	SecretNameGenerator SecretNameGenerator
}

type SecretNameGenerator func(string, string) string

func NewGardenSecretSyncer(gardenClient, runtimeClient client.Client, secretNameGenerator SecretNameGenerator, gardenNamespace, runtimeID string) GardenSecretSyncer {
	return GardenSecretSyncer{
		GardenClient:        gardenClient,
		RuntimeClient:       runtimeClient,
		RuntimeID:           runtimeID,
		GardenNamespace:     gardenNamespace,
		SecretNameGenerator: secretNameGenerator,
	}
}

func (s GardenSecretSyncer) CreateOrUpdate(ctx context.Context, registryCaches []imv1.ImageRegistryCache) error {

	cachesWithSecret := getRegistryCachesWithSecret(registryCaches)

	if len(cachesWithSecret) == 0 {
		return nil
	}

	var gardenSecrets v12.SecretList
	err := s.GardenClient.List(ctx, &gardenSecrets, client.MatchingLabels{RuntimeSecretLabel: s.RuntimeID}, client.InNamespace(s.GardenNamespace))

	if err != nil {
		return fmt.Errorf("failed to list garden secrets: %w", err)
	}

	for _, cache := range cachesWithSecret {
		gardenerSecret, err := findSecret(gardenSecrets.Items, cache.UID)

		if gardenerSecret == nil {
			err = s.copySecretFromRuntimeToGardenCluster(ctx, s.RuntimeID, cache)
			if err != nil {
				return fmt.Errorf("failed to copy secret for registry cache %s: %w", cache.Name, err)
			}
		} else {
			err = s.updateSecretInGardenCluster(ctx, cache, *gardenerSecret)
			if err != nil {
				return fmt.Errorf("failed to update secret for registry cache %s: %w", cache.Name, err)
			}
		}
	}

	return nil
}

func findSecret(secrets []v12.Secret, cacheUUID string) (*v12.Secret, error) {
	for _, secret := range secrets {
		if secret.Annotations[CacheIDAnnotation] == cacheUUID {
			return &secret, nil
		}
	}

	return nil, nil
}

func (s GardenSecretSyncer) copySecretFromRuntimeToGardenCluster(ctx context.Context, runtimeID string, cacheConfig imv1.ImageRegistryCache) error {

	var secret v12.Secret
	err := s.RuntimeClient.Get(ctx, client.ObjectKey{Name: *cacheConfig.Config.SecretReferenceName, Namespace: cacheConfig.Namespace}, &secret)

	if err != nil {
		return err
	}

	newSecret := v12.Secret{
		ObjectMeta: v1.ObjectMeta{
			Name:      s.SecretNameGenerator(runtimeID, cacheConfig.UID),
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

func (s GardenSecretSyncer) updateSecretInGardenCluster(ctx context.Context, cacheConfig imv1.ImageRegistryCache, gardenerSecret v12.Secret) error {
	var runtimeSecret v12.Secret
	err := s.RuntimeClient.Get(ctx, client.ObjectKey{Name: *cacheConfig.Config.SecretReferenceName, Namespace: cacheConfig.Namespace}, &runtimeSecret)

	if err != nil {
		return err
	}

	gardenerSecret.Data = runtimeSecret.Data

	return s.GardenClient.Update(ctx, &gardenerSecret)
}

func (s GardenSecretSyncer) Delete(ctx context.Context, registryCaches []imv1.ImageRegistryCache) error {

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

func (s GardenSecretSyncer) DeleteAll(ctx context.Context) error {

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

func GardenSecretNeedToBeRemoved(currentExtensions []gardener.Extension, desiredRegistryCacheConfig []imv1.ImageRegistryCache) (bool, error) {
	var registryCacheExt *gardener.Extension

	for _, ext := range currentExtensions {
		if ext.Type == extensions.RegistryCacheExtensionType {
			if ext.Disabled != nil && !*ext.Disabled {
				registryCacheExt = &ext
			}
		}
	}

	if registryCacheExt == nil {
		return false, nil
	}

	imageRegistryConfigWithSecrets := getRegistryCachesWithSecret(desiredRegistryCacheConfig)

	var registryConfig registrycacheext.RegistryConfig

	err := json.Unmarshal(registryCacheExt.ProviderConfig.Raw, &registryConfig)
	if err != nil {
		return false, fmt.Errorf("failed to unmarshal registry cache config: %w", err)
	}

	for _, cache := range registryConfig.Caches {
		if cache.SecretReferenceName == nil {
			continue
		}

		secretNotReferencedInRuntimeCR := slices.ContainsFunc(imageRegistryConfigWithSecrets, func(c imv1.ImageRegistryCache) bool {
			return *cache.SecretReferenceName == fmt.Sprintf(extensions.RegistryCacheSecretNameFmt, c.UID)
		})

		if !secretNotReferencedInRuntimeCR {
			return true, nil
		}
	}

	return false, nil
}
