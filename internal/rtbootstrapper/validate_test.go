package rtbootstrapper

import (
	"context"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	util "k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
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
		Type: corev1.SecretTypeDockercfg,
	}

	scheme := runtime.NewScheme()

	t.Run("No errors on valid config", func(t *testing.T) {
		{
			// given
			config := Config{
				ManifestsPath:            "./testdata/manifests.yaml",
				DeploymentNamespacedName: "default/my-deployment",
				ConfigName:               "test-config",
			}

			util.Must(corev1.AddToScheme(scheme))
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(configMap).Build()

			// when
			err := NewValidator(config, fakeClient).Validate(context.Background())

			// then
			require.NoError(t, err)
		}

		{
			// given
			config := Config{
				ManifestsPath:            "./testdata/manifests.yaml",
				DeploymentNamespacedName: "default/my-deployment",
				ConfigName:               "test-config",
				PullSecretName:           "test-pull-secret",
			}

			scheme := runtime.NewScheme()
			util.Must(corev1.AddToScheme(scheme))
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(configMap, secret).Build()

			// when
			err := NewValidator(config, fakeClient).Validate(context.Background())

			// then
			require.NoError(t, err)
		}
	})

	t.Run("Empty manifest path", func(t *testing.T) {
		// given
		config := Config{}

		// when
		err := NewValidator(config, nil).Validate(context.Background())

		// then
		require.ErrorContains(t, err, "manifests path is required")
	})

	t.Run("Manifest file doesn't exist", func(t *testing.T) {
		// given
		config := Config{
			ManifestsPath: "non-existent-file.yaml",
		}

		// when
		err := NewValidator(config, nil).Validate(context.Background())

		// then
		require.ErrorContains(t, err, "non-existent-file.yaml")
	})

	t.Run("Deployment namespace incorrect", func(t *testing.T) {
		{
			// given
			config := Config{
				DeploymentNamespacedName: "/invalid-deployment-name",
				ManifestsPath:            "./testdata/manifests.yaml",
			}

			// when
			err := NewValidator(config, nil).Validate(context.Background())

			// then
			require.ErrorContains(t, err, "deployment namespaced name is invalid")
		}

		{
			// given
			config := Config{
				DeploymentNamespacedName: "invalid-deployment-name/",
				ManifestsPath:            "./testdata/manifests.yaml",
			}

			// when
			err := NewValidator(config, nil).Validate(context.Background())

			// then
			require.ErrorContains(t, err, "deployment namespaced name is invalid")
		}

		{
			// given
			config := Config{
				DeploymentNamespacedName: "",
				ManifestsPath:            "./testdata/manifests.yaml",
			}

			// when
			err := NewValidator(config, nil).Validate(context.Background())

			// then
			require.ErrorContains(t, err, "deployment namespaced name is invalid")
		}
	})

	t.Run("Configuration ConfigMap not exists", func(t *testing.T) {
		// given
		config := Config{
			DeploymentNamespacedName: "default/my-deployment",
			ManifestsPath:            "./testdata/manifests.yaml",
			ConfigName:               "test-config",
		}

		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects().Build()

		// when
		err := NewValidator(config, fakeClient).Validate(context.Background())

		// then
		require.ErrorContains(t, err, "unable to find Runtime Bootstrapper ConfigMap")
	})

	t.Run("Pull secret not exists", func(t *testing.T) {
		// given
		config := Config{
			DeploymentNamespacedName: "default/my-deployment",
			ManifestsPath:            "./testdata/manifests.yaml",
			ConfigName:               "test-config",
			PullSecretName:           "test-pull-secret",
		}

		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(configMap).Build()

		// when
		err := NewValidator(config, fakeClient).Validate(context.Background())

		// then
		require.ErrorContains(t, err, "unable to find Runtime Bootstrapper PullSecret")
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
			DeploymentNamespacedName: "default/my-deployment",
			ManifestsPath:            "./testdata/manifests.yaml",
			ConfigName:               "test-config",
			PullSecretName:           "test-pull-secret",
		}

		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(configMap, invalidSecret).Build()

		// when
		err := NewValidator(config, fakeClient).Validate(context.Background())

		// then
		require.ErrorContains(t, err, "pull secret has invalid type")
	})
}
