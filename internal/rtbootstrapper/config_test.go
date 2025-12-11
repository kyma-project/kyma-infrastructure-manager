package rtbootstrapper

import (
	"context"
	"k8s.io/apimachinery/pkg/types"
	"testing"

	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/internal/controller/runtime/fsm/mocks"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	util "k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func Test_Configure(t *testing.T) {
	config := Config{
		PullSecretName:         "test-registry-credentials",
		ClusterTrustBundleName: "",
		ManifestsPath:          "",
		ConfigName:             "test-runtime-bootstrapper-kcp-config",
	}

	pullSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-registry-credentials",
			Namespace: "kcp-system",
		},
		Data: map[string][]byte{
			".dockerconfigjson": []byte(`{"auths":{"test-registry.io":{"username":"test-user","password":"test-password","email":"test-email"}}}`),
		},
		Type: corev1.SecretTypeDockercfg,
	}

	bootstrapperConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-runtime-bootstrapper-kcp-config",
			Namespace: "kcp-system",
		},
		Data: map[string]string{
			"rt-bootstrapper-config.json": "some-configuration-data",
		},
	}

	runtimeCR := minimalRuntime()
	scheme := runtime.NewScheme()
	util.Must(corev1.AddToScheme(scheme))

	t.Run("Should successfully apply bootstrapper ConfigMap and PullSecret to the runtime cluster", func(t *testing.T) {
		// given
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
		fakeKcpClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(pullSecret, bootstrapperConfigMap).Build()
		m := mocks.NewRuntimeClientGetter(t)
		m.On("Get", context.Background(), runtimeCR).Return(fakeClient, nil)

		// when
		configurator := NewConfigurator(fakeKcpClient, m, config)
		err := configurator.Configure(context.Background(), runtimeCR)

		// then
		m.AssertExpectations(t)
		require.NoError(t, err)

		var skrSecret corev1.Secret
		err = fakeClient.Get(context.Background(), types.NamespacedName{Name: "registry-credentials", Namespace: "kyma-system"}, &skrSecret)
		require.NoError(t, err)

		assert.Equal(t, corev1.SecretTypeDockerConfigJson, skrSecret.Type)
		assert.NotNil(t, skrSecret.Data[corev1.DockerConfigJsonKey])
		assert.Equal(t, pullSecret.Data[corev1.DockerConfigJsonKey], skrSecret.Data[corev1.DockerConfigJsonKey])
		assert.Equal(t, "registry-credentials", skrSecret.Name)
		assert.Equal(t, "kyma-system", skrSecret.Namespace)

		var skrConfigMap corev1.ConfigMap
		err = fakeClient.Get(context.Background(), types.NamespacedName{Name: "rt-bootstrapper-config", Namespace: "kyma-system"}, &skrConfigMap)
		require.NoError(t, err)

		assert.Equal(t, "rt-bootstrapper-config", skrConfigMap.Name)
		assert.Equal(t, "kyma-system", skrConfigMap.Namespace)
	})

	t.Run("Should successfully apply only ConfigMap when PullSecret is not configured", func(t *testing.T) {
		// given
		configWithoutSecret := config
		configWithoutSecret.PullSecretName = ""
		kcpClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(bootstrapperConfigMap).Build()
		m := mocks.NewRuntimeClientGetter(t)
		m.On("Get", context.Background(), runtimeCR).Return(fake.NewClientBuilder().WithScheme(scheme).Build(), nil)

		// when
		configurator := NewConfigurator(kcpClient, m, configWithoutSecret)
		err := configurator.Configure(context.Background(), runtimeCR)

		// then
		m.AssertExpectations(t)
		require.NoError(t, err)

	})

	t.Run("Should return error when ConfigMap was not present on KCP", func(t *testing.T) {
		// given
		kcpClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects().Build()

		// when
		configurator := NewConfigurator(kcpClient, nil, config)
		err := configurator.Configure(context.Background(), imv1.Runtime{})

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to prepare bootstrapper ConfigMap")
	})

	t.Run("Should return error when PullSecret was not found on KCP, but it was set as required", func(t *testing.T) {
		// given
		kcpClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(bootstrapperConfigMap).Build()

		// when
		configurator := NewConfigurator(kcpClient, nil, config)
		err := configurator.Configure(context.Background(), imv1.Runtime{})

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to prepare bootstrapper PullSecret")
	})

	t.Run("Should return error when unable to get runtime client", func(t *testing.T) {
		// given
		kcpClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(pullSecret, bootstrapperConfigMap).Build()
		m := mocks.NewRuntimeClientGetter(t)
		m.On("Get", context.Background(), runtimeCR).Return(nil, errors.New("unable to get runtime client"))

		// when
		configurator := NewConfigurator(kcpClient, m, config)
		err := configurator.Configure(context.Background(), runtimeCR)

		// then
		m.AssertExpectations(t)
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to get runtimeClient")
	})

	t.Run("Should return error when unable to apply ConfigMap to runtime cluster", func(t *testing.T) {
		// given
		kcpClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(pullSecret, bootstrapperConfigMap).Build()
		m := mocks.NewRuntimeClientGetter(t)
		fakeRuntimeClient := fake.NewClientBuilder().WithScheme(scheme).
			WithInterceptorFuncs(interceptor.Funcs{
				Patch: func(ctx context.Context, client client.WithWatch, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
					if obj.GetObjectKind().GroupVersionKind().Kind == "ConfigMap" {
						return errors.New("unable to apply bootstrapper configmap to the runtime")
					}

					return nil
				},
			}).Build()
		m.On("Get", context.Background(), runtimeCR).Return(fakeRuntimeClient, nil)

		// when
		configurator := NewConfigurator(kcpClient, m, config)
		err := configurator.Configure(context.Background(), runtimeCR)

		// then
		m.AssertExpectations(t)
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to apply bootstrapper ConfigMap to runtime cluster")
	})

	t.Run("Should return error when unable to apply PullSecret to runtime cluster", func(t *testing.T) {
		// given
		kcpClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(pullSecret, bootstrapperConfigMap).Build()
		m := mocks.NewRuntimeClientGetter(t)
		fakeRuntimeClient := fake.NewClientBuilder().WithScheme(scheme).
			WithInterceptorFuncs(interceptor.Funcs{
				Patch: func(ctx context.Context, client client.WithWatch, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
					if obj.GetObjectKind().GroupVersionKind().Kind == "Secret" {
						return errors.New("unable to apply pull secret to the runtime")
					}
					return nil
				},
			}).Build()
		m.On("Get", context.Background(), runtimeCR).Return(fakeRuntimeClient, nil)

		// when
		configurator := NewConfigurator(kcpClient, m, config)
		err := configurator.Configure(context.Background(), runtimeCR)

		// then
		m.AssertExpectations(t)
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to apply bootstrapper PullSecret to runtime cluster")
	})
}
