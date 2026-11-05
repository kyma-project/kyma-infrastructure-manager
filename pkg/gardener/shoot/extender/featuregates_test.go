package extender

import (
	"testing"

	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/extender/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"
)

func TestFeatureGatesExtender(t *testing.T) {
	t.Run("Feature gates should be added to shoot", func(t *testing.T) {
		// given
		shoot := testutils.FixEmptyGardenerShoot("test", "kcp-system")
		runtime := imv1.Runtime{}

		apiServerFeatureGates := map[string]bool{
			"SomeFeature":    true,
			"AnotherFeature": false,
		}

		kubeletFeatureGates := map[string]bool{
			"SomeOtherFeature": true,
		}

		// when
		extender := NewFeatureGatesExtender(apiServerFeatureGates, kubeletFeatureGates, "1.34", "")
		err := extender(runtime, &shoot)

		// then
		require.NoError(t, err)
		require.NotNil(t, shoot.Spec.Kubernetes.KubeAPIServer)
		assert.Equal(t, apiServerFeatureGates, shoot.Spec.Kubernetes.KubeAPIServer.FeatureGates)
		assert.Equal(t, kubeletFeatureGates, shoot.Spec.Kubernetes.Kubelet.FeatureGates)
	})

	for _, tc := range []struct {
		name            string
		runtimeVersion  *string
		defaultVersion  string
		currentVersion  string
		expectedEnabled bool
	}{
		{name: "default version lower than 1.35", defaultVersion: "1.34.2", expectedEnabled: false},
		{name: "default version equal to 1.35", defaultVersion: "1.35", expectedEnabled: true},
		{name: "default version greater than 1.35", defaultVersion: "1.36.1", expectedEnabled: true},
		{name: "empty runtime version falls back to default", runtimeVersion: ptr.To(""), defaultVersion: "1.35.0", expectedEnabled: true},
		{name: "runtime version lower than 1.35 overrides default", runtimeVersion: ptr.To("1.33.5"), defaultVersion: "1.35.0", expectedEnabled: false},
		{name: "runtime version greater than 1.35 overrides default", runtimeVersion: ptr.To("1.35.1"), defaultVersion: "1.34.0", expectedEnabled: true},
		{name: "invalid version", runtimeVersion: ptr.To("invalid"), defaultVersion: "1.35.0", expectedEnabled: false},
		{name: "auto-updated shoot version greater than default", defaultVersion: "1.34.2", currentVersion: "1.35.1", expectedEnabled: true},
		{name: "auto-updated shoot version greater than runtime version", runtimeVersion: ptr.To("1.34.2"), defaultVersion: "1.34.2", currentVersion: "1.35.0", expectedEnabled: true},
		{name: "current shoot version lower than runtime version", runtimeVersion: ptr.To("1.35.0"), defaultVersion: "1.34.2", currentVersion: "1.34.2", expectedEnabled: true},
		{name: "current shoot version lower than 1.35", defaultVersion: "1.34.2", currentVersion: "1.34.3", expectedEnabled: false},
	} {
		t.Run("ClusterTrustBundle kubelet feature gate: "+tc.name, func(t *testing.T) {
			// given
			shoot := testutils.FixEmptyGardenerShoot("test", "kcp-system")
			runtime := imv1.Runtime{}
			runtime.Spec.Shoot.Kubernetes.Version = tc.runtimeVersion

			kubeletFeatureGates := map[string]bool{
				"ClusterTrustBundleProjection": true,
				"ClusterTrustBundle":           true,
			}

			// when
			extender := NewFeatureGatesExtender(map[string]bool{}, kubeletFeatureGates, tc.defaultVersion, tc.currentVersion)
			err := extender(runtime, &shoot)

			// then
			require.NoError(t, err)
			assert.True(t, shoot.Spec.Kubernetes.Kubelet.FeatureGates["ClusterTrustBundleProjection"])
			_, found := shoot.Spec.Kubernetes.Kubelet.FeatureGates["ClusterTrustBundle"]
			assert.Equal(t, tc.expectedEnabled, found)
			// the input map shared between runtimes must not be modified
			assert.Len(t, kubeletFeatureGates, 2)
		})
	}
}
