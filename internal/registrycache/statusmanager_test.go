package registrycache

import (
	"context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"

	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	registrycache "github.com/kyma-project/kim-snatch/api/v1beta1"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestStatusManager(t *testing.T) {
	RegisterTestingT(t)

	// Setup fake runtime and registry cache objects
	scheme := runtime.NewScheme()
	_ = registrycache.AddToScheme(scheme)

	runtimeInstance := &imv1.Runtime{
		Spec: imv1.RuntimeSpec{
			Caching: []imv1.ImageRegistryCache{
				{
					Name:      "cache1",
					Namespace: "default",
					UID:       "uid1",
				},
				{
					Name:      "cache2",
					Namespace: "default",
					UID:       "uid2",
				},
			},
		},
	}

	registryCache1 := &registrycache.RegistryCacheConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cache1",
			Namespace: "default",
		},
	}

	registryCache2 := &registrycache.RegistryCacheConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cache2",
			Namespace: "default",
		},
	}

	// Create a fake client with the registry cache objects
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(registryCache1, registryCache2).Build()

	// Create the StatusManager
	statusManager := NewStatusManager(fakeClient)

	t.Run("SetStatusReady", func(t *testing.T) {
		err := statusManager.SetStatusReady(context.Background(), *runtimeInstance, registrycache.ConditionReasonRegistryCacheConfigured)
		Expect(err).To(BeNil())

		// Verify the status of the first registry cache
		updatedRegistryCache1 := &registrycache.RegistryCacheConfig{}
		err = fakeClient.Get(context.Background(), types.NamespacedName{Name: "cache1", Namespace: "default"}, updatedRegistryCache1)
		Expect(err).To(BeNil())
		Expect(updatedRegistryCache1.Status.Conditions[0].Reason).To(Equal(registrycache.ConditionReasonRegistryCacheConfigured))
		Expect(updatedRegistryCache1.Status.Conditions[0].Status).To(Equal("True"))

		// Verify the status of the second registry cache
		updatedRegistryCache2 := &registrycache.RegistryCacheConfig{}
		err = fakeClient.Get(context.Background(), types.NamespacedName{Name: "cache2", Namespace: "default"}, updatedRegistryCache2)
		Expect(err).To(BeNil())
		Expect(updatedRegistryCache2.Status.Conditions[0].Reason).To(Equal(registrycache.ConditionReasonRegistryCacheConfigured))
		Expect(updatedRegistryCache2.Status.Conditions[0].Status).To(Equal("True"))
	})

	t.Run("SetStatusFailed", func(t *testing.T) {
		err := statusManager.SetStatusFailed(context.Background(), *runtimeInstance, registrycache.ConditionReasonRegistryCacheExtensionConfigurationFailed, "Error occurred")
		Expect(err).To(BeNil())

		// Verify the status of the first registry cache
		updatedRegistryCache1 := &registrycache.RegistryCacheConfig{}
		err = fakeClient.Get(context.Background(), types.NamespacedName{Name: "cache1", Namespace: "default"}, updatedRegistryCache1)
		Expect(err).To(BeNil())
		Expect(updatedRegistryCache1.Status.Conditions[0].Reason).To(Equal(registrycache.ConditionReasonRegistryCacheExtensionConfigurationFailed))
		Expect(updatedRegistryCache1.Status.Conditions[0].Status).To(Equal("False"))
		Expect(updatedRegistryCache1.Status.Conditions[0].Message).To(Equal("Error occurred"))

		// Verify the status of the second registry cache
		updatedRegistryCache2 := &registrycache.RegistryCacheConfig{}
		err = fakeClient.Get(context.Background(), types.NamespacedName{Name: "cache2", Namespace: "default"}, updatedRegistryCache2)
		Expect(err).To(BeNil())
		Expect(updatedRegistryCache2.Status.Conditions[0].Reason).To(Equal(registrycache.ConditionReasonRegistryCacheExtensionConfigurationFailed))
		Expect(updatedRegistryCache2.Status.Conditions[0].Status).To(Equal("False"))
		Expect(updatedRegistryCache2.Status.Conditions[0].Message).To(Equal("Error occurred"))
	})

	t.Run("SetStatusPending", func(t *testing.T) {
		err := statusManager.SetStatusPending(context.Background(), *runtimeInstance, registrycache.ConditionReasonRegistryCacheConfigured)
		Expect(err).To(BeNil())

		// Verify the status of the first registry cache
		updatedRegistryCache1 := &registrycache.RegistryCacheConfig{}
		err = fakeClient.Get(context.Background(), types.NamespacedName{Name: "cache1", Namespace: "default"}, updatedRegistryCache1)
		Expect(err).To(BeNil())
		Expect(updatedRegistryCache1.Status.Conditions[0].Reason).To(Equal(registrycache.ConditionReasonRegistryCacheConfigured))
		Expect(updatedRegistryCache1.Status.Conditions[0].Status).To(Equal("Unknown"))

		// Verify the status of the second registry cache
		updatedRegistryCache2 := &registrycache.RegistryCacheConfig{}
		err = fakeClient.Get(context.Background(), types.NamespacedName{Name: "cache2", Namespace: "default"}, updatedRegistryCache2)
		Expect(err).To(BeNil())
		Expect(updatedRegistryCache2.Status.Conditions[0].Reason).To(Equal(registrycache.ConditionReasonRegistryCacheConfigured))
		Expect(updatedRegistryCache2.Status.Conditions[0].Status).To(Equal("Unknown"))
	})
}
