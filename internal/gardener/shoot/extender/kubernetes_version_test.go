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
		kubernetesVersionExtender := NewKubernetesVersionExtender("1.99")
		err := kubernetesVersionExtender(runtime, &shoot)

		// then
		require.NoError(t, err)
		assert.Equal(t, "1.99", shoot.Spec.Kubernetes.Version)
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
		kubernetesVersionExtender := NewKubernetesVersionExtender("1.99")
		err := kubernetesVersionExtender(runtime, &shoot)

		// then
		require.NoError(t, err)
		assert.Equal(t, "1.88", shoot.Spec.Kubernetes.Version)
	})
}
