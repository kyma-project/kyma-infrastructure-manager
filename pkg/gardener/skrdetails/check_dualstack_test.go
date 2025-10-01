package skrdetails

import (
	"testing"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/extender/testutils"
	"github.com/stretchr/testify/require"
)

func TestIsDualStackEnabled(t *testing.T) {
	t.Run("Should return false when Networking is nil", func(t *testing.T) {
		// given
		shoot := testutils.FixEmptyGardenerShoot("name", "namespace")

		// when
		got := IsDualStackEnabled(&shoot)

		// then
		require.Equal(t, false, got)
	})

	t.Run("Should return true when both IPv4 and IPv6 are present", func(t *testing.T) {
		// given
		shoot := testutils.FixEmptyGardenerShoot("name", "namespace")
		shoot.Spec.Networking = &gardener.Networking{
			IPFamilies: []gardener.IPFamily{gardener.IPFamilyIPv4, gardener.IPFamilyIPv6},
		}

		// when
		got := IsDualStackEnabled(&shoot)

		// then
		require.Equal(t, true, got)
	})

	t.Run("Should return false if configuration is invalid", func(t *testing.T) {
		// given
		shoot := testutils.FixEmptyGardenerShoot("name", "namespace")
		shoot.Spec.Networking = &gardener.Networking{
			IPFamilies: []gardener.IPFamily{gardener.IPFamilyIPv4, gardener.IPFamilyIPv6, "magic-ip-family"},
		}

		// when
		got := IsDualStackEnabled(&shoot)

		// then
		require.Equal(t, false, got)
	})
}
