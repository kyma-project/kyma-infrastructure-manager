package registrycache

import (
	"context"
	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	kyma "github.com/kyma-project/lifecycle-manager/api/v1beta2"
	registrycache "github.com/kyma-project/registry-cache/api/v1beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"time"
)

const (
	secretForClusterWithRegistryCacheConfig1 = "kubeconfig-secret-for-cluster-with-cache-config1"
	secretForClusterWithRegistryCacheConfig2 = "kubeconfig-secret-for-cluster-with-cache-config2"
	secretNotManagedByKIM                    = "secret-not-managed-by-kim"
	runtimeWithoutRegistryCacheConfig        = "test-runtime-without-registry-cache"
	runtimeWithRegistryCacheEnabled          = "test-runtime-with-registry-cache-enabled"
	runtimeThatShouldNotBeModified           = "test-runtime-that-should-not-be-modified"
)

var _ = Describe("Registry Cache Config Controller", func() {
	Context("When reconciling a Secret resource", func() {
		ctx := context.Background()

		const shootNameForRuntimeWithoutRegistryCache = "shoot-without-registry-cache-ext"

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
					runtime.Spec.Caching[0].Config == expectedRuntimeRegistryCacheConfig[0].Config &&
					runtime.Spec.Caching[0].Name == expectedRuntimeRegistryCacheConfig[0].Name &&
					runtime.Spec.Caching[0].Namespace == expectedRuntimeRegistryCacheConfig[0].Namespace &&
					runtime.Spec.Caching[0].UID == expectedRuntimeRegistryCacheConfig[0].UID &&
					runtime.Spec.Caching[1].Config == expectedRuntimeRegistryCacheConfig[1].Config &&
					runtime.Spec.Caching[1].Name == expectedRuntimeRegistryCacheConfig[1].Name &&
					runtime.Spec.Caching[1].Namespace == expectedRuntimeRegistryCacheConfig[1].Namespace &&
					runtime.Spec.Caching[1].UID == expectedRuntimeRegistryCacheConfig[1].UID

			}, time.Second*300, time.Second*3).Should(BeTrue())
		},
			Entry("with registry cache enabled",
				createRuntimeStub(runtimeWithoutRegistryCacheConfig, shootNameForRuntimeWithoutRegistryCache, nil),
				createSecretStub(secretForClusterWithRegistryCacheConfig1, getSecretLabels(runtimeWithoutRegistryCacheConfig, "infrastructure-manager"))),
			Entry("with registry cache disabled",
				createRuntimeStub(runtimeWithRegistryCacheEnabled, shootNameForRuntimeWithRegistryCacheEnabled, &imv1.ImageRegistryCache{
					Name:      "config3",
					Namespace: "test3",
					Config: registrycache.RegistryCacheConfigSpec{
						Upstream: "some.registry.com",
					},
				}),
				createSecretStub(secretForClusterWithRegistryCacheConfig2, getSecretLabels(runtimeWithRegistryCacheEnabled, "infrastructure-manager"))))

		It("Should not update runtime when secret is not managed by KIM", func() {
			const ShootName = "shoot-cluster-3"
			const secretToIgnore = "secret-to-ignore"
			const secretThatRefersNonExistentShoot = "secret-that-refers-non-existent-shoot"

			By("Creating a Runtime resource")
			runtime := createRuntimeStub(runtimeThatShouldNotBeModified, ShootName, nil)
			Expect(k8sClient.Create(ctx, runtime)).To(Succeed())

			By("Creating a Secret with registry cache config but not managed by KIM")
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

func fixRuntimeClients() map[string]client.Client {
	return map[string]client.Client{
		runtimeWithoutRegistryCacheConfig: fixRuntimeClient(append(fixRegistryCache(), fixKymaCR())...),
		runtimeWithRegistryCacheEnabled:   fixRuntimeClient(append(fixRegistryCache(), fixKymaCR())...),
		runtimeThatShouldNotBeModified:    fixRuntimeClient(),
	}
}

func fixRuntimeClientGetter(runtimeClients map[string]client.Client) func(secret v1.Secret) (client.Client, error) {
	return func(secret v1.Secret) (client.Client, error) {
		runtimeID := secret.Labels["kyma-project.io/runtime-id"]
		runtimeClient, exists := runtimeClients[runtimeID]
		if !exists {
			return nil, errors.New("runtime client not found")
		}
		return runtimeClient, nil
	}
}

func fixRuntimeClient(objs ...client.Object) client.Client {
	scheme := runtime.NewScheme()

	_ = registrycache.AddToScheme(scheme)
	_ = v1.AddToScheme(scheme)
	_ = kyma.AddToScheme(scheme)
	_ = apiextensions.AddToScheme(scheme)

	return fake.NewClientBuilder().WithScheme(scheme).WithObjects(objs...).Build()
}

func fixKymaCR() client.Object {
	return &kyma.Kyma{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "default",
			Namespace: "kyma-system",
		},
		Spec: kyma.KymaSpec{
			Channel: "stable",
			Modules: []kyma.Module{
				{
					Name: "registry-cache",
				},
			},
		},
	}
}

func fixRegistryCache() []client.Object {
	return []client.Object{&v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
		},
	},
		&registrycache.RegistryCacheConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "config1",
				Namespace: "test",
			},
			Spec: registrycache.RegistryCacheConfigSpec{
				Upstream: "docker.io",
			},
		}, &registrycache.RegistryCacheConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "config2",
				Namespace: "test",
			},
			Spec: registrycache.RegistryCacheConfigSpec{
				Upstream: "quay.io",
			},
		},
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
