package aws

import (
	"github.com/gardener/gardener-extension-provider-aws/pkg/apis/aws/v1alpha1"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestAWSZonesWithCustomNodeIPRange(t *testing.T) {

	for tname, tcase := range map[string]struct {
		givenNodesCidr   string
		givenZoneNames   []string
		expectedAwsZones []v1alpha1.Zone
	}{
		"AWS three zones and 10.250.0.0/16": {
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
		"AWS three zones 10.180.0.0/23": {
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
			zones, err := generateAWSZones(tcase.givenNodesCidr, tcase.givenZoneNames)

			assert.NoError(t, err)
			assert.Equal(t, len(tcase.expectedAwsZones), len(zones))

			for i, expectedZone := range tcase.expectedAwsZones {
				assertAWSIpRanges(t, expectedZone, zones[i])
			}
		})
	}

	// error cases

	for tname, tcase := range map[string]struct {
		givenNodesCidr string
		givenZoneNames []string
	}{
		"AWS should return error when more than 8 zones provided": {
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
		"AWS should return error when no zones are provided": {
			givenNodesCidr: "10.180.0.0/23",
			givenZoneNames: []string{},
		},
	} {
		t.Run(tname, func(t *testing.T) {
			zones, err := generateAWSZones(tcase.givenNodesCidr, tcase.givenZoneNames)

			assert.Error(t, err)
			assert.Equal(t, 0, len(zones))
		})
	}
}

func assertAWSIpRanges(t *testing.T, zone v1alpha1.Zone, input v1alpha1.Zone) {
	assert.Equal(t, zone.Internal, input.Internal)
	assert.Equal(t, zone.Workers, input.Workers)
	assert.Equal(t, zone.Public, input.Public)
	assert.Equal(t, zone.Name, input.Name)
}
