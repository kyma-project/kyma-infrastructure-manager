package rtbootstrapper

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestValidations(t *testing.T) {
	t.Run("Empty manifest path", func(t *testing.T) {
		// given
		config := Config{}

		// when
		_, err := NewInstaller(config, nil, nil, nil)
		require.ErrorContains(t, err, "manifests path is required")
	})

	t.Run("Manifest file doesn't exist", func(t *testing.T) {
		// given
		config := Config{
			ManifestsPath: "non-existent-file.yaml",
		}

		// when
		_, err := NewInstaller(config, nil, nil, nil)

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
			_, err := NewInstaller(config, nil, nil, nil)

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
			_, err := NewInstaller(config, nil, nil, nil)

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
			_, err := NewInstaller(config, nil, nil, nil)

			// then
			require.ErrorContains(t, err, "deployment namespaced name is invalid")
		}
	})
}
