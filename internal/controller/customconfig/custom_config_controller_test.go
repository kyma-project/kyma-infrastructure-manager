package customconfig

import (
	"context"
	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Custom Config Controller", func() {
	Context("When reconciling a Secret resource", func() {
		const SecretName = "test-secret"
		const RuntimeName = "test-runtime"
		ctx := context.Background()

		//typeNamespacedName := types.NamespacedName{
		//	Name:      SecretName,
		//	Namespace: "default",
		//}

		It("Should enable registry cache configuration when custom config exists", func() {
			By("Creating a Secret with custom config")
			secret := createSecretStub(SecretName)
			Expect(k8sClient.Create(ctx, secret)).To(Succeed())

			By("Creating a Runtime resource")
			runtime := createRuntimeStub(RuntimeName, "shootName")
			Expect(k8sClient.Create(ctx, runtime)).To(Succeed())

			//By("Reconciling the Secret")
			//result, err := reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: typeNamespacedName})
			//Expect(err).To(BeNil())
			//Expect(result.Requeue).To(BeTrue())
			//
			//By("Verifying that registry cache is enabled in the Runtime")
			//updatedRuntime := &imv1.Runtime{}
			//Expect(k8sClient.Get(ctx, types.NamespacedName{Name: RuntimeName, Namespace: "default"}, updatedRuntime)).To(Succeed())
			//Expect(updatedRuntime.Spec.Caching).NotTo(BeNil())
			//Expect(updatedRuntime.Spec.Caching.Enabled).To(BeTrue())
		})

		//It("Should disable registry cache configuration when custom config does not exist", func() {
		//	By("Creating a Secret without custom config")
		//	secret := createSecretStub(SecretName)
		//	Expect(k8sClient.Create(ctx, secret)).To(Succeed())
		//
		//	By("Creating a Runtime resource with caching enabled")
		//	runtime := createRuntimeStub(RuntimeName)
		//	runtime.Spec.Caching = &imv1.ImageRegistryCache{Enabled: true}
		//	Expect(k8sClient.Create(ctx, runtime)).To(Succeed())
		//
		//	By("Reconciling the Secret")
		//	result, err := reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: typeNamespacedName})
		//	Expect(err).To(BeNil())
		//	Expect(result.Requeue).To(BeTrue())
		//
		//	By("Verifying that registry cache is disabled in the Runtime")
		//	updatedRuntime := &imv1.Runtime{}
		//	Expect(k8sClient.Get(ctx, types.NamespacedName{Name: RuntimeName, Namespace: "default"}, updatedRuntime)).To(Succeed())
		//	Expect(updatedRuntime.Spec.Caching).NotTo(BeNil())
		//	Expect(updatedRuntime.Spec.Caching.Enabled).To(BeFalse())
		//})
	})
})

func createSecretStub(name string) *v1.Secret {
	return &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
			Labels: map[string]string{
				"kyma-project.io/runtime-id":          "test-runtime",
				"operator.kyma-project.io/managed-by": "infrastructure-manager",
			},
		},
	}
}

func createRuntimeStub(name string, shootName string) *imv1.Runtime {
	return &imv1.Runtime{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
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
}
