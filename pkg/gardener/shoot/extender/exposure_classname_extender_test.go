package extender

import (
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/extender/testutils"
	"testing"

	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/hyperscaler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtendWithExposureClassName(t *testing.T) {
	for _, testCase := range []struct {
		name                         string
		providerType                 string
		expectedExposureClassNameSet bool
	}{
		{
			name:                         "ExposureClassName not set for AWS",
			providerType:                 hyperscaler.TypeAWS,
			expectedExposureClassNameSet: false,
		},
		{
			name:                         "ExposureClassName set for OpenStack",
			providerType:                 hyperscaler.TypeOpenStack,
			expectedExposureClassNameSet: true,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			// given
			runtime := imv1.Runtime{
				Spec: imv1.RuntimeSpec{
					Shoot: imv1.RuntimeShoot{
						Name: "myshoot",
						Provider: imv1.Provider{
							Type: testCase.providerType,
						},
					},
				},
			}
			shoot := testutils.FixEmptyGardenerShoot("test", "dev")

			// when
			err := ExtendWithExposureClassName(runtime, &shoot)

			exposureClassNameSet := (shoot.Spec.ExposureClassName != nil) && (*shoot.Spec.ExposureClassName == "converged-cloud-internet")

			// then
			require.NoError(t, err)
			assert.Equal(t, testCase.expectedExposureClassNameSet, exposureClassNameSet)
		})
	}
}
