package azure

import (
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/hyperscaler/networking"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestAzureZonesWithCustomNodeIPRange(t *testing.T) {

	for tname, tcase := range map[string]struct {
		givenNodesCidr     string
		givenZoneNames     []string
		expectedAzureZones []Zone
	}{
		"Azure no zone (azure lite) and Default 10.250.0.0/16": {
			givenNodesCidr: "10.250.0.0/16",
			givenZoneNames: []string{},
		},
		"Azure one zone and Default 10.250.0.0/16": {
			givenNodesCidr: "10.250.0.0/16",
			givenZoneNames: []string{"1"},
			expectedAzureZones: []Zone{
				{
					Name: 1,
					CIDR: "10.250.0.0/19",
				},
			},
		},
		"Azure two zones and Default 10.250.0.0/16": {
			givenNodesCidr: "10.250.0.0/16",
			givenZoneNames: []string{"2", "3"},
			expectedAzureZones: []Zone{
				{
					Name: 2,
					CIDR: "10.250.0.0/19",
				},
				{
					Name: 3,
					CIDR: "10.250.32.0/19",
				},
			},
		},
		"Azure three zones and Default 10.250.0.0/16": {
			givenNodesCidr: "10.250.0.0/16",
			givenZoneNames: []string{"1", "2", "3"},
			expectedAzureZones: []Zone{
				{
					Name: 1,
					CIDR: "10.250.0.0/19",
				},
				{
					Name: 2,
					CIDR: "10.250.32.0/19",
				},
				{
					Name: 3,
					CIDR: "10.250.64.0/19",
				},
			},
		},
		"Azure three zones reverse order and Default 10.250.0.0/16": {
			givenNodesCidr: "10.250.0.0/16",
			givenZoneNames: []string{"3", "2", "1"},
			expectedAzureZones: []Zone{
				{
					Name: 3,
					CIDR: "10.250.0.0/19",
				},
				{
					Name: 2,
					CIDR: "10.250.32.0/19",
				},
				{
					Name: 1,
					CIDR: "10.250.64.0/19",
				},
			},
		},
		"Azure four zones and Default 10.250.0.0/16": {
			givenNodesCidr: "10.250.0.0/16",
			givenZoneNames: []string{"1", "2", "3", "4"},
			expectedAzureZones: []Zone{
				{
					Name: 1,
					CIDR: "10.250.0.0/19",
				},
				{
					Name: 2,
					CIDR: "10.250.32.0/19",
				},
				{
					Name: 3,
					CIDR: "10.250.64.0/19",
				},
				{
					Name: 4,
					CIDR: "10.250.96.0/19",
				},
			},
		},
		"Azure five zones and Default 10.250.0.0/16": {
			givenNodesCidr: "10.250.0.0/16",
			givenZoneNames: []string{"1", "2", "3", "4", "5"},
			expectedAzureZones: []Zone{
				{
					Name: 1,
					CIDR: "10.250.0.0/19",
				},
				{
					Name: 2,
					CIDR: "10.250.32.0/19",
				},
				{
					Name: 3,
					CIDR: "10.250.64.0/19",
				},
				{
					Name: 4,
					CIDR: "10.250.96.0/19",
				},
				{
					Name: 5,
					CIDR: "10.250.128.0/19",
				},
			},
		},
		"Azure six zones and Default 10.250.0.0/16": {
			givenNodesCidr: "10.250.0.0/16",
			givenZoneNames: []string{"1", "2", "3", "4", "5", "6"},
			expectedAzureZones: []Zone{
				{
					Name: 1,
					CIDR: "10.250.0.0/19",
				},
				{
					Name: 2,
					CIDR: "10.250.32.0/19",
				},
				{
					Name: 3,
					CIDR: "10.250.64.0/19",
				},
				{
					Name: 4,
					CIDR: "10.250.96.0/19",
				},
				{
					Name: 5,
					CIDR: "10.250.128.0/19",
				},
				{
					Name: 6,
					CIDR: "10.250.160.0/19",
				},
			},
		},
		"Azure seven zones and Default 10.250.0.0/16": {
			givenNodesCidr: "10.250.0.0/16",
			givenZoneNames: []string{"1", "2", "3", "4", "5", "6", "7"},
			expectedAzureZones: []Zone{
				{
					Name: 1,
					CIDR: "10.250.0.0/19",
				},
				{
					Name: 2,
					CIDR: "10.250.32.0/19",
				},
				{
					Name: 3,
					CIDR: "10.250.64.0/19",
				},
				{
					Name: 4,
					CIDR: "10.250.96.0/19",
				},
				{
					Name: 5,
					CIDR: "10.250.128.0/19",
				},
				{
					Name: 6,
					CIDR: "10.250.160.0/19",
				},
				{
					Name: 7,
					CIDR: "10.250.192.0/19",
				},
			},
		},
		"Azure eight zones and Default 10.250.0.0/16": {
			givenNodesCidr: "10.250.0.0/16",
			givenZoneNames: []string{"1", "2", "3", "4", "5", "6", "7", "8"},
			expectedAzureZones: []Zone{
				{
					Name: 1,
					CIDR: "10.250.0.0/19",
				},
				{
					Name: 2,
					CIDR: "10.250.32.0/19",
				},
				{
					Name: 3,
					CIDR: "10.250.64.0/19",
				},
				{
					Name: 4,
					CIDR: "10.250.96.0/19",
				},
				{
					Name: 5,
					CIDR: "10.250.128.0/19",
				},
				{
					Name: 6,
					CIDR: "10.250.160.0/19",
				},
				{
					Name: 7,
					CIDR: "10.250.192.0/19",
				},
				{
					Name: 8,
					CIDR: "10.250.224.0/19",
				},
			},
		},
		"Azure three zones and 10.180.0.0/17": {
			givenNodesCidr: "10.180.0.0/17",
			givenZoneNames: []string{"1", "2", "3"},
			expectedAzureZones: []Zone{
				{
					Name: 1,
					CIDR: "10.180.0.0/20",
				},
				{
					Name: 2,
					CIDR: "10.180.16.0/20",
				},
				{
					Name: 3,
					CIDR: "10.180.32.0/20",
				},
			},
		},
		"Azure no zone (azure lite) and 10.250.0.0/22": {
			givenNodesCidr:     "10.250.0.0/22",
			givenZoneNames:     []string{},
			expectedAzureZones: []Zone{},
		},
		"Azure single zone and 10.250.0.0/22": {
			givenNodesCidr: "10.250.0.0/22",
			givenZoneNames: []string{"1"},
			expectedAzureZones: []Zone{
				{
					Name: 1,
					CIDR: "10.250.0.0/25",
				},
			},
		},
		"Azure eight zones and 10.250.0.0/22": {
			givenNodesCidr: "10.250.0.0/22",
			givenZoneNames: []string{"1", "2", "3", "4", "5", "6", "7", "8"},
			expectedAzureZones: []Zone{
				{
					Name: 1,
					CIDR: "10.250.0.0/25",
				},
				{
					Name: 2,
					CIDR: "10.250.0.128/25",
				},
				{
					Name: 3,
					CIDR: "10.250.1.0/25",
				},
				{
					Name: 4,
					CIDR: "10.250.1.128/25",
				},
				{
					Name: 5,
					CIDR: "10.250.2.0/25",
				},
				{
					Name: 6,
					CIDR: "10.250.2.128/25",
				},
				{
					Name: 7,
					CIDR: "10.250.3.0/25",
				},
				{
					Name: 8,
					CIDR: "10.250.3.128/25",
				},
			},
		},
		"Azure eight zones and 10.250.0.0/23": {
			givenNodesCidr: "10.250.0.0/23",
			givenZoneNames: []string{"1", "2", "3", "4", "5", "6", "7", "8"},
			expectedAzureZones: []Zone{
				{
					Name: 1,
					CIDR: "10.250.0.0/26",
				},
				{
					Name: 2,
					CIDR: "10.250.0.64/26",
				},
				{
					Name: 3,
					CIDR: "10.250.0.128/26",
				},
				{
					Name: 4,
					CIDR: "10.250.0.192/26",
				},
				{
					Name: 5,
					CIDR: "10.250.1.0/26",
				},
				{
					Name: 6,
					CIDR: "10.250.1.64/26",
				},
				{
					Name: 7,
					CIDR: "10.250.1.128/26",
				},
				{
					Name: 8,
					CIDR: "10.250.1.192/26",
				},
			},
		},
		"Azure eight zones and 10.250.0.0/24": {
			givenNodesCidr: "10.250.0.0/24",
			givenZoneNames: []string{"1", "2", "3", "4", "5", "6", "7", "8"},
			expectedAzureZones: []Zone{
				{
					Name: 1,
					CIDR: "10.250.0.0/27",
				},
				{
					Name: 2,
					CIDR: "10.250.0.32/27",
				},
				{
					Name: 3,
					CIDR: "10.250.0.64/27",
				},
				{
					Name: 4,
					CIDR: "10.250.0.96/27",
				},
				{
					Name: 5,
					CIDR: "10.250.0.128/27",
				},
				{
					Name: 6,
					CIDR: "10.250.0.160/27",
				},
				{
					Name: 7,
					CIDR: "10.250.0.192/27",
				},
				{
					Name: 8,
					CIDR: "10.250.0.224/27",
				},
			},
		},
	} {
		t.Run(tname, func(t *testing.T) {
			zones, err := generateAzureZones(tcase.givenNodesCidr, tcase.givenZoneNames)

			require.NoError(t, err)
			assert.Equal(t, len(tcase.expectedAzureZones), len(zones))

			for i, expectedZone := range tcase.expectedAzureZones {
				assertAzureZone(t, tcase.givenNodesCidr, expectedZone, zones[i])
			}
		})
	}

	// error cases

	for tname, tcase := range map[string]struct {
		givenNodesCidr string
		givenZoneNames []string
		message        string
	}{
		"Azure should return error when more than 8 zones provided": {
			givenNodesCidr: "10.250.0.0/16",
			givenZoneNames: []string{
				"1",
				"2",
				"3",
				"4",
				"5",
				"6",
				"7",
				"8",
				"9",
			},
		},
		"Azure should return error when when prefix is too big for ex 10.250.0.0/25": {
			givenNodesCidr: "10.250.0.0/25",
			message:        "CIDR prefix length must be between 16 and 24",
			givenZoneNames: []string{
				"1",
			},
		},
		"Azure should return error when when prefix is too small for ex 10.250.0.0/15": {
			givenNodesCidr: "10.250.0.0/15",
			message:        "CIDR prefix length must be between 16 and 24",
			givenZoneNames: []string{
				"1",
			},
		},
		"Azure should return error when cannot parse nodes CIDR": {
			givenNodesCidr: "888.888.888.0/77",
			givenZoneNames: []string{
				"eu-central-1a",
			},
			message: "failed to parse worker network CIDR",
		},
	} {
		t.Run(tname, func(t *testing.T) {
			zones, err := generateAzureZones(tcase.givenNodesCidr, tcase.givenZoneNames)
			assert.Contains(t, err.Error(), tcase.message)
			assert.Error(t, err)
			assert.Equal(t, 0, len(zones))
		})
	}
}

func assertAzureZone(t *testing.T, nodesCIDR string, expected Zone, verified Zone) {
	result, err := networking.IsSubnetInsideWorkerCIDR(nodesCIDR, expected.CIDR)
	assert.NoError(t, err)
	assert.True(t, result)

	assert.Equal(t, expected.CIDR, verified.CIDR)
	assert.Equal(t, expected.Name, verified.Name)
	require.NotNil(t, verified.NatGateway)
	assert.Equal(t, true, verified.NatGateway.Enabled)
	assert.Equal(t, defaultConnectionTimeOutMinutes, verified.NatGateway.IdleConnectionTimeoutMinutes)
}
