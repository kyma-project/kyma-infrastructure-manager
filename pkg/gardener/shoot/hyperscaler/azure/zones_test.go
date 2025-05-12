package azure

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestAzureZonesWithCustomNodeIPRange(t *testing.T) {

	for tname, tcase := range map[string]struct {
		givenNodesCidr     string
		givenZoneNames     []string
		expectedAzureZones []Zone
	}{
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
	} {
		t.Run(tname, func(t *testing.T) {
			zones := generateAzureZones(tcase.givenNodesCidr, tcase.givenZoneNames)

			assert.Equal(t, len(tcase.expectedAzureZones), len(zones))

			for i, expectedZone := range tcase.expectedAzureZones {
				assertAzureZone(t, expectedZone, zones[i])
			}
		})
	}
}

func assertAzureZone(t *testing.T, zone Zone, input Zone) {
	assert.Equal(t, zone.CIDR, input.CIDR)
	assert.Equal(t, zone.Name, input.Name)
}
