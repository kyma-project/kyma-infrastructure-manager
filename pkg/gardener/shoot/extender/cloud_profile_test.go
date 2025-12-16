package extender

import (
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/extender/testutils"
	"testing"

	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/hyperscaler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"
)

func TestExtendWithCloudProfile(t *testing.T) {
	for _, testCase := range []struct {
		name            string
		providerType    string
		expectedProfile *string
	}{
		{
			name:            "Set cloud profile name for aws",
			providerType:    hyperscaler.TypeAWS,
			expectedProfile: ptr.To(DefaultAWSCloudProfileName),
		},
		{
			name:            "Set cloud profile name for azure",
			providerType:    hyperscaler.TypeAzure,
			expectedProfile: ptr.To(DefaultAzureCloudProfileName),
		},
		{
			name:            "Set cloud profile for gcp",
			providerType:    hyperscaler.TypeGCP,
			expectedProfile: ptr.To(DefaultGCPCloudProfileName),
		},
		{
			name:            "Set cloud profile for openstack",
			providerType:    hyperscaler.TypeOpenStack,
			expectedProfile: ptr.To(DefaultOpenStackCloudProfileName),
		},
		{
			name:            "Set cloud profile for alicloud",
			providerType:    hyperscaler.TypeAlicloud,
			expectedProfile: ptr.To(DefaultAlicloudCloudProfileName),
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
			err := ExtendWithCloudProfile(runtime, &shoot)

			// then
			require.NoError(t, err)
			assert.Equal(t, testCase.expectedProfile, shoot.Spec.CloudProfileName) //nolint:staticcheck
		})
	}

	t.Run("", func(t *testing.T) {
		// given
		runtime := imv1.Runtime{
			Spec: imv1.RuntimeSpec{
				Shoot: imv1.RuntimeShoot{
					Name: "myshoot",
					Provider: imv1.Provider{
						Type: "unknown",
					},
				},
			},
		}
		shoot := testutils.FixEmptyGardenerShoot("test", "dev")

		// when
		err := ExtendWithCloudProfile(runtime, &shoot)

		// then
		require.Error(t, err)
	})
}
