package extender

import (
	"testing"

	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"
)

func TestKubernetesVersionExtender(t *testing.T) {
	t.Run("Use default kubernetes version", func(t *testing.T) {
		// given
		shoot := fixEmptyGardenerShoot("test", "kcp-system")
		runtime := imv1.Runtime{}

		// when
		kubernetesVersionExtender := NewKubernetesExtender("1.99", "1.99")
		err := kubernetesVersionExtender(runtime, &shoot)

		// then
		require.NoError(t, err)
		assert.Equal(t, "1.99", shoot.Spec.Kubernetes.Version)
	})

	t.Run("Disable static token kubeconfig", func(t *testing.T) {
		// given
		shoot := fixEmptyGardenerShoot("test", "kcp-system")
		runtime := imv1.Runtime{}

		// when
		kubernetesVersionExtender := NewKubernetesExtender("1.99", "1.99")
		err := kubernetesVersionExtender(runtime, &shoot)

		// then
		require.NoError(t, err)
		assert.Equal(t, false, *shoot.Spec.Kubernetes.EnableStaticTokenKubeconfig)
	})

	t.Run("Use version provided in the Runtime CR", func(t *testing.T) {
		// given
		shoot := fixEmptyGardenerShoot("test", "kcp-system")
		runtime := imv1.Runtime{
			Spec: imv1.RuntimeSpec{
				Shoot: imv1.RuntimeShoot{
					Kubernetes: imv1.Kubernetes{
						Version: ptr.To("1.88"),
					},
				},
			},
		}

		// when
		kubernetesVersionExtender := NewKubernetesExtender("1.99", "1.88")
		err := kubernetesVersionExtender(runtime, &shoot)

		// then
		require.NoError(t, err)
		assert.Equal(t, "1.88", shoot.Spec.Kubernetes.Version)
	})

	t.Run("Use current Kubernetes version when it is greater than version provided in the Runtime CR", func(t *testing.T) {
		// given
		shoot := fixEmptyGardenerShoot("test", "kcp-system")
		runtime := imv1.Runtime{
			Spec: imv1.RuntimeSpec{
				Shoot: imv1.RuntimeShoot{
					Kubernetes: imv1.Kubernetes{
						Version: ptr.To("1.88.0"),
					},
				},
			},
		}

		// when
		kubernetesVersionExtender := NewKubernetesExtender("1.99.0", "2.0.0")
		err := kubernetesVersionExtender(runtime, &shoot)

		// then
		require.NoError(t, err)
		assert.Equal(t, "2.0.0", shoot.Spec.Kubernetes.Version)
	})

	t.Run("Override current Kubernetes version when it is smaller than version provided in the Runtime CR", func(t *testing.T) {
		// given
		shoot := fixEmptyGardenerShoot("test", "kcp-system")
		runtime := imv1.Runtime{
			Spec: imv1.RuntimeSpec{
				Shoot: imv1.RuntimeShoot{
					Kubernetes: imv1.Kubernetes{
						Version: ptr.To("1.99"),
					},
				},
			},
		}

		// when
		kubernetesVersionExtender := NewKubernetesExtender("1.88", "1.77")
		err := kubernetesVersionExtender(runtime, &shoot)

		// then
		require.NoError(t, err)
		assert.Equal(t, "1.99", shoot.Spec.Kubernetes.Version)
	})

	t.Run("Override current Kubernetes version when it is smaller than default version and Runtime CR has no version specified", func(t *testing.T) {
		// given
		shoot := fixEmptyGardenerShoot("test", "kcp-system")
		runtime := imv1.Runtime{
			Spec: imv1.RuntimeSpec{
				Shoot: imv1.RuntimeShoot{
					Kubernetes: imv1.Kubernetes{
						Version: nil,
					},
				},
			},
		}

		// when
		kubernetesVersionExtender := NewKubernetesExtender("1.88", "1.77")
		err := kubernetesVersionExtender(runtime, &shoot)

		// then
		require.NoError(t, err)
		assert.Equal(t, "1.88", shoot.Spec.Kubernetes.Version)
	})
}

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		name      string
		version1  string
		version2  string
		expected  int
		expectErr bool
	}{
		{
			name:     "version1 is less than version2",
			version1: "1.0.0",
			version2: "2.0.0",
			expected: -1,
		},
		{
			name:     "version1 is less than version2 with minor version",
			version1: "1.0.0",
			version2: "1.1.0",
			expected: -1,
		},
		{
			name:     "version1 is less than version2 with patch version",
			version1: "1.0.0",
			version2: "1.0.1",
			expected: -1,
		},
		{
			name:     "version1 is equal to version2",
			version1: "1.0.0",
			version2: "1.0.0",
			expected: 0,
		},
		{
			name:     "version1 is greater than version2",
			version1: "2.0.0",
			version2: "1.0.0",
			expected: 1,
		},
		{
			name:     "version1 is greater than version2 with minor version",
			version1: "1.1.0",
			version2: "1.0.0",
			expected: 1,
		},
		{
			name:     "version1 is greater than version2 with patch version",
			version1: "1.0.1",
			version2: "1.0.0",
			expected: 1,
		},
		{
			name:     "versions are in strange format 1",
			version1: "10.6.2800-118",
			version2: "10.6.2800-119",
			expected: -1,
		},
		{
			name:     "versions are in strange format 2",
			version1: "15.5.20240522+fips",
			version2: "15.5.20240524+fips",
			expected: -1,
		},
		{
			name:     "versions are in strange format 3",
			version1: "10.6.2800-118",
			version2: "10.6.2900-118",
			expected: -1,
		},
		{
			name:     "versions are in strange format 4",
			version1: "15.5.20240522+fips",
			version2: "15.6.20240522+fips",
			expected: -1,
		},
		{
			name:      "invalid version1",
			version1:  "invalid",
			version2:  "1.0.0",
			expectErr: true,
		},
		{
			name:      "empty version1",
			version1:  "",
			version2:  "1.0.0",
			expectErr: true,
		},
		{
			name:      "invalid version2",
			version1:  "1.0.0",
			version2:  "invalid",
			expectErr: true,
		},
		{
			name:      "empty version2",
			version1:  "1.0.0",
			version2:  "",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := compareVersions(tt.version1, tt.version2)
			if tt.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}
