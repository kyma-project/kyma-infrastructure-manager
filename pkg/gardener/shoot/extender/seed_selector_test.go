package extender

import (
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/extender/testutils"
	"testing"

	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSeedSelectorExtender(t *testing.T) {
	t.Run("Add and populate seed selector field if RuntimeCR has SeedInSameRegionFlag set to true", func(t *testing.T) {
		// given
		runtimeShoot := getRuntimeWithSeedInSameRegionFlag(true)
		shoot := testutils.FixEmptyGardenerShoot("test", "dev")

		// when
		err := ExtendWithSeedSelector(runtimeShoot, &shoot)

		// then
		require.NoError(t, err)
		assert.NotNil(t, shoot.Spec.SeedSelector)
		assert.Equal(t, runtimeShoot.Spec.Shoot.Region, shoot.Spec.SeedSelector.LabelSelector.MatchLabels[seedRegionSelectorLabel])
	})

	t.Run("Don't add seed selector field if RuntimeCR has SeedInSameRegionFlag set to false", func(t *testing.T) {
		// given
		runtimeShoot := getRuntimeWithSeedInSameRegionFlag(false)
		shoot := testutils.FixEmptyGardenerShoot("test", "dev")

		// when
		err := ExtendWithSeedSelector(runtimeShoot, &shoot)

		// then
		require.NoError(t, err)
		assert.Nil(t, shoot.Spec.SeedSelector)
	})

	t.Run("Don't add seed selector field if RuntimeCR has no SeedInSameRegionFlag set", func(t *testing.T) {
		// given
		runtimeShoot := getRuntimeWithoutSeedInSameRegionFlag()
		shoot := testutils.FixEmptyGardenerShoot("test", "dev")

		// when
		err := ExtendWithSeedSelector(runtimeShoot, &shoot)

		// then
		require.NoError(t, err)
		assert.Nil(t, shoot.Spec.SeedSelector)
	})
}

func getRuntimeWithSeedInSameRegionFlag(enabled bool) imv1.Runtime {
	return imv1.Runtime{
		Spec: imv1.RuntimeSpec{
			Shoot: imv1.RuntimeShoot{
				Name:                "myshoot",
				EnforceSeedLocation: &enabled,
				Region:              "far-far-away",
			},
		},
	}
}

func getRuntimeWithoutSeedInSameRegionFlag() imv1.Runtime {
	return imv1.Runtime{
		Spec: imv1.RuntimeSpec{
			Shoot: imv1.RuntimeShoot{
				Name:   "myshoot",
				Region: "far-far-away",
			},
		},
	}
}
