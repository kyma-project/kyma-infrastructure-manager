package registrycache

import (
	"context"
	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/internal/controller/registrycache/mocks"
	registrycache "github.com/kyma-project/kim-snatch/api/v1beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"time"
)

const (
	secretForClusterWithCustomConfig1 = "kubeconfig-secret-for-cluster-with-custom-config1"
	secretForClusterWithCustomConfig2 = "kubeconfig-secret-for-cluster-with-custom-config2"
	secretNotManagedByKIM             = "secret-not-managed-by-kim"
)

var _ = Describe("Custom Config Controller", func() {
	Context("When reconciling a Secret resource", func() {
		ctx := context.Background()

		const runtimeWithoutRegistryCacheConfig = "test-runtime-without-registry-cache"
		const shootNameForRuntimeWithoutRegistryCache = "shoot-without-registry-cache-ext"
		const runtimeWithRegistryCacheEnabled = "test-runtime-with-registry-cache-enabled"
		const shootNameForRuntimeWithRegistryCacheEnabled = "shoot-with-registry-cache-enabled-ext"

		DescribeTable("Should update Runtime CR", func(newRuntime *imv1.Runtime, newSecret *v1.Secret) {

			Expect(k8sClient.Create(ctx, newRuntime)).To(Succeed())

			Expect(k8sClient.Create(ctx, newSecret)).To(Succeed())

			expectedRuntimeRegistryCacheConfig := []imv1.ImageRegistryCache{
				{
					Name:      "config1",
					Namespace: "test",

					Config: registrycache.RegistryCacheConfigSpec{
						Upstream: "docker.io",
					},
				},
				{
					Name:      "config2",
					Namespace: "test",

					Config: registrycache.RegistryCacheConfigSpec{
						Upstream: "quay.io",
					},
				},
			}

			// Check if Runtime CR has registry cache enabled
			Eventually(func() bool {
				runtime := imv1.Runtime{}
				if err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      newRuntime.Name,
					Namespace: newRuntime.Namespace,
				}, &runtime); err != nil {
					return false
				}

				return len(runtime.Spec.Caching) == 2 &&
					runtime.Spec.Caching[0] == expectedRuntimeRegistryCacheConfig[0] &&
					runtime.Spec.Caching[1] == expectedRuntimeRegistryCacheConfig[1]

			}, time.Second*300, time.Second*3).Should(BeTrue())
		},
			Entry("with registry cache enabled",
				createRuntimeStub(runtimeWithoutRegistryCacheConfig, shootNameForRuntimeWithoutRegistryCache, nil),
				createSecretStub(secretForClusterWithCustomConfig1, getSecretLabels(runtimeWithoutRegistryCacheConfig, "infrastructure-manager"))),
			Entry("with registry cache disabled",
				createRuntimeStub(runtimeWithRegistryCacheEnabled, shootNameForRuntimeWithRegistryCacheEnabled, &imv1.ImageRegistryCache{
					Name:      "config3",
					Namespace: "test3",
					Config: registrycache.RegistryCacheConfigSpec{
						Upstream: "some.registry.com",
					},
				}),
				createSecretStub(secretForClusterWithCustomConfig2, getSecretLabels(runtimeWithRegistryCacheEnabled, "infrastructure-manager"))))

		It("Should not update runtime when secret is not managed by KIM", func() {
			const ShootName = "shoot-cluster-3"
			const runtimeThatShouldNotBeModified = "test-runtime-that-should-not-be-modified"
			const secretToIgnore = "secret-to-ignore"
			const secretThatRefersNonExistentShoot = "secret-that-refers-non-existent-shoot"

			By("Creating a Runtime resource")
			runtime := createRuntimeStub(runtimeThatShouldNotBeModified, ShootName, nil)
			Expect(k8sClient.Create(ctx, runtime)).To(Succeed())

			By("Creating a Secret with custom config but not managed by KIM")
			secret := createSecretStub(secretNotManagedByKIM, getSecretLabels(runtimeThatShouldNotBeModified, "something-else"))
			Expect(k8sClient.Create(ctx, secret)).To(Succeed())

			By("Creating a Secret that does not contain kubeconfig")
			secretNotManaged := createSecretStub(secretToIgnore, map[string]string{})
			Expect(k8sClient.Create(ctx, secretNotManaged)).To(Succeed())

			By("Creating a Secret that refers to a non-existent shoot")
			secretWithNonExistentShoot := createSecretStub(secretThatRefersNonExistentShoot, getSecretLabels(runtimeThatShouldNotBeModified, "infrastructure-manager"))
			Expect(k8sClient.Create(ctx, secretWithNonExistentShoot)).To(Succeed())

			By("Check if Runtime CR has registry cache enabled")
			Consistently(func() bool {
				runtime := imv1.Runtime{}
				if err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      runtimeThatShouldNotBeModified,
					Namespace: "default",
				}, &runtime); err != nil {
					return false
				}

				return runtime.Spec.Caching == nil

			}, time.Second*30, time.Second*3).Should(BeTrue())
		})
	})
})

func fixMockedRegistryCache() func(secret v1.Secret) (RegistryCache, error) {
	callsMap := map[string]int{
		secretForClusterWithCustomConfig1: 0,
		secretForClusterWithCustomConfig2: 0,
		secretNotManagedByKIM:             0,
	}

	testConfig := []registrycache.RegistryCacheConfig{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "config1",
				Namespace: "test",
			},
			Spec: registrycache.RegistryCacheConfigSpec{
				Upstream: "docker.io",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "config2",
				Namespace: "test",
			},
			Spec: registrycache.RegistryCacheConfigSpec{
				Upstream: "quay.io",
			},
		},
	}

	resultsMap := map[string][]registrycache.RegistryCacheConfig{
		secretForClusterWithCustomConfig1: testConfig,
		secretForClusterWithCustomConfig2: testConfig,
		secretNotManagedByKIM:             testConfig,
	}

	return func(secret v1.Secret) (RegistryCache, error) {

		if _, found := callsMap[secret.Name]; !found {
			return nil, errors.Errorf("unexpected secret name %s", secret.Name)
		}

		if callsMap[secret.Name] == 0 {
			callsMap[secret.Name]++
			return nil, errors.New("failed to get registry cache config")
		}

		registryCacheMock := &mocks.RegistryCache{}
		registryCacheMock.On("GetRegistryCacheConfig").Return(resultsMap[secret.Name], nil)

		return registryCacheMock, nil
	}
}

func getSecretLabels(runtimeID, managedBy string) map[string]string {
	return map[string]string{
		"kyma-project.io/runtime-id":          runtimeID,
		"operator.kyma-project.io/managed-by": managedBy,
	}
}

func createSecretStub(name string, labels map[string]string) *v1.Secret {
	return &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
			Labels:    labels,
		},
	}
}

func createRuntimeStub(name string, shootName string, registryCacheConfig *imv1.ImageRegistryCache) *imv1.Runtime {
	runtime := &imv1.Runtime{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
			Labels: map[string]string{
				"kyma-project.io/runtime-id": name,
			},
		},
		Spec: imv1.RuntimeSpec{
			Shoot: imv1.RuntimeShoot{
				Name: shootName,
				Provider: imv1.Provider{
					Type: "aws",
					Workers: []gardener.Worker{
						{
							Name:  "worker-0",
							Zones: []string{"zone1", "zone2"},
						},
					},
				},
			},
			Security: imv1.Security{
				Administrators: []string{"admin@example.com"},
			},
		},
	}

	if registryCacheConfig != nil {
		runtime.Spec.Caching = []imv1.ImageRegistryCache{*registryCacheConfig}
	}

	return runtime
}
