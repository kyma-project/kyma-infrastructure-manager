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
		secret1 := fixRegistryCacheSecret("secret1", "test")
		secret2 := fixRegistryCacheSecret("secret2", "default")

		runtimeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			fixRegistryCacheSecret("orphaned-secret1", "test"),
			secret1,
			secret2,
		).Build()

		seedClient := fake.NewClientBuilder().WithScheme(scheme).Build()

		registryCacheWithSecret1 := fixRegistryCacheConfigWithSecret("config-with-secret-1", secret1.Namespace, "id2", "quay.io", secret1.Name)
		registryCacheWithSecret2 := fixRegistryCacheConfigWithSecret("config-with-secret-2", secret2.Namespace, "id3", "gcr.io", secret2.Name)

		registryCacheConfigs := []imv1.ImageRegistryCache{
			fixRegistryCacheConfigWithoutSecret("config-without-secret-1", "test", "id1", "docker.io"),
			registryCacheWithSecret1,
			registryCacheWithSecret2,
		}

		// when
		secretSyncer := NewSecretSyncer(seedClient, runtimeClient, runtimeID)
		err := secretSyncer.CreateOrUpdate(registryCacheConfigs)

		// then
		Expect(err).To(BeNil())

		secrets, err := getSeedSecrets(ctx, seedClient, runtimeID)
		Expect(err).To(BeNil())

		Expect(len(secrets)).To(Equal(2))

		gardenerSecret1, err := getSeedSecret(ctx, seedClient, fmt.Sprintf(RegistryCacheSecretNameFmt, registryCacheWithSecret1.UID), "test")
		Expect(err).To(BeNil())
		Expect(gardenerSecret1).To(Not(BeNil()))

		Expect(gardenerSecret1.Labels[RegistryCacheSecretLabel]).To(Equal(runtimeID))
		Expect(gardenerSecret1.Data).To(Equal(secret1.Data))

		gardenerSecret2, err := getSeedSecret(ctx, seedClient, fmt.Sprintf(RegistryCacheSecretNameFmt, registryCacheWithSecret2.UID), "default")
		Expect(err).To(BeNil())
		Expect(gardenerSecret2).To(Not(BeNil()))

		Expect(gardenerSecret1.Labels[RegistryCacheSecretLabel]).To(Equal(runtimeID))
		Expect(gardenerSecret2.Data).To(Equal(secret2.Data))
	})

	t.Run("Should update existing secrets", func(t *testing.T) {

	})
}

func fixRegistryCacheSecret(name, namespace string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"kyma-project.io/runtime-id": "test-runtime-id",
			},
		},
		Data: map[string][]byte{
			"username": []byte("test-user"),
			"password": []byte(fmt.Sprintf("test-password-%s", name))},
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

func getSeedSecret(ctx context.Context, seedClient client.Client, name, namespace string) (*corev1.Secret, error) {
	secret := &corev1.Secret{}
	err := seedClient.Get(ctx, types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}, secret)

	return secret, err
}

func getSeedSecrets(ctx context.Context, seedClient client.Client, runtimeID string) ([]corev1.Secret, error) {
	secretList := &corev1.SecretList{}
	err := seedClient.List(ctx, secretList, client.MatchingLabels{RegistryCacheSecretLabel: runtimeID})
	if err != nil {
		return nil, err
	}

	return secretList.Items, nil
}
