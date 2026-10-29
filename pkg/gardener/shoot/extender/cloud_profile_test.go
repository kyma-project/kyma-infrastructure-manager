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
		name              string
		providerType      string
		gdchOverride      string
		openstackOverride string
		expectedProfile   *gardener.CloudProfileReference
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
			name:              "Override cloud profile for openstack when provided",
			providerType:      hyperscaler.TypeOpenStack,
			openstackOverride: "converged-cloud-iaas-de5955",
			expectedProfile:   CreateCloudProfileReference("converged-cloud-iaas-de5955"),
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
			gdchOverride:    "custom-gdch",
			expectedProfile: CreateCloudProfileReference("custom-gdch"),
		},
		{
			name:              "Ignore overrides for non-matching providers",
			providerType:      hyperscaler.TypeAWS,
			gdchOverride:      "custom-gdch",
			openstackOverride: "custom-openstack",
			expectedProfile:   CreateCloudProfileReference(DefaultAWSCloudProfileName),
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
			err := ExtendWithCloudProfile("", testCase.gdchOverride, testCase.openstackOverride)(runtime, &shoot)

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
		err := ExtendWithCloudProfile("", "", "")(runtime, &shoot)

		// then
		require.Error(t, err)
	})
}

func TestCloudProfileExtenderForPatch(t *testing.T) {
	// The cloud profile is immutable in Gardener. On patch the existing shoot's
	// cloud profile must always be preserved and re-stated, never re-derived from
	// config, so a changed config value can never rewrite it on an existing shoot.
	for _, testCase := range []struct {
		name                     string
		providerType             string
		existingCloudProfileName string
		gdchOverride             string
		openstackOverride        string
		expectedProfile          *gardener.CloudProfileReference
		expectError              bool
	}{
		{
			name:                     "Preserve existing cloud profile on patch, ignoring config override",
			providerType:             hyperscaler.TypeOpenStack,
			existingCloudProfileName: "converged-cloud-iaas-de5955",
			openstackOverride:        "converged-cloud-different",
			expectedProfile:          CreateCloudProfileReference("converged-cloud-iaas-de5955"),
		},
		{
			name:                     "Preserve existing cloud profile on patch for non-OpenStack provider",
			providerType:             hyperscaler.TypeAWS,
			existingCloudProfileName: "custom-aws-profile",
			expectedProfile:          CreateCloudProfileReference("custom-aws-profile"),
		},
		{
			name:                     "Keep existing cloud profile on patch when it matches the config-derived value",
			providerType:             hyperscaler.TypeGDCH,
			existingCloudProfileName: DefaultGDCHCloudProfileName,
			expectedProfile:          CreateCloudProfileReference(DefaultGDCHCloudProfileName),
		},
		{
			name:              "Fall back to config-derived profile when existing is empty",
			providerType:      hyperscaler.TypeOpenStack,
			openstackOverride: "converged-cloud-fallback",
			expectedProfile:   CreateCloudProfileReference("converged-cloud-fallback"),
		},
		{
			name:                     "Return error on patch when provider has no cloud profile",
			providerType:             "unknown",
			existingCloudProfileName: "some-existing-profile",
			expectError:              true,
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
			err := ExtendWithCloudProfile(testCase.existingCloudProfileName, testCase.gdchOverride, testCase.openstackOverride)(runtime, &shoot)

			// then
			if testCase.expectError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			// CloudProfileName is deprecated field, we verify it's nil to ensure we use CloudProfile instead
			//nolint:staticcheck // SA1019: CloudProfileName is deprecated, but we verify it's not set
			assert.Nil(t, shoot.Spec.CloudProfileName)
			assert.Equal(t, testCase.expectedProfile, shoot.Spec.CloudProfile)
		})
	}
}
