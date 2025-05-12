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
			zones := generateAWSZones(tcase.givenNodesCidr, tcase.givenZoneNames)

			assert.Equal(t, len(tcase.expectedAwsZones), len(zones))

			for i, expectedZone := range tcase.expectedAwsZones {
				assertAWSIpRanges(t, expectedZone, zones[i])
			}
		})
	}
}

func assertAWSIpRanges(t *testing.T, zone v1alpha1.Zone, input v1alpha1.Zone) {
	assert.Equal(t, zone.Internal, input.Internal)
	assert.Equal(t, zone.Workers, input.Workers)
	assert.Equal(t, zone.Public, input.Public)
	assert.Equal(t, zone.Name, input.Name)
}
