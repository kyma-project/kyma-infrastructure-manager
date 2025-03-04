package extensions

import (
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/extender/testutils"
	"k8s.io/utils/ptr"
	"testing"

	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNetworkingFilterExtender(t *testing.T) {
	t.Run("Enable networking-filter extension", func(t *testing.T) {
		// given
		runtimeShoot := getRuntimeWithNetworkingFilter(true)

		// when
		extension, err := NewNetworkFilterExtension(runtimeShoot.Spec.Security.Networking.Filter)

		// then
		require.NoError(t, err)
		assert.Equal(t, ptr.To(false), extension.Disabled)
		assert.Equal(t, NetworkFilterType, extension.Type)
	})

	t.Run("Disable networking-filter extension", func(t *testing.T) {
		// given
		runtimeShoot := getRuntimeWithNetworkingFilter(false)
		shoot := testutils.FixEmptyGardenerShoot("test", "dev")

		// when
		err := ExtendWithNetworkFilter(runtimeShoot, &shoot)

		// then
		require.NoError(t, err)
		assert.Equal(t, true, *shoot.Spec.Extensions[0].Disabled)
		assert.Equal(t, NetworkFilterType, shoot.Spec.Extensions[0].Type)
	})
}

func getRuntimeWithNetworkingFilter(enabled bool) imv1.Runtime {
	return imv1.Runtime{
		Spec: imv1.RuntimeSpec{
			Shoot: imv1.RuntimeShoot{
				Name: "myshoot",
			},
			Security: imv1.Security{
				Networking: imv1.NetworkingSecurity{
					Filter: imv1.Filter{
						Egress: imv1.Egress{
							Enabled: enabled,
						},
					},
				},
			},
		},
	}
}
