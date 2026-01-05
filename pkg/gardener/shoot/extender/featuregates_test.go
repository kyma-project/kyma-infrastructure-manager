package extender

import (
	"testing"

	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/extender/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

		// when
		extender := NewFeatureGatesExtender(apiServerFeatureGates)
		err := extender(runtime, &shoot)

		// then
		require.NoError(t, err)
		require.NotNil(t, shoot.Spec.Kubernetes.KubeAPIServer)
		assert.Equal(t, apiServerFeatureGates, shoot.Spec.Kubernetes.KubeAPIServer.FeatureGates)
	})
}
