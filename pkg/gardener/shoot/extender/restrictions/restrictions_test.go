package restrictions

import (
	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/extender/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestExtendWithAccessRestriction(t *testing.T) {
	for _, testCase := range []struct {
		name                       string
		platformRegion             string
		expectedAccessRestrictions []gardener.AccessRestrictionWithOptions
	}{
		{
			name:           "Should add eu-access-only access restriction if platform region is cf-eu11",
			platformRegion: "cf-eu11",
			expectedAccessRestrictions: []gardener.AccessRestrictionWithOptions{
				{
					AccessRestriction: gardener.AccessRestriction{
						Name: "eu-access-only",
					},
					Options: map[string]string{
						euAccessAddons: "true",
						euAccessNodes:  "true",
					},
				},
			},
		},
		{
			name:           "Should add eu-access-only access restriction if platform region is cf-ch20",
			platformRegion: "cf-ch20",
			expectedAccessRestrictions: []gardener.AccessRestrictionWithOptions{
				{
					AccessRestriction: gardener.AccessRestriction{
						Name: "eu-access-only",
					},
					Options: map[string]string{
						euAccessAddons: "true",
						euAccessNodes:  "true",
					},
				},
			},
		},
		{
			name:                       "Should do not add eu-access-only restriction if platform region is different than cf-eu11 or cf-ch20",
			platformRegion:             "test-region",
			expectedAccessRestrictions: nil,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			// given
			runtime := imv1.Runtime{
				Spec: imv1.RuntimeSpec{
					Shoot: imv1.RuntimeShoot{
						Name:           "test",
						PlatformRegion: testCase.platformRegion,
					},
				},
			}
			shoot := testutils.FixEmptyGardenerShoot("test", "dev")

			// when
			err := ExtendWithAccessRestriction()(runtime, &shoot)

			// then
			require.NoError(t, err)
			assert.Equal(t, testCase.expectedAccessRestrictions, shoot.Spec.AccessRestrictions)
		})
	}
}
