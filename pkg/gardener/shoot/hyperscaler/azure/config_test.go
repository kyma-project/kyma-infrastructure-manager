package azure

import (
	"encoding/json"
	"k8s.io/utils/ptr"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	DefaultNodesCIDR = "10.250.0.0/22"
)

func TestControlPlaneConfig(t *testing.T) {
	t.Run("Create Control Plane config", func(t *testing.T) {
		// when
		controlPlaneConfigBytes, err := GetControlPlaneConfig(nil)

		// then
		require.NoError(t, err)

		var controlPlaneConfig ControlPlaneConfig
		err = json.Unmarshal(controlPlaneConfigBytes, &controlPlaneConfig)
		assert.NoError(t, err)

		assert.Equal(t, apiVersion, controlPlaneConfig.APIVersion)
		assert.Equal(t, controlPlaneConfigKind, controlPlaneConfig.Kind)
	})
}

func TestInfrastructureConfig(t *testing.T) {
	for tname, tcase := range map[string]struct {
		givenVnetCidr      string
		givenZoneNames     []string
		expectedAzureZones []Zone
		expectedIsZoned    bool
	}{
		"Zoned setup for 1 zone with default CIDR 10.250.0.0/22": {
			expectedIsZoned: true,
			givenVnetCidr:   DefaultNodesCIDR,
			givenZoneNames: []string{
				"1",
			},
			expectedAzureZones: []Zone{
				{
					Name: 1,
					CIDR: "10.250.0.0/25",
					NatGateway: &NatGateway{
						Enabled:                      true,
						IdleConnectionTimeoutMinutes: defaultConnectionTimeOutMinutes,
					},
				},
			},
		},
		"Zoned setup for 2 zones and default CIDR 10.250.0.0/22": {
			expectedIsZoned: true,
			givenVnetCidr:   DefaultNodesCIDR,
			givenZoneNames: []string{
				"2",
				"3",
			},
			expectedAzureZones: []Zone{
				{
					Name: 2,
					CIDR: "10.250.0.0/25",
					NatGateway: &NatGateway{
						Enabled:                      true,
						IdleConnectionTimeoutMinutes: defaultConnectionTimeOutMinutes,
					},
				},
				{
					Name: 3,
					CIDR: "10.250.0.128/25",
					NatGateway: &NatGateway{
						Enabled:                      true,
						IdleConnectionTimeoutMinutes: defaultConnectionTimeOutMinutes,
					},
				},
			},
		},
		"Zoned setup for 3 zones and default CIDR 10.250.0.0/22": {
			expectedIsZoned: true,
			givenVnetCidr:   DefaultNodesCIDR,
			givenZoneNames: []string{
				"1",
				"2",
				"3",
			},
			expectedAzureZones: []Zone{
				{
					Name: 1,
					CIDR: "10.250.0.0/25",
					NatGateway: &NatGateway{
						Enabled:                      true,
						IdleConnectionTimeoutMinutes: defaultConnectionTimeOutMinutes,
					},
				},
				{
					Name: 2,
					CIDR: "10.250.0.128/25",
					NatGateway: &NatGateway{
						Enabled:                      true,
						IdleConnectionTimeoutMinutes: defaultConnectionTimeOutMinutes,
					},
				},
				{
					Name: 3,
					CIDR: "10.250.1.0/25",
					NatGateway: &NatGateway{
						Enabled:                      true,
						IdleConnectionTimeoutMinutes: defaultConnectionTimeOutMinutes,
					},
				},
			},
		},
	} {
		t.Run(tname, func(t *testing.T) {
			// when
			infrastructureConfigBytes, err := GetInfrastructureConfig(tcase.givenVnetCidr, tcase.givenZoneNames)

			// then
			assert.NoError(t, err)

			// when
			var infrastructureConfig InfrastructureConfig
			err = json.Unmarshal(infrastructureConfigBytes, &infrastructureConfig)

			// then
			require.NoError(t, err)
			assert.Equal(t, apiVersion, infrastructureConfig.APIVersion)
			assert.Equal(t, infrastructureConfigKind, infrastructureConfig.Kind)

			assert.Equal(t, tcase.givenVnetCidr, *infrastructureConfig.Networks.VNet.CIDR)

			assert.Equal(t, tcase.givenVnetCidr, *infrastructureConfig.Networks.VNet.CIDR)
			assert.Equal(t, true, infrastructureConfig.Zoned)

			for i, actualZone := range infrastructureConfig.Networks.Zones {
				assertAzureZoneCidrs(t, tcase.expectedAzureZones[i], actualZone)
			}
		})
	}
}

