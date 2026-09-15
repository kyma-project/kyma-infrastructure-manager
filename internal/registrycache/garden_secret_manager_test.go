package registrycache

import (
	"context"
	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/extender/extensions"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

func TestGardenSecretManager(t *testing.T) {
	RegisterTestingT(t)

	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	ctx := context.Background()

	t.Run("Should return map of cache UUID to secret name", func(t *testing.T) {
		// given
		runtimeID := "test-runtime-id-map"
		gardenNamespace := "garden-dev"
		secretNameGenerator := fixSecretNameGenerator()

		labels1 := fixRegistryCacheGardenSecretLabels(runtimeID, "id1")
		labels2 := fixRegistryCacheGardenSecretLabels(runtimeID, "id2")
		labelsOtherRuntime := fixRegistryCacheGardenSecretLabels("other-runtime", "id3")

		secret1 := fixRegistryCacheSecret(secretNameGenerator(runtimeID, "id1"), gardenNamespace, labels1, map[string]string{}, "user1", "password1")
		secret2 := fixRegistryCacheSecret(secretNameGenerator(runtimeID, "id2"), gardenNamespace, labels2, map[string]string{}, "user2", "password2")
		secretOtherRuntime := fixRegistryCacheSecret(secretNameGenerator("other-runtime", "id3"), gardenNamespace, labelsOtherRuntime, map[string]string{}, "user3", "password3")
		secretNoLabel := fixRegistryCacheSecret("no-label-secret", gardenNamespace, map[string]string{}, map[string]string{}, "user4", "password4")

		gardenClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			secret1,
			secret2,
			secretOtherRuntime,
			secretNoLabel,
		).Build()

		secretManager := NewGardenSecretManager(gardenClient, gardenNamespace, runtimeID)

		// when
		result, err := secretManager.GetCacheUIDToSecretNameMap(ctx)

		// then
		Expect(err).To(BeNil())
		Expect(result).To(HaveLen(2))
		Expect(result["id1"]).To(Equal(secret1.Name))
		Expect(result["id2"]).To(Equal(secret2.Name))
	})

	t.Run("Should return empty map when no secrets exist for runtime", func(t *testing.T) {
		// given
		runtimeID := "test-runtime-id-empty"
		gardenNamespace := "garden-dev"

		gardenClient := fake.NewClientBuilder().WithScheme(scheme).Build()
		secretManager := NewGardenSecretManager(gardenClient, gardenNamespace, runtimeID)

		// when
		result, err := secretManager.GetCacheUIDToSecretNameMap(ctx)

		// then
		Expect(err).To(BeNil())
		Expect(result).To(BeEmpty())
	})

	t.Run("Should delete orphaned and dirty secrets", func(t *testing.T) {
		// given
		runtimeID := "test-runtime-id-3"
		secretNameGenerator := fixSecretNameGenerator()
		gardenNamespace := "garden-dev"

		registryCacheWithSecret1 := fixRegistryCacheConfigWithSecret("config-with-secret-1", "test", "id1", "quay.io", "secret1")
		registryCacheWithSecret3 := fixRegistryCacheConfigWithSecret("config-with-secret-3", "default", "id3", "gcr.io", "secret3")

		registryCacheConfigs := []imv1.ImageRegistryCache{
			fixRegistryCacheConfigWithoutSecret("config-without-secret-1", "test", "id1", "docker.io"),
			registryCacheWithSecret1,
			registryCacheWithSecret3,
		}

		labels1 := fixRegistryCacheGardenSecretLabels(runtimeID, registryCacheWithSecret1.UID)
		labels2 := fixRegistryCacheGardenSecretLabels(runtimeID, "id2")
		labelsDirty := fixRegistryCacheGardenSecretLabels(runtimeID, registryCacheWithSecret3.UID)
		labelsDirty[DirtyLabel] = "true"
		annotations1 := fixRegistryCacheGardenSecretAnnotations(registryCacheWithSecret1.Name, registryCacheWithSecret1.Namespace)
		annotations2 := fixRegistryCacheGardenSecretAnnotations("config-with-secret-2", "test")

		keptSecret := fixRegistryCacheSecret(secretNameGenerator(runtimeID, registryCacheWithSecret1.UID), gardenNamespace, labels1, annotations1, "user1", "password1")
		orphanedSecret := fixRegistryCacheSecret("reg-cache-id", gardenNamespace, labels2, annotations2, "user2", "password2")
		dirtySecret := fixRegistryCacheSecret("old-reg-cache-id3", gardenNamespace, labelsDirty, annotations2, "user3", "password3")

		gardenClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			keptSecret,
			orphanedSecret,
			dirtySecret,
		).Build()

		// when
		secretManager := NewGardenSecretManager(gardenClient, gardenNamespace, runtimeID)
		err := secretManager.DeleteUnused(context.Background(), registryCacheConfigs)

		// then
		Expect(err).To(BeNil())

		secrets, err := getGardenSecrets(ctx, gardenClient, runtimeID)
		Expect(err).To(BeNil())
		Expect(len(secrets)).To(Equal(1))

		remainingSecret, err := getGardenSecret(ctx, gardenClient, runtimeID, registryCacheWithSecret1.UID, gardenNamespace)
		Expect(err).To(BeNil())
		Expect(remainingSecret).To(Not(BeNil()))
		Expect(remainingSecret).To(Equal(keptSecret))
	})

	t.Run("Should exclude dirty secrets from cache UID map", func(t *testing.T) {
		// given
		runtimeID := "test-runtime-id-dirty-map"
		gardenNamespace := "garden-dev"
		secretNameGenerator := fixSecretNameGenerator()

		labelsClean := fixRegistryCacheGardenSecretLabels(runtimeID, "id1")
		labelsDirty := fixRegistryCacheGardenSecretLabels(runtimeID, "id2")
		labelsDirty[DirtyLabel] = "true"

		cleanSecret := fixRegistryCacheSecret(secretNameGenerator(runtimeID, "id1"), gardenNamespace, labelsClean, map[string]string{}, "user1", "password1")
		dirtySecret := fixRegistryCacheSecret(secretNameGenerator(runtimeID, "id2"), gardenNamespace, labelsDirty, map[string]string{}, "user2", "password2")

		gardenClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(cleanSecret, dirtySecret).Build()
		secretManager := NewGardenSecretManager(gardenClient, gardenNamespace, runtimeID)

		// when
		result, err := secretManager.GetCacheUIDToSecretNameMap(ctx)

		// then
		Expect(err).To(BeNil())
		Expect(result).To(HaveLen(1))
		Expect(result["id1"]).To(Equal(cleanSecret.Name))
		Expect(result).NotTo(HaveKey("id2"))
	})
}

