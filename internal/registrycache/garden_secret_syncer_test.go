package registrycache

import (
	"context"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
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

	t.Run("Should replace existing secrets when data changed", func(t *testing.T) {
		// given
		runtimeID := "test-runtime-id-2"
		secretNameGenerator := fixSecretNameGenerator()
		gardenNamespace := "garden-dev"

		// runtime secret has updated credentials
		secret1 := fixRegistryCacheSecret("secret1", "test", map[string]string{}, map[string]string{}, "newuser", "newpassword")
		// runtime secret2 is unchanged
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

		// old secrets have different names to simulate a prior reconciliation cycle
		// old secret1 has stale credentials; old secret2 has up-to-date credentials
		oldGardenerSecret1 := fixRegistryCacheSecret("old-secret-id1", gardenNamespace, labels1, annotations1, "user1", "password1")
		oldGardenerSecret2 := fixRegistryCacheSecret("old-secret-id2", gardenNamespace, labels2, annotations2, "user2", "password2")

		gardenClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			oldGardenerSecret1,
			oldGardenerSecret2,
		).Build()

		// when
		secretSyncer := NewGardenSecretSyncer(gardenClient, runtimeClient, secretNameGenerator, gardenNamespace, runtimeID)
		err := secretSyncer.Do(context.Background(), registryCacheConfigs)

		// then
		Expect(err).To(BeNil())

		// secret1 was replaced: old marked dirty + new created = 3 total
		// secret2 was unchanged: no action = stays as is
		secrets, err := getGardenSecrets(ctx, gardenClient, runtimeID)
		Expect(err).To(BeNil())
		Expect(len(secrets)).To(Equal(3))

		// new secret1 exists with updated data
		newGardenerSecret1, err := getGardenSecret(ctx, gardenClient, runtimeID, registryCacheWithSecret1.UID, gardenNamespace)
		Expect(err).To(BeNil())
		Expect(newGardenerSecret1).To(Not(BeNil()))
		verifyGardenSecret(newGardenerSecret1, secret1, registryCacheWithSecret1, runtimeID)

		// old secret1 should be marked dirty
		var dirtySecret1 corev1.Secret
		Expect(gardenClient.Get(ctx, types.NamespacedName{Name: oldGardenerSecret1.Name, Namespace: gardenNamespace}, &dirtySecret1)).To(BeNil())
		Expect(dirtySecret1.Labels[DirtyLabel]).To(Equal("true"))

		// old secret2 should be untouched (data unchanged)
		var untouchedSecret2 corev1.Secret
		Expect(gardenClient.Get(ctx, types.NamespacedName{Name: oldGardenerSecret2.Name, Namespace: gardenNamespace}, &untouchedSecret2)).To(BeNil())
		Expect(untouchedSecret2.Labels[DirtyLabel]).To(BeEmpty())
	})

	t.Run("Should not replace existing secrets when data is unchanged", func(t *testing.T) {
		// given
		runtimeID := "test-runtime-id-noop"
		secretNameGenerator := fixSecretNameGenerator()
		gardenNamespace := "garden-dev"

		secret1 := fixRegistryCacheSecret("secret1", "test", map[string]string{}, map[string]string{}, "user1", "password1")

		runtimeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(secret1).Build()

		registryCacheWithSecret1 := fixRegistryCacheConfigWithSecret("config-with-secret-1", secret1.Namespace, "id1", "quay.io", secret1.Name)
		registryCacheConfigs := []imv1.ImageRegistryCache{registryCacheWithSecret1}

		labels1 := fixRegistryCacheGardenSecretLabels(runtimeID, registryCacheWithSecret1.UID)
		annotations1 := fixRegistryCacheGardenSecretAnnotations(registryCacheWithSecret1.Name, registryCacheWithSecret1.Namespace)
		existingGardenSecret := fixRegistryCacheSecret("existing-secret-id1", gardenNamespace, labels1, annotations1, "user1", "password1")

		gardenClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(existingGardenSecret).Build()

		// when
		secretSyncer := NewGardenSecretSyncer(gardenClient, runtimeClient, secretNameGenerator, gardenNamespace, runtimeID)
		err := secretSyncer.Do(context.Background(), registryCacheConfigs)

		// then
		Expect(err).To(BeNil())

		// no new secret created, existing one untouched
		secrets, err := getGardenSecrets(ctx, gardenClient, runtimeID)
		Expect(err).To(BeNil())
		Expect(len(secrets)).To(Equal(1))
		Expect(secrets[0].Name).To(Equal(existingGardenSecret.Name))
		Expect(secrets[0].Labels[DirtyLabel]).To(BeEmpty())
	})
}
