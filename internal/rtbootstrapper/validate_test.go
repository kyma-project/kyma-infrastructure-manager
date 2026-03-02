package rtbootstrapper

import (
	"context"
	"github.com/stretchr/testify/assert"
	certificatesv1beta1 "k8s.io/api/certificates/v1beta1"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	util "k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestValidations(t *testing.T) {
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-config",
			Namespace: "kcp-system",
		},
		Data: map[string]string{"some-key": "some-value"},
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pull-secret",
			Namespace: "kcp-system",
		},
		Data: map[string][]byte{".dockerconfigjson": []byte(`{"auths":{"test-registry.io":{"username":"test-user","password":"test-password","email":"test-email"}}}`)},
		Type: corev1.SecretTypeDockerConfigJson,
	}

	clusterTrustBundle := newClusterTrustBundle("test-trust-bundle", "data")

	validManifestsPath := "./testdata/manifests.yaml"
	invalidManifestsPath := "./testdata/invalid.yaml"

	manifestsConfigMapName := "test-manifests-config"
	invalidManifestsConfigMapName := "test-manifests-config"

	manifestsConfigMap, err := createManifestsConfigMap(validManifestsPath, manifestsConfigMapName, "kcp-system")
	require.NoError(t, err)

	manifestsConfigMapWithInvalidManifests, err := createManifestsConfigMap(invalidManifestsPath, invalidManifestsConfigMapName, "kcp-system")
	require.NoError(t, err)

	scheme := runtime.NewScheme()

	util.Must(corev1.AddToScheme(scheme))
	util.Must(certificatesv1beta1.AddToScheme(scheme))

	t.Run("No errors on valid config", func(t *testing.T) {
		{
			// given
			config := Config{
				KCPConfig: KCPConfig{
					ManifestsConfigMapName: manifestsConfigMapName,
					ConfigName:             "test-config",
				},
				SKRConfig: SKRConfig{
					DeploymentName: "my-deployment",
				},
			}

			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(configMap, manifestsConfigMap).Build()

			// when
			err := NewValidator(config, fakeClient).Validate(context.Background())

			// then
			assert.NoError(t, err)
		}

		{
			// given
			config := Config{
				KCPConfig: KCPConfig{
					ManifestsConfigMapName: manifestsConfigMapName,
					ConfigName:             "test-config",
					PullSecretName:         "test-pull-secret",
					ClusterTrustBundleName: "test-trust-bundle",
				},
				SKRConfig: SKRConfig{
					DeploymentName: "my-deployment",
				},
			}

			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(configMap, secret, manifestsConfigMap, clusterTrustBundle).Build()

			// when
			err := NewValidator(config, fakeClient).Validate(context.Background())

			// then
			assert.NoError(t, err)
		}
	})

	t.Run("Empty manifests config map name", func(t *testing.T) {
		// given
		config := Config{}

		// when
		err := NewValidator(config, nil).Validate(context.Background())

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "manifests config map name is required")
	})

	t.Run("Invalid YAML in manifests", func(t *testing.T) {
		// given
		fakeClient := fake.NewClientBuilder().WithObjects(configMap, manifestsConfigMapWithInvalidManifests).Build()

		config := Config{
			KCPConfig: KCPConfig{
				ManifestsConfigMapName: invalidManifestsConfigMapName,
				ConfigName:             "test-config",
			},
			SKRConfig: SKRConfig{
				DeploymentName: "default/my-deployment",
			},
		}

		validator := NewValidator(config, fakeClient)

		// when
		err := validator.Validate(context.Background())

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid YAML")
	})

	t.Run("Deployment name empty", func(t *testing.T) {
		// given
		fakeClient := fake.NewClientBuilder().WithObjects(configMap, manifestsConfigMap).Build()

		config := Config{
			KCPConfig: KCPConfig{
				ManifestsConfigMapName: manifestsConfigMapName,
			},
			SKRConfig: SKRConfig{
				DeploymentName: "",
			},
		}

		// when
		err := NewValidator(config, fakeClient).Validate(context.Background())

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "deployment name cannot be empty")
	})

	t.Run("Configuration ConfigMap not exists", func(t *testing.T) {
		// given
		config := Config{
			KCPConfig: KCPConfig{
				ManifestsConfigMapName: manifestsConfigMapName,
				ConfigName:             "test-config",
			},
			SKRConfig: SKRConfig{
				DeploymentName: "default/my-deployment",
			},
		}

		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(manifestsConfigMap).Build()

		// when
		err := NewValidator(config, fakeClient).Validate(context.Background())

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "unable to find Runtime Bootstrapper ConfigMap")
	})

	t.Run("ClusterTrustBundle not exists", func(t *testing.T) {
		// given
		config := Config{
			KCPConfig: KCPConfig{
				ManifestsConfigMapName: manifestsConfigMapName,
				ConfigName:             "test-config",
				ClusterTrustBundleName: "test-trust-bundle",
			},
			SKRConfig: SKRConfig{
				DeploymentName: "default/my-deployment",
			},
		}

		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(manifestsConfigMap).Build()

		// when
		err := NewValidator(config, fakeClient).Validate(context.Background())

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "unable to find Runtime Bootstrapper ConfigMap")
	})

	t.Run("Manifests ConfigMap not exists", func(t *testing.T) {
		// given
		config := Config{
			KCPConfig: KCPConfig{
				ManifestsConfigMapName: manifestsConfigMapName,
				ConfigName:             "test-config",
			},
			SKRConfig: SKRConfig{
				DeploymentName: "default/my-deployment",
			},
		}

		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(configMap).Build()

		// when
		err := NewValidator(config, fakeClient).Validate(context.Background())

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "unable to find Manifests ConfigMap in KCP cluster")
	})

	t.Run("Pull secret not exists", func(t *testing.T) {
		// given
		config := Config{
			KCPConfig: KCPConfig{
				ManifestsConfigMapName: manifestsConfigMapName,
				ConfigName:             "test-config",
				PullSecretName:         "test-pull-secret",
			},
			SKRConfig: SKRConfig{
				DeploymentName: "default/my-deployment",
			},
		}

		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(configMap, manifestsConfigMap).Build()

		// when
		err := NewValidator(config, fakeClient).Validate(context.Background())

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "unable to find Runtime Bootstrapper pull secret")
	})

	t.Run("Cluster Trust Bundle not exists", func(t *testing.T) {
		// given
		config := Config{
			KCPConfig: KCPConfig{
				ManifestsConfigMapName: manifestsConfigMapName,
				ConfigName:             "test-config",
				ClusterTrustBundleName: "some-test-trust-bundle",
			},
			SKRConfig: SKRConfig{
				DeploymentName: "default/my-deployment",
			},
		}

		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(configMap, manifestsConfigMap).Build()

		// when
		err := NewValidator(config, fakeClient).Validate(context.Background())

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "unable to find Cluster Trust Bundle")
	})

	t.Run("Pull secret has incorrect type", func(t *testing.T) {
		// given
		invalidSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pull-secret",
				Namespace: "kcp-system",
			},
			Data: map[string][]byte{".dockerconfigjson": []byte(`{"auths":{"test-registry.io":{"username":"test-user","password":"test-password","email":"test-email"}}}`)},
			Type: corev1.SecretTypeOpaque,
		}

		config := Config{
			KCPConfig: KCPConfig{
				ManifestsConfigMapName: manifestsConfigMapName,
				ConfigName:             "test-config",
				PullSecretName:         "test-pull-secret",
			},
			SKRConfig: SKRConfig{
				DeploymentName: "default/my-deployment",
			},
		}

		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(configMap, invalidSecret, manifestsConfigMap).Build()

		// when
		err := NewValidator(config, fakeClient).Validate(context.Background())

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "pull secret has invalid type")
	})
}
