package registrycache

import (
	"context"
	"testing"

	registrycachev1beta1 "github.com/kyma-project/kim-snatch/api/v1beta1"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestConfigExplorer_RegistryCacheConfigExists(t *testing.T) {
	RegisterTestingT(t)
	ctx := context.Background()

	scheme := runtime.NewScheme()
	_ = registrycachev1beta1.AddToScheme(scheme)

	t.Run("Return true if at least one RegistryCacheConfig CR exist", func(t *testing.T) {
		// given
		registryCacheConfig := &registrycachev1beta1.RegistryCacheConfig{
			Spec: registrycachev1beta1.RegistryCacheConfigSpec{
				Upstream: "docker.io",
			},
		}

		client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(registryCacheConfig).Build()
		explorer := NewConfigExplorer(ctx, client)

		// when
		exists, err := explorer.RegistryCacheConfigExists()

		// then
		Expect(err).To(BeNil())
		Expect(exists).To(BeTrue())
	})

	t.Run("Return false no RegistryCacheConfig CR exist", func(t *testing.T) {
		// given
		client := fake.NewClientBuilder().WithScheme(scheme).Build()
		explorer := NewConfigExplorer(ctx, client)

		// when
		exists, err := explorer.RegistryCacheConfigExists()

		// then
		Expect(err).To(BeNil())
		Expect(exists).To(BeFalse())
	})

	t.Run("Return error when failed to list RegistryCacheConfig CRs", func(t *testing.T) {
		// given
		client := fake.NewClientBuilder().Build()
		explorer := NewConfigExplorer(ctx, client)

		// when
		exists, err := explorer.RegistryCacheConfigExists()

		// then
		Expect(err).To(Not(BeNil()))
		Expect(exists).To(BeFalse())
	})
}

func TestConfigExplorer_GetRegistryCacheConfig(t *testing.T) {
	RegisterTestingT(t)

	ctx := context.Background()

	scheme := runtime.NewScheme()
	_ = registrycachev1beta1.AddToScheme(scheme)

	t.Run("Return non empty RegistryCacheConfig list", func(t *testing.T) {
		// given
		customConfig := &registrycachev1beta1.RegistryCacheConfig{
			Spec: registrycachev1beta1.RegistryCacheConfigSpec{
				Upstream: "docker.io",
			},
		}
		client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(customConfig).Build()

		explorer := NewConfigExplorer(ctx, client)

		// when
		configs, err := explorer.GetRegistryCacheConfig()

		// then
		Expect(err).To(BeNil())
		Expect(configs).To(HaveLen(1))
		Expect(configs[0].Spec.Upstream).To(Equal("docker.io"))
	})

	t.Run("Return empty RegistryCacheConfig list", func(t *testing.T) {
		// given
		client := fake.NewClientBuilder().WithScheme(scheme).Build()
		explorer := NewConfigExplorer(ctx, client)

		// when
		configs, err := explorer.GetRegistryCacheConfig()

		// then
		Expect(err).To(BeNil())
		Expect(configs).To(BeEmpty())
	})

	t.Run("Return error when failed to list RegistryCacheConfig CRs", func(t *testing.T) {
		// given
		client := fake.NewClientBuilder().Build()
		explorer := NewConfigExplorer(ctx, client)

		// when
		configs, err := explorer.GetRegistryCacheConfig()

		// then
		Expect(err).To(Not(BeNil()))
		Expect(configs).To(BeNil())
	})
}