func TestInfrastructureConfigPatch(t *testing.T) {
	// given
	existingInfrastructureConfig := InfrastructureConfig{
		ResourceGroup: &ResourceGroup{
			Name: "existing-rg",
		},
		Networks: NetworkConfig{
			VNet: VNet{
				Name:          ptr.To("existing-vnet"),
				CIDR:          ptr.To("10.250.0.0/22"),
				ResourceGroup: ptr.To("existing-rg"),
			},
			Zones: []Zone{
				{
					Name:             1,
					CIDR:             "10.250.0.0/25",
					ServiceEndpoints: []string{"endpoint1", "endpoint2"},
					NatGateway: &NatGateway{
						Enabled:                      true,
						IdleConnectionTimeoutMinutes: defaultConnectionTimeOutMinutes,
					},
				},
			},
			ServiceEndpoints: []string{"endpoint11", "endpoint22"},
			NatGateway: &NatGateway{
				Enabled:                      true,
				IdleConnectionTimeoutMinutes: defaultConnectionTimeOutMinutes,
				Zone:                         1,
				IPAddresses: []PublicIPReference{
					{
						Name:          "existing-ip",
						ResourceGroup: "existing-rg",
						Zone:          1,
					},
				},
			},
		},
	}

	givenZoneNames := []string{
		"1",
		"2",
		"3",
	}

	expectedAzureZones := []Zone{
		{
			Name: 1,
			CIDR: "10.250.0.0/25",
			NatGateway: &NatGateway{
				Enabled:                      true,
				IdleConnectionTimeoutMinutes: defaultConnectionTimeOutMinutes,
			},
		},
		{
			Name: 2,
			CIDR: "10.250.0.128/25",
			NatGateway: &NatGateway{
				Enabled:                      true,
				IdleConnectionTimeoutMinutes: defaultConnectionTimeOutMinutes,
			},
		},
		{
			Name: 3,
			CIDR: "10.250.1.0/25",
			NatGateway: &NatGateway{
				Enabled:                      true,
				IdleConnectionTimeoutMinutes: defaultConnectionTimeOutMinutes,
			},
		},
	}

	expectedIsZoned := true
	givenVnetCidr := DefaultNodesCIDR

	t.Run("Create Infrastructure config for patch", func(t *testing.T) {
		// given
		existingInfrastructureConfigBytes, err := json.Marshal(existingInfrastructureConfig)
		require.NoError(t, err)

		// when
		infrastructureConfigBytes, err := GetInfrastructureConfigForPatch(givenVnetCidr, givenZoneNames, existingInfrastructureConfigBytes)

		// then
		assert.NoError(t, err)

		// when
		var infrastructureConfig InfrastructureConfig
		err = json.Unmarshal(infrastructureConfigBytes, &infrastructureConfig)

		// then
		require.NoError(t, err)
		assert.Equal(t, apiVersion, infrastructureConfig.APIVersion)
		assert.Equal(t, infrastructureConfigKind, infrastructureConfig.Kind)

		assert.Equal(t, givenVnetCidr, *infrastructureConfig.Networks.VNet.CIDR)
		assert.Equal(t, true, infrastructureConfig.Zoned)

		for i, actualZone := range infrastructureConfig.Networks.Zones {
			assertAzureZoneCidrs(t, expectedAzureZones[i], actualZone)
		}

		assert.Equal(t, expectedIsZoned, infrastructureConfig.Zoned)
		assert.Equal(t, existingInfrastructureConfig.ResourceGroup, infrastructureConfig.ResourceGroup)
		assert.Equal(t, existingInfrastructureConfig.Networks.VNet, infrastructureConfig.Networks.VNet)
		assert.Equal(t, existingInfrastructureConfig.Networks.ServiceEndpoints, infrastructureConfig.Networks.ServiceEndpoints)
		assert.Equal(t, existingInfrastructureConfig.Networks.NatGateway, infrastructureConfig.Networks.NatGateway)
		assert.Equal(t, existingInfrastructureConfig.Networks.Zones[0].ServiceEndpoints, infrastructureConfig.Networks.Zones[0].ServiceEndpoints)
		assert.Equal(t, existingInfrastructureConfig.Networks.Zones[0].NatGateway, infrastructureConfig.Networks.Zones[0].NatGateway)
		assert.Equal(t, existingInfrastructureConfig.Networks.Zones[0].Name, infrastructureConfig.Networks.Zones[0].Name)
		assert.Equal(t, existingInfrastructureConfig.Networks.Zones[0].CIDR, infrastructureConfig.Networks.Zones[0].CIDR)
	})
}

func assertAzureZoneCidrs(t *testing.T, expectedZone Zone, actualZone Zone) {
	assert.Equal(t, expectedZone.Name, actualZone.Name)
	assert.Equal(t, expectedZone.CIDR, actualZone.CIDR)
	assert.Equal(t, expectedZone.NatGateway.Enabled, actualZone.NatGateway.Enabled)
	assert.Equal(t, expectedZone.NatGateway.IdleConnectionTimeoutMinutes, actualZone.NatGateway.IdleConnectionTimeoutMinutes)
}
