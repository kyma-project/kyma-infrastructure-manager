package alicloud

import (
	"encoding/json"
	"testing"

	"github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud/v1alpha1"
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
		givenNodesCidr string
		givenZoneNames []string
		expectedZones  []v1alpha1.Zone
	}{
		"Regular 10.250.0.0/16": {
			givenNodesCidr: "10.250.0.0/16",
			givenZoneNames: []string{
				"eu-central-1a",
				"eu-central-1b",
				"eu-central-1c",
			},
			expectedZones: []v1alpha1.Zone{
				{
					Name:    "eu-central-1a",
					Workers: "10.250.0.0/19",
				},
				{
					Name:    "eu-central-1b",
					Workers: "10.250.32.0/19",
				},
				{
					Name:    "eu-central-1c",
					Workers: "10.250.64.0/19",
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
			expectedZones: []v1alpha1.Zone{
				{
					Name:    "eu-central-1a",
					Workers: "10.180.0.0/26",
				},
				{
					Name:    "eu-central-1b",
					Workers: "10.180.0.64/26",
				},
				{
					Name:    "eu-central-1c",
					Workers: "10.180.0.128/26",
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
				assertIPRanges(t, tcase.expectedZones[i], actualZone)
			}
		})
	}
}

func assertIPRanges(t *testing.T, expectedZone v1alpha1.Zone, actualZone v1alpha1.Zone) {
	assert.Equal(t, expectedZone.Name, actualZone.Name)
	assert.Equal(t, expectedZone.Worker, actualZone.Worker)
	assert.Equal(t, expectedZone.Workers, actualZone.Workers)
}
