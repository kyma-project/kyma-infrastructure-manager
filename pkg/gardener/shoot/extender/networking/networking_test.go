package networking

import (
	"testing"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/extender/testutils"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/hyperscaler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"
)

var expectedIPFamilies = []gardener.IPFamily{gardener.IPFamilyIPv4, gardener.IPFamilyIPv6}

func TestExtendWithNetworking(t *testing.T) {
	t.Run("Should configure an DualStackIP shoot if the provider is AWS, landscape supports DualStack and DualStack is enabled on RuntimeCR", func(t *testing.T) {
		// given
		runtime := prepareRuntimeStub(hyperscaler.TypeAWS, true)
		shoot := testutils.FixEmptyGardenerShoot("test-shoot", "kcp-dev")

		// when
		networkExtender := ExtendWithNetworking(true)
		err := networkExtender(runtime, &shoot)

		// then
		require.NoError(t, err)
		assert.Equal(t, expectedIPFamilies, shoot.Spec.Networking.IPFamilies)
	})

	t.Run("Should configure an DualStackIP shoot if the provider is GCP, landscape supports DualStack and DualStack is enabled on RuntimeCR", func(t *testing.T) {
		// given
		runtime := prepareRuntimeStub(hyperscaler.TypeGCP, true)
		shoot := testutils.FixEmptyGardenerShoot("test-shoot", "kcp-dev")

		// when
		networkExtender := ExtendWithNetworking(true)
		err := networkExtender(runtime, &shoot)

		// then
		require.NoError(t, err)
		assert.Equal(t, expectedIPFamilies, shoot.Spec.Networking.IPFamilies)
	})

	t.Run("Should do not configure an DualStackIP shoot if the provider is other than AWS or GCP", func(t *testing.T) {
		// given
		runtime := prepareRuntimeStub(hyperscaler.TypeAzure, true)
		shoot := testutils.FixEmptyGardenerShoot("test-shoot", "kcp-dev")
		shoot.Spec.Networking = &gardener.Networking{}

		// when
		networkExtender := ExtendWithNetworking(true)
		err := networkExtender(runtime, &shoot)

		// then
		require.NoError(t, err)
		assert.Nil(t, shoot.Spec.Networking.IPFamilies)
	})

	t.Run("Should do not configure an DualStackIP shoot if the landscape do not supports DualStack", func(t *testing.T) {
		// given
		runtime := prepareRuntimeStub(hyperscaler.TypeAWS, true)
		shoot := testutils.FixEmptyGardenerShoot("test-shoot", "kcp-dev")
		shoot.Spec.Networking = &gardener.Networking{}

		// when
		networkExtender := ExtendWithNetworking(false)
		err := networkExtender(runtime, &shoot)

		// then
		require.NoError(t, err)
		assert.Nil(t, shoot.Spec.Networking.IPFamilies)
	})

	t.Run("Should do not configure an DualStackIP shoot if the DualStack was disabled for RuntimeCR", func(t *testing.T) {
		// given
		runtime := prepareRuntimeStub(hyperscaler.TypeAWS, false)
		shoot := testutils.FixEmptyGardenerShoot("test-shoot", "kcp-dev")
		shoot.Spec.Networking = &gardener.Networking{}

		// when
		networkExtender := ExtendWithNetworking(true)
		err := networkExtender(runtime, &shoot)

		// then
		require.NoError(t, err)
		assert.Nil(t, shoot.Spec.Networking.IPFamilies)
	})

	t.Run("Should append DualStackIP configuration to existing networking config", func(t *testing.T) {
		// given
		runtime := prepareRuntimeStub(hyperscaler.TypeAWS, true)
		shoot := testutils.FixEmptyGardenerShoot("test-shoot", "kcp-dev")
		shoot.Spec.Networking = &gardener.Networking{
			Type: ptr.To("test-type"),
		}

		// when
		networkExtender := ExtendWithNetworking(true)
		err := networkExtender(runtime, &shoot)

		// then
		require.NoError(t, err)
		assert.Equal(t, expectedIPFamilies, shoot.Spec.Networking.IPFamilies)
	})
}

func prepareRuntimeStub(providerType string, dualStack bool) imv1.Runtime {
	return imv1.Runtime{
		Spec: imv1.RuntimeSpec{
			Shoot: imv1.RuntimeShoot{
				Provider: imv1.Provider{
					Type: providerType,
				},
				Networking: imv1.Networking{
					DualStack: ptr.To(dualStack),
				},
			},
		},
	}
}
