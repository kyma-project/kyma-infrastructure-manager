package extender

import (
	"testing"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/extender/testutils"

	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/hyperscaler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtendWithCloudProfile(t *testing.T) {
	for _, testCase := range []struct {
		name            string
		providerType    string
		override        string
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
		{
			name:            "Set cloud profile for gdch",
			providerType:    hyperscaler.TypeGDCH,
			expectedProfile: CreateCloudProfileReference(DefaultGDCHCloudProfileName),
		},
		{
			name:            "Override cloud profile for gdch when provided",
			providerType:    hyperscaler.TypeGDCH,
			override:        "custom-gdch",
			expectedProfile: CreateCloudProfileReference("custom-gdch"),
		},
		{
			name:            "Ignore override for non-gdch providers",
			providerType:    hyperscaler.TypeAWS,
			override:        "custom-gdch",
			expectedProfile: CreateCloudProfileReference(DefaultAWSCloudProfileName),
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
			err := ExtendWithCloudProfile(testCase.override)(runtime, &shoot)

			// then
			require.NoError(t, err)
			// CloudProfileName is deprecated field, we verify it's nil to ensure we use CloudProfile instead
			//nolint:staticcheck // SA1019: CloudProfileName is deprecated, but we verify it's not set
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
		err := ExtendWithCloudProfile("")(runtime, &shoot)

		// then
		require.Error(t, err)
	})
}
