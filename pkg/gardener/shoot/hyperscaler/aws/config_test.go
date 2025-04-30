package aws

import (
	"encoding/json"
	"k8s.io/utils/ptr"
	"testing"

	"github.com/gardener/gardener-extension-provider-aws/pkg/apis/aws/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestControlPlaneConfig(t *testing.T) {
	t.Run("Create Control Plane config", func(t *testing.T) {
		// when
		controlPlaneConfigBytes, err := GetControlPlaneConfig(nil)

		// then
		require.NoError(t, err)

		var controlPlaneConfig v1alpha1.ControlPlaneConfig
		err = json.Unmarshal(controlPlaneConfigBytes, &controlPlaneConfig)
		assert.NoError(t, err)

		assert.Equal(t, apiVersion, controlPlaneConfig.APIVersion)
		assert.Equal(t, controlPlaneConfigKind, controlPlaneConfig.Kind)
	})
}

func TestInfrastructureConfig(t *testing.T) {
	for tname, tcase := range map[string]struct {
		givenNodesCidr   string
		givenZoneNames   []string
		expectedAwsZones []v1alpha1.Zone
	}{
		"Regular 10.250.0.0/16": {
			givenNodesCidr: "10.250.0.0/16",
			givenZoneNames: []string{
				"eu-central-1a",
				"eu-central-1b",
				"eu-central-1c",
			},
			expectedAwsZones: []v1alpha1.Zone{
				{
					Name:     "eu-central-1a",
					Workers:  "10.250.0.0/19",
					Public:   "10.250.32.0/20",
					Internal: "10.250.48.0/20",
				},
				{
					Name:     "eu-central-1b",
					Workers:  "10.250.64.0/19",
					Public:   "10.250.96.0/20",
					Internal: "10.250.112.0/20",
				},
				{
					Name:     "eu-central-1c",
					Workers:  "10.250.128.0/19",
					Public:   "10.250.160.0/20",
					Internal: "10.250.176.0/20",
				},
			},
		},
		"Regular 10.180.0.0/23": {
			givenNodesCidr: "10.180.0.0/23",
			givenZoneNames: []string{
				"eu-central-1a",
				"eu-central-1b",
				"eu-central-1c",
			},
			expectedAwsZones: []v1alpha1.Zone{
				{
					Name:     "eu-central-1a",
					Workers:  "10.180.0.0/26",
					Public:   "10.180.0.64/27",
					Internal: "10.180.0.96/27",
				},
				{
					Name:     "eu-central-1b",
					Workers:  "10.180.0.128/26",
					Public:   "10.180.0.192/27",
					Internal: "10.180.0.224/27",
				},
				{
					Name:     "eu-central-1c",
					Workers:  "10.180.1.0/26",
					Public:   "10.180.1.64/27",
					Internal: "10.180.1.96/27",
				},
			},
		},
	} {
		t.Run(tname, func(t *testing.T) {
			// when
			infrastructureConfigBytes, err := GetInfrastructureConfig(tcase.givenNodesCidr, tcase.givenZoneNames)

			// then
			assert.NoError(t, err)

			// when
			var infrastructureConfig v1alpha1.InfrastructureConfig
			err = json.Unmarshal(infrastructureConfigBytes, &infrastructureConfig)

			// then
			require.NoError(t, err)
			assert.Equal(t, apiVersion, infrastructureConfig.APIVersion)
			assert.Equal(t, infrastructureConfigKind, infrastructureConfig.Kind)

			assert.Equal(t, tcase.givenNodesCidr, *infrastructureConfig.Networks.VPC.CIDR)
			for i, actualZone := range infrastructureConfig.Networks.Zones {
				assertIPRanges(t, tcase.expectedAwsZones[i], actualZone)
			}
		})
	}
}

