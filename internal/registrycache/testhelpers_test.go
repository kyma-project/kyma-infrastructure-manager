package registrycache

import (
	"context"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	registrycache "github.com/kyma-project/registry-cache/api/v1beta1"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func verifyGardenSecret(gardenSecret, registryCacheSecret *corev1.Secret, registryCache imv1.ImageRegistryCache, runtimeID string) {
	Expect(gardenSecret.Labels[RuntimeSecretLabel]).To(Equal(runtimeID))
	Expect(gardenSecret.Labels[CacheIDLabel]).To(Equal(registryCache.UID))
	Expect(gardenSecret.Annotations[CacheNameAnnotation]).To(Equal(registryCache.Name))
	Expect(gardenSecret.Annotations[CacheNamespaceAnnotation]).To(Equal(registryCache.Namespace))

	Expect(gardenSecret.Data).To(Equal(registryCacheSecret.Data))
	Expect(*gardenSecret.Immutable).To(Equal(true))
}

func fixRegistryCacheSecret(name, namespace string, labels map[string]string, annotations map[string]string, user string, password string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Labels:      labels,
			Annotations: annotations,
		},
		Immutable: ptr.To(true),
		Data: map[string][]byte{
			"username": []byte(user),
			"password": []byte(password)},
	}
}

func fixRegistryCacheGardenSecretLabels(runtimeID, cacheID string) map[string]string {
	return map[string]string{
		RuntimeSecretLabel: runtimeID,
		CacheIDLabel:       cacheID,
	}
}

func fixRegistryCacheGardenSecretAnnotations(cacheName, cacheNamespace string) map[string]string {
	return map[string]string{
		CacheNameAnnotation:      cacheName,
		CacheNamespaceAnnotation: cacheNamespace,
	}
}

func fixRegistryCacheConfigWithSecret(name, namespace, uuid, upstream, secretName string) imv1.ImageRegistryCache {
	return imv1.ImageRegistryCache{
		Name:      name,
		Namespace: namespace,
		UID:       uuid,
		Config: registrycache.RegistryCacheConfigSpec{
			Upstream:            upstream,
			SecretReferenceName: ptr.To(secretName),
		},
	}
}

func fixRegistryCacheConfigWithoutSecret(name, namespace, uuid, upstream string) imv1.ImageRegistryCache {
	return imv1.ImageRegistryCache{
		Name:      name,
		Namespace: namespace,
		UID:       uuid,
		Config: registrycache.RegistryCacheConfigSpec{
			Upstream: upstream,
		},
	}
}

func fixSecretNameGenerator() SecretNameGenerator {
	return func(runtimeID, uuid string) string {
		return runtimeID + "-" + uuid
	}
}

func getGardenSecret(ctx context.Context, gardenClient client.Client, runtimeID, uuid, namespace string) (*corev1.Secret, error) {
	secret := &corev1.Secret{}
	err := gardenClient.Get(ctx, types.NamespacedName{
		Name:      fixSecretNameGenerator()(runtimeID, uuid),
		Namespace: namespace,
	}, secret)

	return secret, err
}

func getGardenSecrets(ctx context.Context, gardenClient client.Client, runtimeID string) ([]corev1.Secret, error) {
	secretList := &corev1.SecretList{}
	err := gardenClient.List(ctx, secretList, client.MatchingLabels{RuntimeSecretLabel: runtimeID})
	if err != nil {
		return nil, err
	}

	return secretList.Items, nil
}
