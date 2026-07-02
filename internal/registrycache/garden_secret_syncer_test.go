package registrycache

import (
	"context"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

func TestGardenSecretSyncer(t *testing.T) {
	RegisterTestingT(t)

	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	ctx := context.Background()

	t.Run("Should create not existing secrets", func(t *testing.T) {
		// given
		runtimeID := "test-runtime-id-1"
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
		secretSyncer := NewGardenSecretSyncer(gardenClient, runtimeClient, fixSecretNameGenerator(), gardenNamespace, runtimeID)
		err := secretSyncer.Do(context.Background(), registryCacheConfigs)

		// then
		Expect(err).To(BeNil())

		secrets, err := getGardenSecrets(ctx, gardenClient, runtimeID)
		Expect(err).To(BeNil())
		Expect(len(secrets)).To(Equal(2))

		gardenSecret1, err := getGardenSecret(ctx, gardenClient, runtimeID, registryCacheWithSecret1.UID, gardenNamespace)
		Expect(err).To(BeNil())
		Expect(gardenSecret1).To(Not(BeNil()))
		verifyGardenSecret(gardenSecret1, secret1, registryCacheWithSecret1, runtimeID)

		gardenerSecret2, err := getGardenSecret(ctx, gardenClient, runtimeID, registryCacheWithSecret2.UID, gardenNamespace)
		Expect(err).To(BeNil())
		Expect(gardenerSecret2).To(Not(BeNil()))
		verifyGardenSecret(gardenerSecret2, secret2, registryCacheWithSecret2, runtimeID)
	})

	t.Run("Should replace existing secrets", func(t *testing.T) {
		// given
		runtimeID := "test-runtime-id-2"
		secretNameGenerator := fixSecretNameGenerator()
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

		labels1 := fixRegistryCacheGardenSecretLabels(runtimeID, registryCacheWithSecret1.UID)
		labels2 := fixRegistryCacheGardenSecretLabels(runtimeID, registryCacheWithSecret2.UID)
		annotations1 := fixRegistryCacheGardenSecretAnnotations(registryCacheWithSecret1.Name, registryCacheWithSecret1.Namespace)
		annotations2 := fixRegistryCacheGardenSecretAnnotations(registryCacheWithSecret2.Name, registryCacheWithSecret2.Namespace)

		gardenerSecret1 := fixRegistryCacheSecret(secretNameGenerator(runtimeID, registryCacheWithSecret1.UID), gardenNamespace, labels1, annotations1, "user1", "password1")
		gardenerSecret2 := fixRegistryCacheSecret(secretNameGenerator(runtimeID, registryCacheWithSecret2.UID), gardenNamespace, labels2, annotations2, "user2", "password2")

		gardenClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			gardenerSecret1,
			gardenerSecret2,
		).Build()

		// when
		secretSyncer := NewGardenSecretSyncer(gardenClient, runtimeClient, secretNameGenerator, gardenNamespace, runtimeID)
		err := secretSyncer.Do(context.Background(), registryCacheConfigs)

		// then
		Expect(err).To(BeNil())

		secrets, err := getGardenSecrets(ctx, gardenClient, runtimeID)
		Expect(err).To(BeNil())
		Expect(len(secrets)).To(Equal(2))

		updatedGardenerSecret1, err := getGardenSecret(ctx, gardenClient, runtimeID, registryCacheWithSecret1.UID, gardenNamespace)
		Expect(err).To(BeNil())
		Expect(updatedGardenerSecret1).To(Not(BeNil()))
		verifyGardenSecret(updatedGardenerSecret1, secret1, registryCacheWithSecret1, runtimeID)

		updatedGardenerSecret2, err := getGardenSecret(ctx, gardenClient, runtimeID, registryCacheWithSecret2.UID, gardenNamespace)
		Expect(err).To(BeNil())
		Expect(updatedGardenerSecret2).To(Not(BeNil()))
		verifyGardenSecret(updatedGardenerSecret2, secret2, registryCacheWithSecret2, runtimeID)
	})
}
