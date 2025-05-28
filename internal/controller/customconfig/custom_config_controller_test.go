package customconfig

import (
	"context"
	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"time"
)

var _ = Describe("Custom Config Controller", func() {
	Context("When reconciling a Secret resource", func() {
		ctx := context.Background()

		It("Should enable registry cache configuration when custom config exists", func() {
			const RuntimeName = "test-runtime-1"
			const SecretName = "kubeconfig-cluster-1"
			const ShootName = "shoot-cluster-1"

			By("Creating a Runtime resource")
			runtime := createRuntimeStub(RuntimeName, ShootName, false)
			Expect(k8sClient.Create(ctx, runtime)).To(Succeed())

			By("Creating a Secret with custom config")
			secret := createSecretStub(SecretName, RuntimeName, map[string]string{
				"kyma-project.io/runtime-id":          RuntimeName,
				"operator.kyma-project.io/managed-by": "infrastructure-manager",
			})
			Expect(k8sClient.Create(ctx, secret)).To(Succeed())

			By("Check if Runtime CR has registry cache enabled")
			Eventually(func() bool {
				runtime := imv1.Runtime{}
				if err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      RuntimeName,
					Namespace: "default",
				}, &runtime); err != nil {
					return false
				}

				return runtime.Spec.Caching != nil && runtime.Spec.Caching.Enabled

			}, time.Second*300, time.Second*3).Should(BeTrue())
		})

		It("Should disable registry cache configuration when custom config doesn't exist", func() {
			const RuntimeName = "test-runtime-2"
			const SecretName = "kubeconfig-cluster-2"
			const ShootName = "shoot-cluster-2"

			By("Creating a Runtime resource")
			runtime := createRuntimeStub(RuntimeName, ShootName, true)
			Expect(k8sClient.Create(ctx, runtime)).To(Succeed())

			By("Creating a Secret with custom config")
			secret := createSecretStub(SecretName, RuntimeName, map[string]string{
				"kyma-project.io/runtime-id":          RuntimeName,
				"operator.kyma-project.io/managed-by": "infrastructure-manager",
			})
			Expect(k8sClient.Create(ctx, secret)).To(Succeed())

			By("Check if Runtime CR has registry cache enabled")
			Eventually(func() bool {
				runtime := imv1.Runtime{}
				if err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      RuntimeName,
					Namespace: "default",
				}, &runtime); err != nil {
					return false
				}

				return runtime.Spec.Caching != nil && !runtime.Spec.Caching.Enabled

			}, time.Second*300, time.Second*3).Should(BeTrue())
		})

		It("Should not update runtime when secret is not managed by KIM", func() {
			const RuntimeName = "test-runtime-3"
			const SecretName = "kubeconfig-cluster-3"
			const SecretNotContainingKubeconfig = "some-secret"
			const ShootName = "shoot-cluster-3"

			By("Creating a Runtime resource")
			runtime := createRuntimeStub(RuntimeName, ShootName, false)
			Expect(k8sClient.Create(ctx, runtime)).To(Succeed())

			By("Creating a Secret with custom config but not managed by KIM")
			secret := createSecretStub(SecretName, RuntimeName, map[string]string{
				"kyma-project.io/runtime-id": RuntimeName,
			})
			Expect(k8sClient.Create(ctx, secret)).To(Succeed())

			By("Creating a Secret that does not contain kubeconfig")
			secretNotManaged := createSecretStub(SecretNotContainingKubeconfig, RuntimeName, map[string]string{})
			Expect(k8sClient.Create(ctx, secretNotManaged)).To(Succeed())

			By("Check if Runtime CR has registry cache enabled")
			Consistently(func() bool {
				runtime := imv1.Runtime{}
				if err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      RuntimeName,
					Namespace: "default",
				}, &runtime); err != nil {
					return false
				}

				return runtime.Spec.Caching == nil || !runtime.Spec.Caching.Enabled

			}, time.Second*60, time.Second*3).Should(BeTrue())
		})

	})
})

func createSecretStub(name, runtimeName string, labels map[string]string) *v1.Secret {
	return &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
			Labels:    labels,
		},
	}
}

func createRuntimeStub(name string, shootName string, registryCacheEnabled bool) *imv1.Runtime {
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

	if registryCacheEnabled {
		runtime.Spec.Caching = &imv1.ImageRegistryCache{
			Enabled: true,
		}
	}

	return runtime
}
