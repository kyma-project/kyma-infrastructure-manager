package registrycache

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	v12 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const RuntimeSecretLabel = "kyma-project.io/runtime-id"
const CacheIDLabel = "kyma-project.io/registry-cache-id"
const CacheNameAnnotation = "kyma-project.io/registry-cache-name"
const CacheNamespaceAnnotation = "kyma-project.io/registry-cache-namespace"
const ManagedByLabel = "app.kubernetes.io/managed-by"
const ManagedByValue = "infrastructure-manager"
const DirtyLabel = "kyma-project.io/dirty"

type SecretNameGenerator func(string, string) string

type GardenSecretSyncer struct {
	GardenClient        client.Client
	RuntimeClient       client.Client
	RuntimeID           string
	GardenNamespace     string
	SecretNameGenerator SecretNameGenerator
}

func NewGardenSecretSyncer(gardenClient, runtimeClient client.Client, secretNameGenerator SecretNameGenerator, gardenNamespace, runtimeID string) GardenSecretSyncer {
	return GardenSecretSyncer{
		GardenClient:        gardenClient,
		RuntimeClient:       runtimeClient,
		RuntimeID:           runtimeID,
		GardenNamespace:     gardenNamespace,
		SecretNameGenerator: secretNameGenerator,
	}
}

func (s GardenSecretSyncer) Do(ctx context.Context, registryCaches []imv1.ImageRegistryCache) error {
	cachesWithSecret := getRegistryCachesWithSecret(registryCaches)

	for _, cache := range cachesWithSecret {
		var gardenSecretsForCache v12.SecretList
		err := s.GardenClient.List(ctx, &gardenSecretsForCache,
			client.MatchingLabels{RuntimeSecretLabel: s.RuntimeID, CacheIDLabel: cache.UID},
			client.InNamespace(s.GardenNamespace))

		if err != nil {
			return fmt.Errorf("failed to list garden secrets: %w", err)
		}

		var gardenerSecret *v12.Secret
		if len(gardenSecretsForCache.Items) > 0 {
			gardenerSecret = &gardenSecretsForCache.Items[0]
		}

		if gardenerSecret == nil {
			err = s.copySecretFromRuntimeToGardenCluster(ctx, cache)
			if err != nil {
				return fmt.Errorf("failed to copy secret for registry cache %s: %w", cache.Name, err)
			}
		} else {
			err = s.replaceSecretInGardenCluster(ctx, cache, *gardenerSecret)
			if err != nil {
				return fmt.Errorf("failed to replace secret for registry cache %s: %w", cache.Name, err)
			}
		}
	}

	return nil
}

func (s GardenSecretSyncer) copySecretFromRuntimeToGardenCluster(ctx context.Context, cacheConfig imv1.ImageRegistryCache) error {
	var secret v12.Secret
	if err := s.RuntimeClient.Get(ctx, client.ObjectKey{Name: *cacheConfig.Config.SecretReferenceName, Namespace: cacheConfig.Namespace}, &secret); err != nil {
		return err
	}

	return s.GardenClient.Create(ctx, s.newGardenSecret(cacheConfig, secret.Data))
}

func (s GardenSecretSyncer) replaceSecretInGardenCluster(ctx context.Context, cacheConfig imv1.ImageRegistryCache, gardenerSecret v12.Secret) error {
	var runtimeSecret v12.Secret
	if err := s.RuntimeClient.Get(ctx, client.ObjectKey{Name: *cacheConfig.Config.SecretReferenceName, Namespace: cacheConfig.Namespace}, &runtimeSecret); err != nil {
		return err
	}

	patch := client.MergeFrom(gardenerSecret.DeepCopy())
	gardenerSecret.Labels[DirtyLabel] = "true"
	if err := s.GardenClient.Patch(ctx, &gardenerSecret, patch); err != nil && !errors.IsNotFound(err) {
		return err
	}

	return s.GardenClient.Create(ctx, s.newGardenSecret(cacheConfig, runtimeSecret.Data))
}

func (s GardenSecretSyncer) newGardenSecret(cacheConfig imv1.ImageRegistryCache, data map[string][]byte) *v12.Secret {
	return &v12.Secret{
		ObjectMeta: v1.ObjectMeta{
			Name:      s.SecretNameGenerator(s.RuntimeID, cacheConfig.UID),
			Namespace: s.GardenNamespace,
			Labels: map[string]string{
				RuntimeSecretLabel: s.RuntimeID,
				CacheIDLabel:       cacheConfig.UID,
				ManagedByLabel:     ManagedByValue,
			},
			Annotations: map[string]string{
				CacheNameAnnotation:      cacheConfig.Name,
				CacheNamespaceAnnotation: cacheConfig.Namespace,
			},
		},
		Immutable: ptr.To(true),
		Data:      data,
	}
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

// HasCachesWithSecrets reports whether any cache in the list has a secret reference configured.
func HasCachesWithSecrets(caches []imv1.ImageRegistryCache) bool {
	return len(getRegistryCachesWithSecret(caches)) > 0
}

func DefaultGardenSecretNameGenerator(_, _ string) string {
	return fmt.Sprintf("reg-cache-%s", uuid.New().String())
}
