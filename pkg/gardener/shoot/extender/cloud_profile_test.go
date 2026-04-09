package extender

import (
	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/extender/testutils"
	"testing"

	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/hyperscaler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtendWithCloudProfile(t *testing.T) {
	for _, testCase := range []struct {
		name            string
		providerType    string
		expectedProfile *gardener.CloudProfileReference
	}{
		{
			name:            "Set cloud profile name for aws",
			providerType:    hyperscaler.TypeAWS,
			expectedProfile: CreateCloudProfileReference(DefaultAWSCloudProfileName),
		},
		{
			name:            "Set cloud profile name for azure",
			providerType:    hyperscaler.TypeAzure,
			expectedProfile: CreateCloudProfileReference(DefaultAzureCloudProfileName),
		},
		{
			name:            "Set cloud profile for gcp",
			providerType:    hyperscaler.TypeGCP,
			expectedProfile: CreateCloudProfileReference(DefaultGCPCloudProfileName),
		},
		{
			name:            "Set cloud profile for openstack",
			providerType:    hyperscaler.TypeOpenStack,
			expectedProfile: CreateCloudProfileReference(DefaultOpenStackCloudProfileName),
		},
		{
			name:            "Set cloud profile for alicloud",
			providerType:    hyperscaler.TypeAlicloud,
			expectedProfile: CreateCloudProfileReference(DefaultAlicloudCloudProfileName),
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
			assert.Nil(t, shoot.Spec.CloudProfileName)
			assert.Equal(t, testCase.expectedProfile, shoot.Spec.CloudProfile)
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