func TestHasRegistryCacheCountChanged(t *testing.T) {
	RegisterTestingT(t)

	cache1 := fixRegistryCacheConfigWithoutSecret("cache-1", "test", "id1", "quay.io")
	cache2 := fixRegistryCacheConfigWithoutSecret("cache-2", "test", "id2", "docker.io")

	t.Run("Should return true if desired cache count differs from current", func(t *testing.T) {
		// given
		registryCacheExtension, err := extensions.NewRegistryCacheExtension([]imv1.ImageRegistryCache{cache1, cache2}, map[string]string{}, nil)
		Expect(err).To(BeNil())

		// when
		changed, err := HasRegistryCacheCountChanged([]gardener.Extension{*registryCacheExtension}, []imv1.ImageRegistryCache{cache1})

		// then
		Expect(changed).To(Equal(true))
		Expect(err).To(BeNil())
	})

	t.Run("Should return false if desired cache count matches current", func(t *testing.T) {
		// given
		registryCacheExtension, err := extensions.NewRegistryCacheExtension([]imv1.ImageRegistryCache{cache1, cache2}, map[string]string{}, nil)
		Expect(err).To(BeNil())

		// when
		changed, err := HasRegistryCacheCountChanged([]gardener.Extension{*registryCacheExtension}, []imv1.ImageRegistryCache{cache1, cache2})

		// then
		Expect(changed).To(Equal(false))
		Expect(err).To(BeNil())
	})

	t.Run("Should return true if desired cache list is empty", func(t *testing.T) {
		// given
		registryCacheExtension, err := extensions.NewRegistryCacheExtension([]imv1.ImageRegistryCache{cache1}, map[string]string{}, nil)
		Expect(err).To(BeNil())

		// when
		changed, err := HasRegistryCacheCountChanged([]gardener.Extension{*registryCacheExtension}, []imv1.ImageRegistryCache{})

		// then
		Expect(changed).To(Equal(true))
		Expect(err).To(BeNil())
	})

	t.Run("Should return false if registry cache extension is not currently added", func(t *testing.T) {
		// when
		changed, err := HasRegistryCacheCountChanged([]gardener.Extension{}, []imv1.ImageRegistryCache{cache1})

		// then
		Expect(changed).To(Equal(false))
		Expect(err).To(BeNil())
	})

	t.Run("Should return false if registry cache extension is currently disabled", func(t *testing.T) {
		// given
		registryCacheExtension := gardener.Extension{
			Type:     extensions.RegistryCacheExtensionType,
			Disabled: ptr.To(true),
			ProviderConfig: &runtime.RawExtension{
				Raw: []byte("{}"),
			},
		}

		// when
		changed, err := HasRegistryCacheCountChanged([]gardener.Extension{registryCacheExtension}, []imv1.ImageRegistryCache{cache1})

		// then
		Expect(changed).To(Equal(false))
		Expect(err).To(BeNil())
	})
}
