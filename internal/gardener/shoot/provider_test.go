package shoot

import (
	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestProviderExtender(t *testing.T) {

	for tname, runtimeShoot := range map[string]imv1.RuntimeShoot{
		"Create provider specific config for AWS": {
			Provider: imv1.Provider{
				Type: "aws",
				Workers: []gardener.Worker{
					{
						Name: "worker",
						Machine: gardener.Machine{
							Type: "m6i.large",
						},
						Minimum: 1,
						Maximum: 3,
						Zones: []string{
							"eu-central-1a",
							"eu-central-1b",
							"eu-central-1c",
						},
					},
				},
			},
		},
	} {
		t.Run(tname, func(t *testing.T) {
			// given
			shoot := fixEmptyGardenerShoot("cluster", "kcp-system")

			// when
			err := providerExtender(runtimeShoot, &shoot)

			// then
			require.NoError(t, err)

			assert.Equal(t, runtimeShoot.Provider.Type, shoot.Spec.Provider.Type)
			assert.Equal(t, runtimeShoot.Provider.Workers, shoot.Spec.Provider.Workers)
			assert.NotEmpty(t, shoot.Spec.Provider.InfrastructureConfig)
			assert.NotEmpty(t, shoot.Spec.Provider.InfrastructureConfig.Raw)
			assert.NotEmpty(t, shoot.Spec.Provider.ControlPlaneConfig)
			assert.NotEmpty(t, shoot.Spec.Provider.ControlPlaneConfig.Raw)
		})
	}

	t.Run("Return error for unknown provider", func(t *testing.T) {
		// given
		shoot := fixEmptyGardenerShoot("cluster", "kcp-system")
		runtimeShoot := imv1.RuntimeShoot{
			Provider: imv1.Provider{
				Type: "unknown",
			},
		}

		// when
		err := providerExtender(runtimeShoot, &shoot)

		// then
		require.Error(t, err)
	})
}
