package networking

import (
	"testing"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/extender/testutils"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/hyperscaler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var expectedIPFamilies = []gardener.IPFamily{gardener.IPFamilyIPv4, gardener.IPFamilyIPv6}

func TestExtendWithNetworking(t *testing.T) {
	t.Run("Should configure an DualStackIP shoot if the provider is AWS", func(t *testing.T) {
		// given
		runtime := prepareRuntimeStub(hyperscaler.TypeAWS)
		shoot := testutils.FixEmptyGardenerShoot("test-shoot", "kcp-dev")

		// when
		networkExtender := ExtendWithNetworking()
		err := networkExtender(runtime, &shoot)

		// then
		require.NoError(t, err)
		assert.Equal(t, expectedIPFamilies, shoot.Spec.Networking.IPFamilies)
	})

	t.Run("Should configure an DualStackIP shoot if the provider is GCP", func(t *testing.T) {
		// given
		runtime := prepareRuntimeStub(hyperscaler.TypeGCP)
		shoot := testutils.FixEmptyGardenerShoot("test-shoot", "kcp-dev")

		// when
		networkExtender := ExtendWithNetworking()
		err := networkExtender(runtime, &shoot)

		// then
		require.NoError(t, err)
		assert.Equal(t, expectedIPFamilies, shoot.Spec.Networking.IPFamilies)
	})

	t.Run("Should do not configure an DualStackIP shoot if the provider is other than AWS or GCP", func(t *testing.T) {
		// given
		runtime := prepareRuntimeStub(hyperscaler.TypeAzure)
		shoot := testutils.FixEmptyGardenerShoot("test-shoot", "kcp-dev")
		shoot.Spec.Networking = &gardener.Networking{}

		// when
		networkExtender := ExtendWithNetworking()
		err := networkExtender(runtime, &shoot)

		// then
		require.NoError(t, err)
		assert.Nil(t, shoot.Spec.Networking.IPFamilies)
	})
}

func prepareRuntimeStub(providerType string) imv1.Runtime {
	return imv1.Runtime{
		Spec: imv1.RuntimeSpec{
			Shoot: imv1.RuntimeShoot{
				Provider: imv1.Provider{
					Type: providerType,
				},
			},
		},
	}
}