func TestInfrastructureConfigPatch(t *testing.T) {
	// given
	existingInfrastructureConfig := v1alpha1.InfrastructureConfig{
		EnableECRAccess: ptr.To(true),
		DualStack: &v1alpha1.DualStack{
			Enabled: true,
		},
		Networks: v1alpha1.Networks{
			VPC: v1alpha1.VPC{
				ID:               ptr.To("vpc-123456"),
				CIDR:             ptr.To("10.250.0.0/16"),
				GatewayEndpoints: []string{"one", "two"},
			},
			Zones: []v1alpha1.Zone{
				{
					Name:                  "eu-central-1a",
					Workers:               "10.250.0.0/19",
					Public:                "10.250.32.0/20",
					Internal:              "10.250.48.0/20",
					ElasticIPAllocationID: ptr.To("eipalloc-123456"),
				},
			},
		},
		IgnoreTags: &v1alpha1.IgnoreTags{
			Keys:        []string{"one", "two"},
			KeyPrefixes: []string{"p1", "p2"},
		},
	}

	givenZoneNames := []string{
		"eu-central-1a",
		"eu-central-1b",
		"eu-central-1c",
	}

	expectedAwsZones := []v1alpha1.Zone{
		{
			Name:     "eu-central-1a",
			Workers:  "10.250.0.0/19",
			Public:   "10.250.32.0/20",
			Internal: "10.250.48.0/20",
		},
		{
			Name:     "eu-central-1b",
			Workers:  "10.250.64.0/19",
			Public:   "10.250.96.0/20",
			Internal: "10.250.112.0/20",
		},
		{
			Name:     "eu-central-1c",
			Workers:  "10.250.128.0/19",
			Public:   "10.250.160.0/20",
			Internal: "10.250.176.0/20",
		},
	}

	givenNodesCidr := "10.250.0.0/16"

	t.Run("Create Infrastructure config for patch", func(t *testing.T) {
		existingInfrastructureConfigBytes, err := json.Marshal(existingInfrastructureConfig)
		require.NoError(t, err)

		// when
		infrastructureConfigBytes, err := GetInfrastructureConfigForPatch(givenNodesCidr, givenZoneNames, existingInfrastructureConfigBytes)

		// then
		assert.NoError(t, err)

		// when
		var infrastructureConfig v1alpha1.InfrastructureConfig
		err = json.Unmarshal(infrastructureConfigBytes, &infrastructureConfig)

		// then
		require.NoError(t, err)
		assert.Equal(t, apiVersion, infrastructureConfig.APIVersion)
		assert.Equal(t, infrastructureConfigKind, infrastructureConfig.Kind)

		assert.Equal(t, givenNodesCidr, *infrastructureConfig.Networks.VPC.CIDR)
		for i, actualZone := range infrastructureConfig.Networks.Zones {
			assertIPRanges(t, expectedAwsZones[i], actualZone)
		}

		assert.Equal(t, existingInfrastructureConfig.EnableECRAccess, infrastructureConfig.EnableECRAccess)
		assert.Equal(t, existingInfrastructureConfig.DualStack, infrastructureConfig.DualStack)
		assert.Equal(t, existingInfrastructureConfig.Networks.VPC.ID, infrastructureConfig.Networks.VPC.ID)
		assert.Equal(t, existingInfrastructureConfig.Networks.VPC.GatewayEndpoints, infrastructureConfig.Networks.VPC.GatewayEndpoints)
		assert.Equal(t, existingInfrastructureConfig.IgnoreTags, infrastructureConfig.IgnoreTags)
		assert.Equal(t, existingInfrastructureConfig.Networks.Zones[0].ElasticIPAllocationID, infrastructureConfig.Networks.Zones[0].ElasticIPAllocationID)
		assert.Equal(t, existingInfrastructureConfig.Networks.Zones[0].Name, infrastructureConfig.Networks.Zones[0].Name)
		assert.Equal(t, existingInfrastructureConfig.Networks.Zones[0].Internal, infrastructureConfig.Networks.Zones[0].Internal)
		assert.Equal(t, existingInfrastructureConfig.Networks.Zones[0].Public, infrastructureConfig.Networks.Zones[0].Public)
		assert.Equal(t, existingInfrastructureConfig.Networks.Zones[0].Workers, infrastructureConfig.Networks.Zones[0].Workers)
	})
}

func assertIPRanges(t *testing.T, expectedZone v1alpha1.Zone, actualZone v1alpha1.Zone) {
	assert.Equal(t, expectedZone.Name, actualZone.Name)
	assert.Equal(t, expectedZone.Internal, actualZone.Internal)
	assert.Equal(t, expectedZone.Workers, actualZone.Workers)
	assert.Equal(t, expectedZone.Public, actualZone.Public)
}

func TestWorkerConfig(t *testing.T) {
	t.Run("Create worker config", func(t *testing.T) {
		// when
		configBytes, err := GetWorkerConfig()

		// then
		require.NoError(t, err)
		var config v1alpha1.WorkerConfig

		err = json.Unmarshal(configBytes, &config)
		require.NoError(t, err)

		assert.Equal(t, awsIMDSv2HTTPPutResponseHopLimit, *config.InstanceMetadataOptions.HTTPPutResponseHopLimit)
		assert.Equal(t, v1alpha1.HTTPTokensRequired, *config.InstanceMetadataOptions.HTTPTokens)
	})
}
