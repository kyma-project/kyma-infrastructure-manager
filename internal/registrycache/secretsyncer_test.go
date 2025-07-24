package registrycache

import (
	"context"
	"fmt"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	registrycache "github.com/kyma-project/kim-snatch/api/v1beta1"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

func TestSecretSyncer(t *testing.T) {
	RegisterTestingT(t)

	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	ctx := context.Background()

	t.Run("Should create not existing secrets", func(t *testing.T) {
		// given
		runtimeID := "test-runtime-id"
		gardenNamespace := "garden-dev"

		secret1 := fixRegistryCacheSecret("secret1", "test", map[string]string{}, map[string]string{}, "user1", "password1")
		secret2 := fixRegistryCacheSecret("secret2", "default", map[string]string{}, map[string]string{}, "user2", "password2")

		runtimeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			fixRegistryCacheSecret("orphaned-secret1", "test", map[string]string{}, map[string]string{}, "user", "password"),
			secret1,
			secret2,
		).Build()

		gardenClient := fake.NewClientBuilder().WithScheme(scheme).Build()

		registryCacheWithSecret1 := fixRegistryCacheConfigWithSecret("config-with-secret-1", secret1.Namespace, "id1", "quay.io", secret1.Name)
		registryCacheWithSecret2 := fixRegistryCacheConfigWithSecret("config-with-secret-2", secret2.Namespace, "id2", "gcr.io", secret2.Name)

		registryCacheConfigs := []imv1.ImageRegistryCache{
			fixRegistryCacheConfigWithoutSecret("config-without-secret-1", "test", "id1", "docker.io"),
			registryCacheWithSecret1,
			registryCacheWithSecret2,
		}

		// when
		secretSyncer := NewSecretSyncer(gardenClient, runtimeClient, gardenNamespace, runtimeID)
		err := secretSyncer.CreateOrUpdate(context.Background(), registryCacheConfigs)

		// then
		Expect(err).To(BeNil())

		secrets, err := getGardenSecrets(ctx, gardenClient, runtimeID)
		Expect(err).To(BeNil())

		Expect(len(secrets)).To(Equal(2))

		gardenerSecret1, err := getGardenSecret(ctx, gardenClient, fmt.Sprintf(SecretNameFmt, registryCacheWithSecret1.UID), gardenNamespace)
		Expect(err).To(BeNil())
		Expect(gardenerSecret1).To(Not(BeNil()))

		Expect(gardenerSecret1.Labels[RuntimeSecretLabel]).To(Equal(runtimeID))
		Expect(gardenerSecret1.Annotations[CacheIDAnnotation]).To(Equal(registryCacheWithSecret1.UID))

		Expect(gardenerSecret1.Data).To(Equal(secret1.Data))

		gardenerSecret2, err := getGardenSecret(ctx, gardenClient, fmt.Sprintf(SecretNameFmt, registryCacheWithSecret2.UID), gardenNamespace)
		Expect(err).To(BeNil())
		Expect(gardenerSecret2).To(Not(BeNil()))

		Expect(gardenerSecret2.Labels[RuntimeSecretLabel]).To(Equal(runtimeID))
		Expect(gardenerSecret2.Annotations[CacheIDAnnotation]).To(Equal(registryCacheWithSecret2.UID))
		Expect(gardenerSecret2.Data).To(Equal(secret2.Data))
	})

	t.Run("Should update existing secrets", func(t *testing.T) {
		// given
		runtimeID := "test-runtime-id"
		gardenNamespace := "garden-dev"
		secret1 := fixRegistryCacheSecret("secret1", "test", map[string]string{}, map[string]string{}, "newuser", "newpassword")
		secret2 := fixRegistryCacheSecret("secret2", "default", map[string]string{}, map[string]string{}, "user2", "password2")

		runtimeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			secret1,
			secret2,
		).Build()

		registryCacheWithSecret1 := fixRegistryCacheConfigWithSecret("config-with-secret-1", secret1.Namespace, "id1", "quay.io", secret1.Name)
		registryCacheWithSecret2 := fixRegistryCacheConfigWithSecret("config-with-secret-2", secret2.Namespace, "id2", "gcr.io", secret2.Name)

		registryCacheConfigs := []imv1.ImageRegistryCache{
			fixRegistryCacheConfigWithoutSecret("config-without-secret-1", "test", "id1", "docker.io"),
			registryCacheWithSecret1,
			registryCacheWithSecret2,
		}
		labels1 := fixRegistryCacheGardenSecretLabels(runtimeID)
		labels2 := fixRegistryCacheGardenSecretLabels(runtimeID)
		annotations1 := fixRegistryCacheGardenSecretLabels(registryCacheWithSecret1.UID)
		annotations2 := fixRegistryCacheGardenSecretLabels(registryCacheWithSecret1.UID)

		gardenerSecret1 := fixRegistryCacheSecret(GetGardenSecretName(registryCacheWithSecret1.UID), gardenNamespace, labels1, annotations1, "user1", "password1")
		gardenerSecret2 := fixRegistryCacheSecret(GetGardenSecretName(registryCacheWithSecret2.UID), gardenNamespace, labels2, annotations2, "user2", "password2")

		gardenClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			gardenerSecret1,
			gardenerSecret2,
		).Build()

		// when
		secretSyncer := NewSecretSyncer(gardenClient, runtimeClient, gardenNamespace, runtimeID)
		err := secretSyncer.CreateOrUpdate(context.Background(), registryCacheConfigs)

		// then
		Expect(err).To(BeNil())

		secrets, err := getGardenSecrets(ctx, gardenClient, runtimeID)
		Expect(err).To(BeNil())

		Expect(len(secrets)).To(Equal(2))

		updatedGardenerSecret1, err := getGardenSecret(ctx, gardenClient, fmt.Sprintf(SecretNameFmt, registryCacheWithSecret1.UID), gardenNamespace)
		Expect(err).To(BeNil())
		Expect(updatedGardenerSecret1).To(Not(BeNil()))

		Expect(updatedGardenerSecret1.Labels[RuntimeSecretLabel]).To(Equal(runtimeID))
		Expect(updatedGardenerSecret1.Data).To(Equal(secret1.Data))

		updatedGardenerSecret2, err := getGardenSecret(ctx, gardenClient, fmt.Sprintf(SecretNameFmt, registryCacheWithSecret2.UID), gardenNamespace)
		Expect(err).To(BeNil())
		Expect(updatedGardenerSecret2).To(Not(BeNil()))

		Expect(updatedGardenerSecret2.Labels[RuntimeSecretLabel]).To(Equal(runtimeID))
		Expect(updatedGardenerSecret2.Data).To(Equal(secret2.Data))
	})

	t.Run("Should remove unneeded secrets", func(t *testing.T) {
		// given
		runtimeID := "test-runtime-id"
		gardenNamespace := "garden-dev"

		secret1 := fixRegistryCacheSecret("secret1", "test", map[string]string{}, map[string]string{}, "user1", "password1")
		secret3 := fixRegistryCacheSecret("secret3", "default", map[string]string{}, map[string]string{}, "user3", "password3")

		runtimeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			secret1,
			secret3,
		).Build()

		registryCacheWithSecret1 := fixRegistryCacheConfigWithSecret("config-with-secret-1", secret1.Namespace, "id1", "quay.io", secret1.Name)
		registryCacheWithSecret3 := fixRegistryCacheConfigWithSecret("config-with-secret-3", secret3.Namespace, "id3", "gcr.io", secret3.Name)

		registryCacheConfigs := []imv1.ImageRegistryCache{
			fixRegistryCacheConfigWithoutSecret("config-without-secret-1", "test", "id1", "docker.io"),
			registryCacheWithSecret1,
			registryCacheWithSecret3,
		}

		labels1 := fixRegistryCacheGardenSecretLabels(runtimeID)
		labels2 := fixRegistryCacheGardenSecretLabels(runtimeID)
		annotations1 := fixRegistryCacheGardenSecretAnnotations(registryCacheWithSecret1.UID)
		annotations2 := fixRegistryCacheGardenSecretAnnotations("id2")

		gardenerSecret1 := fixRegistryCacheSecret(GetGardenSecretName(registryCacheWithSecret1.UID), gardenNamespace, labels1, annotations1, "user1", "password1")
		gardenerSecret2 := fixRegistryCacheSecret("reg-cache-id", gardenNamespace, labels2, annotations2, "user2", "password2")

		gardenClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			gardenerSecret1,
			gardenerSecret2,
		).Build()

		// when
		secretSyncer := NewSecretSyncer(gardenClient, runtimeClient, gardenNamespace, runtimeID)
		err := secretSyncer.Delete(context.Background(), registryCacheConfigs)

		// then
		Expect(err).To(BeNil())

		secrets, err := getGardenSecrets(ctx, gardenClient, runtimeID)
		Expect(err).To(BeNil())

		Expect(len(secrets)).To(Equal(1))

		updatedGardenerSecret1, err := getGardenSecret(ctx, gardenClient, fmt.Sprintf(SecretNameFmt, registryCacheWithSecret1.UID), gardenNamespace)
		Expect(err).To(BeNil())
		Expect(updatedGardenerSecret1).To(Not(BeNil()))
	})
}

func fixRegistryCacheSecret(name, namespace string, labels map[string]string, annotations map[string]string, user string, password string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Labels:      labels,
			Annotations: annotations,
		},
		Data: map[string][]byte{
			"username": []byte(user),
			"password": []byte(password)},
	}
}

func fixRegistryCacheGardenSecretLabels(runtimeID string) map[string]string {
	return map[string]string{
		RuntimeSecretLabel: runtimeID,
	}
}

func fixRegistryCacheGardenSecretAnnotations(registryCacheID string) map[string]string {
	return map[string]string{
		CacheIDAnnotation: registryCacheID,
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

func getGardenSecret(ctx context.Context, gardenClient client.Client, name, namespace string) (*corev1.Secret, error) {
	secret := &corev1.Secret{}
	err := gardenClient.Get(ctx, types.NamespacedName{
		Name:      name,
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
