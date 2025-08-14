package registrycache

import (
	"context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	util "k8s.io/apimachinery/pkg/util/runtime"
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

	scheme := runtime.NewScheme()
	util.Must(registrycache.AddToScheme(scheme))

	runtimeInstance := &imv1.Runtime{
		Spec: imv1.RuntimeSpec{
			Caching: []imv1.ImageRegistryCache{
				{
					Name:      "cache1",
					Namespace: "test",
					UID:       "uid1",
				},
				{
					Name:      "cache2",
					Namespace: "test",
					UID:       "uid2",
				},
			},
		},
	}

	registryCache1 := &registrycache.RegistryCacheConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cache1",
			Namespace: "test",
		},
	}

	registryCache2 := &registrycache.RegistryCacheConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cache2",
			Namespace: "test",
		},
	}

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(registryCache1, registryCache2).WithObjects(registryCache1, registryCache2).Build()

	statusManager := NewStatusManager(fakeClient)

	t.Run("SetStatusReady", func(t *testing.T) {
		err := statusManager.SetStatusReady(context.Background(), *runtimeInstance, registrycache.ConditionReasonRegistryCacheConfigured)
		Expect(err).To(BeNil())

		updatedRegistryCache1 := &registrycache.RegistryCacheConfig{}
		err = fakeClient.Get(context.Background(), types.NamespacedName{Name: "cache1", Namespace: "test"}, updatedRegistryCache1)
		Expect(err).To(BeNil())
		Expect(updatedRegistryCache1.Status.Conditions[0].Reason).To(Equal(string(registrycache.ConditionReasonRegistryCacheConfigured)))
		Expect(updatedRegistryCache1.Status.Conditions[0].Status).To(Equal(metav1.ConditionTrue))

		updatedRegistryCache2 := &registrycache.RegistryCacheConfig{}
		err = fakeClient.Get(context.Background(), types.NamespacedName{Name: "cache2", Namespace: "test"}, updatedRegistryCache2)
		Expect(err).To(BeNil())
		Expect(updatedRegistryCache2.Status.Conditions[0].Reason).To(Equal(string(registrycache.ConditionReasonRegistryCacheConfigured)))
		Expect(updatedRegistryCache2.Status.Conditions[0].Status).To(Equal(metav1.ConditionTrue))
	})

	t.Run("SetStatusFailed", func(t *testing.T) {
		err := statusManager.SetStatusFailed(context.Background(), *runtimeInstance, registrycache.ConditionReasonRegistryCacheExtensionConfigurationFailed, "Error occurred")
		Expect(err).To(BeNil())

		updatedRegistryCache1 := &registrycache.RegistryCacheConfig{}
		err = fakeClient.Get(context.Background(), types.NamespacedName{Name: "cache1", Namespace: "test"}, updatedRegistryCache1)
		Expect(err).To(BeNil())
		Expect(updatedRegistryCache1.Status.Conditions[0].Reason).To(Equal(string(registrycache.ConditionReasonRegistryCacheExtensionConfigurationFailed)))
		Expect(updatedRegistryCache1.Status.Conditions[0].Status).To(Equal(metav1.ConditionFalse))
		Expect(updatedRegistryCache1.Status.Conditions[0].Message).To(Equal("Error occurred"))

		updatedRegistryCache2 := &registrycache.RegistryCacheConfig{}
		err = fakeClient.Get(context.Background(), types.NamespacedName{Name: "cache2", Namespace: "test"}, updatedRegistryCache2)
		Expect(err).To(BeNil())
		Expect(updatedRegistryCache2.Status.Conditions[0].Reason).To(Equal(string(registrycache.ConditionReasonRegistryCacheExtensionConfigurationFailed)))
		Expect(updatedRegistryCache2.Status.Conditions[0].Status).To(Equal(metav1.ConditionFalse))
		Expect(updatedRegistryCache2.Status.Conditions[0].Message).To(Equal("Error occurred"))
	})

	t.Run("SetStatusPending", func(t *testing.T) {
		err := statusManager.SetStatusPending(context.Background(), *runtimeInstance, registrycache.ConditionReasonRegistryCacheConfigured)
		Expect(err).To(BeNil())

		updatedRegistryCache1 := &registrycache.RegistryCacheConfig{}
		err = fakeClient.Get(context.Background(), types.NamespacedName{Name: "cache1", Namespace: "test"}, updatedRegistryCache1)
		Expect(err).To(BeNil())
		Expect(updatedRegistryCache1.Status.Conditions[0].Reason).To(Equal(string(registrycache.ConditionReasonRegistryCacheConfigured)))
		Expect(updatedRegistryCache1.Status.Conditions[0].Status).To(Equal(metav1.ConditionUnknown))

		updatedRegistryCache2 := &registrycache.RegistryCacheConfig{}
		err = fakeClient.Get(context.Background(), types.NamespacedName{Name: "cache2", Namespace: "test"}, updatedRegistryCache2)
		Expect(err).To(BeNil())
		Expect(updatedRegistryCache2.Status.Conditions[0].Reason).To(Equal(string(registrycache.ConditionReasonRegistryCacheConfigured)))
		Expect(updatedRegistryCache2.Status.Conditions[0].Status).To(Equal(metav1.ConditionUnknown))
	})
}
