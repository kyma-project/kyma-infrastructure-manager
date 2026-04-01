package alicloud

import (
	alicloud "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud/v1alpha1"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/hyperscaler/networking"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestZonesWithCustomNodeIPRange(t *testing.T) {

	for tname, tcase := range map[string]struct {
		givenNodesCidr string
		givenZoneNames []string
		expectedZones  []alicloud.Zone
	}{
		"one zone and Default 10.250.0.0/16": {
			givenNodesCidr: "10.250.0.0/16",
			givenZoneNames: []string{"eu-central-1a"},
			expectedZones: []alicloud.Zone{
				{
					Name:    "eu-central-1a",
					Workers: "10.250.0.0/19",
				},
			},
		},
		"two zones and Default 10.250.0.0/16": {
			givenNodesCidr: "10.250.0.0/16",
			givenZoneNames: []string{"eu-central-1b", "eu-central-1c"},
			expectedZones: []alicloud.Zone{
				{
					Name:    "eu-central-1b",
					Workers: "10.250.0.0/19",
				},
				{
					Name:    "eu-central-1c",
					Workers: "10.250.32.0/19",
				},
			},
		},
		"three zones and Default 10.250.0.0/16": {
			givenNodesCidr: "10.250.0.0/16",
			givenZoneNames: []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"},
			expectedZones: []alicloud.Zone{
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
		"three zones reverse order and Default 10.250.0.0/16": {
			givenNodesCidr: "10.250.0.0/16",
			givenZoneNames: []string{"eu-central-1c", "eu-central-1b", "eu-central-1a"},
			expectedZones: []alicloud.Zone{
				{
					Name:    "eu-central-1c",
					Workers: "10.250.0.0/19",
				},
				{
					Name:    "eu-central-1b",
					Workers: "10.250.32.0/19",
				},
				{
					Name:    "eu-central-1a",
					Workers: "10.250.64.0/19",
				},
			},
		},
		"four zones and Default 10.250.0.0/16": {
			givenNodesCidr: "10.250.0.0/16",
			givenZoneNames: []string{"eu-central-1a", "eu-central-1b", "eu-central-1c", "eu-central-1d"},
			expectedZones: []alicloud.Zone{
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
				{
					Name:    "eu-central-1d",
					Workers: "10.250.96.0/19",
				},
			},
		},
		"five zones and Default 10.250.0.0/16": {
			givenNodesCidr: "10.250.0.0/16",
			givenZoneNames: []string{"eu-central-1a", "eu-central-1b", "eu-central-1c", "eu-central-1d", "eu-central-1e"},
			expectedZones: []alicloud.Zone{
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
				{
					Name:    "eu-central-1d",
					Workers: "10.250.96.0/19",
				},
				{
					Name:    "eu-central-1e",
					Workers: "10.250.128.0/19",
				},
			},
		},
		"six zones and Default 10.250.0.0/16": {
			givenNodesCidr: "10.250.0.0/16",
			givenZoneNames: []string{"eu-central-1a", "eu-central-1b", "eu-central-1c", "eu-central-1d", "eu-central-1e", "eu-central-1f"},
			expectedZones: []alicloud.Zone{
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
				{
					Name:    "eu-central-1d",
					Workers: "10.250.96.0/19",
				},
				{
					Name:    "eu-central-1e",
					Workers: "10.250.128.0/19",
				},
				{
					Name:    "eu-central-1f",
					Workers: "10.250.160.0/19",
				},
			},
		},
		"seven zones and Default 10.250.0.0/16": {
			givenNodesCidr: "10.250.0.0/16",
			givenZoneNames: []string{"eu-central-1a", "eu-central-1b", "eu-central-1c", "eu-central-1d", "eu-central-1e", "eu-central-1f", "eu-central-1g"},
			expectedZones: []alicloud.Zone{
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
				{
					Name:    "eu-central-1d",
					Workers: "10.250.96.0/19",
				},
				{
					Name:    "eu-central-1e",
					Workers: "10.250.128.0/19",
				},
				{
					Name:    "eu-central-1f",
					Workers: "10.250.160.0/19",
				},
				{
					Name:    "eu-central-1g",
					Workers: "10.250.192.0/19",
				},
			},
		},
		"eight zones and Default 10.250.0.0/16": {
			givenNodesCidr: "10.250.0.0/16",
			givenZoneNames: []string{"eu-central-1a", "eu-central-1b", "eu-central-1c", "eu-central-1d", "eu-central-1e", "eu-central-1f", "eu-central-1g", "eu-central-1h"},
			expectedZones: []alicloud.Zone{
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
				{
					Name:    "eu-central-1d",
					Workers: "10.250.96.0/19",
				},
				{
					Name:    "eu-central-1e",
					Workers: "10.250.128.0/19",
				},
				{
					Name:    "eu-central-1f",
					Workers: "10.250.160.0/19",
				},
				{
					Name:    "eu-central-1g",
					Workers: "10.250.192.0/19",
				},
				{
					Name:    "eu-central-1h",
					Workers: "10.250.224.0/19",
				},
			},
		},
		"three zones and 10.180.0.0/17": {
			givenNodesCidr: "10.180.0.0/17",
			givenZoneNames: []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"},
			expectedZones: []alicloud.Zone{
				{
					Name:    "eu-central-1a",
					Workers: "10.180.0.0/20",
				},
				{
					Name:    "eu-central-1b",
					Workers: "10.180.16.0/20",
				},
				{
					Name:    "eu-central-1c",
					Workers: "10.180.32.0/20",
				},
			},
		},
		"single zone and 10.250.0.0/22": {
			givenNodesCidr: "10.250.0.0/22",
			givenZoneNames: []string{"eu-central-1a"},
			expectedZones: []alicloud.Zone{
				{
					Name:    "eu-central-1a",
					Workers: "10.250.0.0/25",
				},
			},
		},
		"eight zones and 10.250.0.0/22": {
			givenNodesCidr: "10.250.0.0/22",
			givenZoneNames: []string{"eu-central-1a", "eu-central-1b", "eu-central-1c", "eu-central-1d", "eu-central-1e", "eu-central-1f", "eu-central-1g", "eu-central-1h"},
			expectedZones: []alicloud.Zone{
				{
					Name:    "eu-central-1a",
					Workers: "10.250.0.0/25",
				},
				{
					Name:    "eu-central-1b",
					Workers: "10.250.0.128/25",
				},
				{
					Name:    "eu-central-1c",
					Workers: "10.250.1.0/25",
				},
				{
					Name:    "eu-central-1d",
					Workers: "10.250.1.128/25",
				},
				{
					Name:    "eu-central-1e",
					Workers: "10.250.2.0/25",
				},
				{
					Name:    "eu-central-1f",
					Workers: "10.250.2.128/25",
				},
				{
					Name:    "eu-central-1g",
					Workers: "10.250.3.0/25",
				},
				{
					Name:    "eu-central-1h",
					Workers: "10.250.3.128/25",
				},
			},
		},
		"eight zones and 10.250.0.0/23": {
			givenNodesCidr: "10.250.0.0/23",
			givenZoneNames: []string{"eu-central-1a", "eu-central-1b", "eu-central-1c", "eu-central-1d", "eu-central-1e", "eu-central-1f", "eu-central-1g", "eu-central-1h"},
			expectedZones: []alicloud.Zone{
				{
					Name:    "eu-central-1a",
					Workers: "10.250.0.0/26",
				},
				{
					Name:    "eu-central-1b",
					Workers: "10.250.0.64/26",
				},
				{
					Name:    "eu-central-1c",
					Workers: "10.250.0.128/26",
				},
				{
					Name:    "eu-central-1d",
					Workers: "10.250.0.192/26",
				},
				{
					Name:    "eu-central-1e",
					Workers: "10.250.1.0/26",
				},
				{
					Name:    "eu-central-1f",
					Workers: "10.250.1.64/26",
				},
				{
					Name:    "eu-central-1g",
					Workers: "10.250.1.128/26",
				},
				{
					Name:    "eu-central-1h",
					Workers: "10.250.1.192/26",
				},
			},
		},
		"eight zones and 10.250.0.0/24": {
			givenNodesCidr: "10.250.0.0/24",
			givenZoneNames: []string{"eu-central-1a", "eu-central-1b", "eu-central-1c", "eu-central-1d", "eu-central-1e", "eu-central-1f", "eu-central-1g", "eu-central-1h"},
			expectedZones: []alicloud.Zone{
				{
					Name:    "eu-central-1a",
					Workers: "10.250.0.0/27",
				},
				{
					Name:    "eu-central-1b",
					Workers: "10.250.0.32/27",
				},
				{
					Name:    "eu-central-1c",
					Workers: "10.250.0.64/27",
				},
				{
					Name:    "eu-central-1d",
					Workers: "10.250.0.96/27",
				},
				{
					Name:    "eu-central-1e",
					Workers: "10.250.0.128/27",
				},
				{
					Name:    "eu-central-1f",
					Workers: "10.250.0.160/27",
				},
				{
					Name:    "eu-central-1g",
					Workers: "10.250.0.192/27",
				},
				{
					Name:    "eu-central-1h",
					Workers: "10.250.0.224/27",
				},
			},
		},
	} {
		t.Run(tname, func(t *testing.T) {
			zones, err := generateZones(tcase.givenNodesCidr, tcase.givenZoneNames)

			require.NoError(t, err)
			assert.Equal(t, len(tcase.expectedZones), len(zones))

			for i, expectedZone := range tcase.expectedZones {
				assertZone(t, tcase.givenNodesCidr, expectedZone, zones[i])
			}
		})
	}

	// error cases

	for tname, tcase := range map[string]struct {
		givenNodesCidr string
		givenZoneNames []string
		message        string
	}{
		"should return error when more than 8 zones provided": {
			givenNodesCidr: "10.250.0.0/16",
			givenZoneNames: []string{
				"eu-central-1a",
				"eu-central-1b",
				"eu-central-1c",
				"eu-central-1d",
				"eu-central-1e",
				"eu-central-1f",
				"eu-central-1g",
				"eu-central-1h",
				"eu-central-1i",
			},
		},
		"should return error when when prefix is too big for ex 10.250.0.0/25": {
			givenNodesCidr: "10.250.0.0/25",
			message:        "CIDR prefix length must be less than or equal to 24",
			givenZoneNames: []string{
				"eu-central-1a",
			},
		},
		"should return error when when prefix is too small for ex 10.250.0.0/15": {
			givenNodesCidr: "10.250.0.0/15",
			message:        "CIDR prefix length must be bigger than or equal to 16",
			givenZoneNames: []string{
				"eu-central-1a",
			},
		},
		"should return error when cannot parse nodes CIDR": {
			givenNodesCidr: "888.888.888.0/77",
			givenZoneNames: []string{
				"eu-central-1a",
			},
			message: "failed to parse worker network CIDR",
		},
	} {
		t.Run(tname, func(t *testing.T) {
			zones, err := generateZones(tcase.givenNodesCidr, tcase.givenZoneNames)
			assert.Contains(t, err.Error(), tcase.message)
			assert.Error(t, err)
			assert.Equal(t, 0, len(zones))
		})
	}
}

func assertZone(t *testing.T, nodesCIDR string, expected alicloud.Zone, verified alicloud.Zone) {
	result, err := networking.IsSubnetInsideWorkerCIDR(nodesCIDR, expected.Workers)
	assert.NoError(t, err)
	assert.True(t, result)

	assert.Equal(t, expected.Workers, verified.Workers)
	assert.Equal(t, expected.Name, verified.Name)
}
