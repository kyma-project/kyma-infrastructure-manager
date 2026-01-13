package extender

import (
	"testing"

	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/extender/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKubernetesRuntimeConfigExtender(t *testing.T) {
	t.Run("Runtime config should be added to shoot", func(t *testing.T) {
		// given
		shoot := testutils.FixEmptyGardenerShoot("test", "kcp-system")
		runtime := imv1.Runtime{}

		runtimeConfig := map[string]bool{
			"api/alpha": true,
			"api/beta":  false,
		}

		// when
		extender := NewKubernetesRuntimeConfigExtender(runtimeConfig)
		err := extender(runtime, &shoot)

		// then
		require.NoError(t, err)
		require.NotNil(t, shoot.Spec.Kubernetes.KubeAPIServer)
		assert.Equal(t, runtimeConfig, shoot.Spec.Kubernetes.KubeAPIServer.RuntimeConfig)
	})
}
