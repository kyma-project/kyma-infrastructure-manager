package rtbootstrapper

import (
	"context"
	"k8s.io/apimachinery/pkg/types"
	"testing"

	"github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/internal/controller/runtime/fsm/mocks"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	certificatesv1beta1 "k8s.io/api/certificates/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	util "k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

// newConfig creates a Config for tests with provided parameters.
func newConfig(pullSecretName, clusterTrustBundleName, configName string) Config {
	return Config{
		KCPConfig: KCPConfig{
			PullSecretName:         pullSecretName,
			ClusterTrustBundleName: clusterTrustBundleName,
			ConfigName:             configName,
		},
		SKRConfig: SKRConfig{
			PullSecretName:         pullSecretName,
			ClusterTrustBundleName: clusterTrustBundleName,
			//ManifestsConfigMapName: manifestsConfigMapName,
			ConfigName: configName,
		},
	}
}

func Test_Configure(t *testing.T) {
	config := newConfig("test-registry-credentials", "test-cluster-trust-bundle", "test-runtime-bootstrapper-kcp-config")

	kcpPullSecret := newPullSecret(config.KCPConfig.PullSecretName, "kcp-system", []byte(`{"auths":{"test-registry.io":{"username":"test-user","password":"test-password","email":"test-email"}}}`))
	kcpBootstrapperConfigMap := newBootstrapperConfigMap(config.KCPConfig.ConfigName, "kcp-system", map[string]string{"rt-bootstrapper-config.json": "some-configuration-data"})
	kcpClusterTrustBundle := newClusterTrustBundle(config.KCPConfig.ClusterTrustBundleName, "-----BEGIN CERTIFICATE-----\ntest-certificate-data\n-----END CERTIFICATE-----")

	skrPullSecret := newPullSecret(config.SKRConfig.PullSecretName, "kyma-system", []byte(`{"auths":{"test-registry.io":{"username":"test-user","password":"test-password","email":"test-email"}}}`))
	skrBootstrapperConfigMap := newBootstrapperConfigMap(config.SKRConfig.ConfigName, "kyma-system", map[string]string{"rt-bootstrapper-config.json": "some-configuration-data"})
	skrClusterTrustBundle := newClusterTrustBundle(config.SKRConfig.ClusterTrustBundleName, "-----BEGIN CERTIFICATE-----\ntest-certificate-data\n-----END CERTIFICATE-----")

	runtimeCR := minimalRuntime()
	scheme := runtime.NewScheme()
	util.Must(corev1.AddToScheme(scheme))
	util.Must(certificatesv1beta1.AddToScheme(scheme))

	t.Run("Should skip configuration if resources are up-to-date", func(t *testing.T) {
		// given
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(skrPullSecret, skrBootstrapperConfigMap, skrClusterTrustBundle).WithInterceptorFuncs(interceptor.Funcs{
			Patch: func(ctx context.Context, client client.WithWatch, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
				if obj.GetObjectKind().GroupVersionKind().Kind == "ConfigMap" {
					return errors.New("should not patch ConfigMap when resources are up-to-date")
				}

				return nil
			},
		}).Build()
		fakeKcpClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(kcpPullSecret, kcpBootstrapperConfigMap, kcpClusterTrustBundle).Build()
		m := mocks.NewRuntimeClientGetter(t)
		m.On("Get", context.Background(), runtimeCR).Return(fakeClient, nil)

		// when
		configurator := NewConfigurator(fakeKcpClient, m, config)
		err := configurator.Configure(context.Background(), runtimeCR)

		// then
		m.AssertExpectations(t)
		require.NoError(t, err)

		assertPullSecret(t, config, fakeClient, skrPullSecret)
		assertConfigMap(t, config, skrBootstrapperConfigMap, fakeClient)
		assertClusterTrustBundle(t, config, fakeClient, skrClusterTrustBundle)
	})

	t.Run("Should update configuration if resources require update", func(t *testing.T) {
		// given
		newKCPPullSecret := newPullSecret(config.KCPConfig.PullSecretName, "kcp-system", []byte(`{"auths":{"test-registry.io":{"username":"new-test-user","password":"new-test-password","email":"test-email"}}}`))
		newKCPBootstrapperConfigMap := newBootstrapperConfigMap(config.KCPConfig.ConfigName, "kcp-system", map[string]string{"rt-bootstrapper-config.json": "some-new-configuration-data"})
		newKCPClusterTrustBundle := newClusterTrustBundle(config.KCPConfig.ClusterTrustBundleName, "-----BEGIN CERTIFICATE-----\nnew-test-certificate-data\n-----END CERTIFICATE-----")

		newSKRPullSecret := newPullSecret(config.SKRConfig.PullSecretName, "kyma-system", []byte(`{"auths":{"test-registry.io":{"username":"new-test-user","password":"new-test-password","email":"test-email"}}}`))
		newSKRBootstrapperConfigMap := newBootstrapperConfigMap(config.SKRConfig.ConfigName, "kyma-system", map[string]string{"rt-bootstrapper-config.json": "some-new-configuration-data"})
		newSKRClusterTrustBundle := newClusterTrustBundle(config.SKRConfig.ClusterTrustBundleName, "-----BEGIN CERTIFICATE-----\nnew-test-certificate-data\n-----END CERTIFICATE-----")

		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(kcpPullSecret, kcpBootstrapperConfigMap, kcpClusterTrustBundle).Build()
		fakeKcpClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(newKCPPullSecret, newKCPBootstrapperConfigMap, newKCPClusterTrustBundle).Build()
		m := mocks.NewRuntimeClientGetter(t)
		m.On("Get", context.Background(), runtimeCR).Return(fakeClient, nil)

		// when
		configurator := NewConfigurator(fakeKcpClient, m, config)
		err := configurator.Configure(context.Background(), runtimeCR)

		// then
		m.AssertExpectations(t)
		require.NoError(t, err)

		assertPullSecret(t, config, fakeClient, newSKRPullSecret)
		assertConfigMap(t, config, newSKRBootstrapperConfigMap, fakeClient)
		assertClusterTrustBundle(t, config, fakeClient, newSKRClusterTrustBundle)

	})

	t.Run("Should successfully apply bootstrapper ConfigMap, PullSecret and ClusterTrustBundle to the runtime cluster", func(t *testing.T) {
		// given
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
		fakeKcpClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(kcpPullSecret, kcpBootstrapperConfigMap, kcpClusterTrustBundle).Build()
		m := mocks.NewRuntimeClientGetter(t)
		m.On("Get", context.Background(), runtimeCR).Return(fakeClient, nil)

		// when
		configurator := NewConfigurator(fakeKcpClient, m, config)
		err := configurator.Configure(context.Background(), runtimeCR)

		// then
		m.AssertExpectations(t)
		require.NoError(t, err)

		assertPullSecret(t, config, fakeClient, skrPullSecret)
		assertConfigMap(t, config, skrBootstrapperConfigMap, fakeClient)
		assertClusterTrustBundle(t, config, fakeClient, skrClusterTrustBundle)
	})

	t.Run("Should successfully apply bootstrapper ConfigMap and PullSecret to the runtime cluster", func(t *testing.T) {
		// given
		configWithoutCTB := newConfig("test-registry-credentials", "", "test-runtime-bootstrapper-kcp-config")
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
		fakeKcpClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(kcpPullSecret, kcpBootstrapperConfigMap).Build()
		m := mocks.NewRuntimeClientGetter(t)
		m.On("Get", context.Background(), runtimeCR).Return(fakeClient, nil)

		// when
		configurator := NewConfigurator(fakeKcpClient, m, configWithoutCTB)
		err := configurator.Configure(context.Background(), runtimeCR)

		// then
		m.AssertExpectations(t)
		require.NoError(t, err)

		assertPullSecret(t, config, fakeClient, skrPullSecret)
		assertConfigMap(t, config, skrBootstrapperConfigMap, fakeClient)
	})

	t.Run("Should successfully apply only ConfigMap when PullSecret is not configured", func(t *testing.T) {
		// given
		configWithoutSecret := newConfig("", "", "test-runtime-bootstrapper-kcp-config")
		kcpClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(kcpBootstrapperConfigMap).Build()
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
		m := mocks.NewRuntimeClientGetter(t)
		m.On("Get", context.Background(), runtimeCR).Return(fakeClient, nil)

		// when
		configurator := NewConfigurator(kcpClient, m, configWithoutSecret)
		err := configurator.Configure(context.Background(), runtimeCR)

		// then
		m.AssertExpectations(t)
		require.NoError(t, err)

		assertConfigMap(t, config, skrBootstrapperConfigMap, fakeClient)
	})

	t.Run("Should return error when ConfigMap was not present on KCP", func(t *testing.T) {
		// given
		kcpClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects().Build()

		// when
		configurator := NewConfigurator(kcpClient, nil, config)
		err := configurator.Configure(context.Background(), v1.Runtime{})

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to prepare bootstrapper ConfigMap")
	})

	t.Run("Should return error when PullSecret was not found on KCP, but it was set as required", func(t *testing.T) {
		// given
		kcpClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(kcpBootstrapperConfigMap).Build()

		// when
		configurator := NewConfigurator(kcpClient, nil, config)
		err := configurator.Configure(context.Background(), v1.Runtime{})

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to prepare bootstrapper PullSecret")
	})

	t.Run("Should return error when ClusterTrustBundle was not found on KCP, but it was set as required", func(t *testing.T) {
		// given
		kcpClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(kcpPullSecret, kcpBootstrapperConfigMap).Build()

		// when
		configurator := NewConfigurator(kcpClient, nil, config)
		err := configurator.Configure(context.Background(), v1.Runtime{})

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to prepare ClusterTrustBundle")
	})

	t.Run("Should return error when unable to get runtime client", func(t *testing.T) {
		// given
		configWithoutCTB := newConfig("test-registry-credentials", "", "test-runtime-bootstrapper-kcp-config")
		kcpClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(kcpPullSecret, kcpBootstrapperConfigMap).Build()
		m := mocks.NewRuntimeClientGetter(t)
		m.On("Get", context.Background(), runtimeCR).Return(nil, errors.New("unable to get runtime client"))

		// when
		configurator := NewConfigurator(kcpClient, m, configWithoutCTB)
		err := configurator.Configure(context.Background(), runtimeCR)

		// then
		m.AssertExpectations(t)
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to get runtimeClient")
	})

	t.Run("Should return error when unable to apply ConfigMap to runtime cluster", func(t *testing.T) {
		// given
		configWithoutCTB := newConfig("test-registry-credentials", "", "test-runtime-bootstrapper-kcp-config")
		kcpClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(kcpPullSecret, kcpBootstrapperConfigMap).Build()
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
		configurator := NewConfigurator(kcpClient, m, configWithoutCTB)
		err := configurator.Configure(context.Background(), runtimeCR)

		// then
		m.AssertExpectations(t)
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to apply bootstrapper ConfigMap to runtime cluster")
	})

	t.Run("Should return error when unable to apply PullSecret to runtime cluster", func(t *testing.T) {
		// given
		configWithoutCTB := newConfig("test-registry-credentials", "", "test-runtime-bootstrapper-kcp-config")
		kcpClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(kcpPullSecret, kcpBootstrapperConfigMap).Build()
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
		configurator := NewConfigurator(kcpClient, m, configWithoutCTB)
		err := configurator.Configure(context.Background(), runtimeCR)

		// then
		m.AssertExpectations(t)
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to apply bootstrapper PullSecret to runtime cluster")
	})

	t.Run("Should return error when unable to apply ClusterTrustBundle to runtime cluster", func(t *testing.T) {
		// given
		kcpClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(kcpPullSecret, kcpBootstrapperConfigMap, kcpClusterTrustBundle).Build()
		m := mocks.NewRuntimeClientGetter(t)
		fakeRuntimeClient := fake.NewClientBuilder().WithScheme(scheme).
			WithInterceptorFuncs(interceptor.Funcs{
				Patch: func(ctx context.Context, client client.WithWatch, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
					if obj.GetObjectKind().GroupVersionKind().Kind == "ClusterTrustBundle" {
						return errors.New("unable to apply ClusterTrustBundle to the runtime")
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
		assert.ErrorContains(t, err, "failed to apply ClusterTrustBundle to runtime cluster")
	})
}

func newPullSecret(name, namespace string, dockerConfigJSON []byte) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: map[string][]byte{
			corev1.DockerConfigJsonKey: dockerConfigJSON,
		},
		Type: corev1.SecretTypeDockercfg,
	}
}

func newBootstrapperConfigMap(name, namespace string, data map[string]string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: data,
	}
}

func newClusterTrustBundle(name string, trustBundle string) *certificatesv1beta1.ClusterTrustBundle {
	return &certificatesv1beta1.ClusterTrustBundle{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: certificatesv1beta1.ClusterTrustBundleSpec{
			TrustBundle: trustBundle,
		},
	}
}

func assertPullSecret(t *testing.T, config Config, runtimeClient client.Client, expectedSecret *corev1.Secret) {
	var skrSecret corev1.Secret
	err := runtimeClient.Get(context.Background(), types.NamespacedName{Name: config.SKRConfig.PullSecretName, Namespace: "kyma-system"}, &skrSecret)
	require.NoError(t, err)

	assert.Equal(t, expectedSecret.Type, skrSecret.Type)
	assert.NotNil(t, skrSecret.Data[corev1.DockerConfigJsonKey])
	assert.Equal(t, expectedSecret.Data[corev1.DockerConfigJsonKey], skrSecret.Data[corev1.DockerConfigJsonKey])
	assert.Equal(t, expectedSecret.Name, skrSecret.Name)
	assert.Equal(t, expectedSecret.Namespace, skrSecret.Namespace)
}

func assertConfigMap(t *testing.T, config Config, expectedConfigMap *corev1.ConfigMap, runtimeClient client.Client) {
	var skrConfigMap corev1.ConfigMap
	err := runtimeClient.Get(context.Background(), types.NamespacedName{Name: config.SKRConfig.ConfigName, Namespace: "kyma-system"}, &skrConfigMap)
	require.NoError(t, err)

	assert.Equal(t, expectedConfigMap.Name, skrConfigMap.Name)
	assert.Equal(t, expectedConfigMap.Namespace, skrConfigMap.Namespace)

	assert.Equal(t, expectedConfigMap.Data, skrConfigMap.Data)
}

func assertClusterTrustBundle(t *testing.T, config Config, runtimeClient client.Client, expectedCTB *certificatesv1beta1.ClusterTrustBundle) {
	var skrCTB certificatesv1beta1.ClusterTrustBundle
	err := runtimeClient.Get(context.Background(), types.NamespacedName{Name: config.SKRConfig.ClusterTrustBundleName}, &skrCTB)
	require.NoError(t, err)

	assert.Equal(t, expectedCTB.Spec.TrustBundle, skrCTB.Spec.TrustBundle)
}
