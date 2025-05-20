package aws

import (
	"github.com/gardener/gardener-extension-provider-aws/pkg/apis/aws/v1alpha1"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/hyperscaler/networking"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestAWSZonesWithCustomNodeIPRange(t *testing.T) {
	for tname, tcase := range map[string]struct {
		givenNodesCidr   string
		givenZoneNames   []string
		expectedAwsZones []v1alpha1.Zone
	}{
		"AWS one zone and 10.250.0.0/16": {
			givenNodesCidr: "10.250.0.0/16",
			givenZoneNames: []string{
				"eu-central-1a",
			},
			expectedAwsZones: []v1alpha1.Zone{
				{
					Name:     "eu-central-1a",
					Workers:  "10.250.0.0/19",
					Public:   "10.250.32.0/20",
					Internal: "10.250.48.0/20",
				},
			},
		},
		"AWS two zones and 10.250.0.0/16": {
			givenNodesCidr: "10.250.0.0/16",
			givenZoneNames: []string{
				"eu-central-1a",
				"eu-central-1b",
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
			},
		},
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
		"AWS four zones and 10.250.0.0/16": {
			givenNodesCidr: "10.250.0.0/16",
			givenZoneNames: []string{
				"eu-central-1a",
				"eu-central-1b",
				"eu-central-1c",
				"eu-central-1d",
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
				{
					Name:     "eu-central-1d",
					Workers:  "10.250.192.0/22",
					Public:   "10.250.196.0/22",
					Internal: "10.250.200.0/22",
				},
			},
		},
		"AWS five zones and 10.250.0.0/16": {
			givenNodesCidr: "10.250.0.0/16",
			givenZoneNames: []string{
				"eu-central-1a",
				"eu-central-1b",
				"eu-central-1c",
				"eu-central-1d",
				"eu-central-1e",
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
				{
					Name:     "eu-central-1d",
					Workers:  "10.250.192.0/22",
					Public:   "10.250.196.0/22",
					Internal: "10.250.200.0/22",
				},
				{
					Name:     "eu-central-1e",
					Workers:  "10.250.204.0/22",
					Public:   "10.250.208.0/22",
					Internal: "10.250.212.0/22",
				},
			},
		},
		"AWS six zones and 10.250.0.0/16": {
			givenNodesCidr: "10.250.0.0/16",
			givenZoneNames: []string{
				"eu-central-1a",
				"eu-central-1b",
				"eu-central-1c",
				"eu-central-1d",
				"eu-central-1e",
				"eu-central-1f",
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
				{
					Name:     "eu-central-1d",
					Workers:  "10.250.192.0/22",
					Public:   "10.250.196.0/22",
					Internal: "10.250.200.0/22",
				},
				{
					Name:     "eu-central-1e",
					Workers:  "10.250.204.0/22",
					Public:   "10.250.208.0/22",
					Internal: "10.250.212.0/22",
				},
				{
					Name:     "eu-central-1f",
					Workers:  "10.250.216.0/22",
					Public:   "10.250.220.0/22",
					Internal: "10.250.224.0/22",
				},
			},
		},
		"AWS seven zones and 10.250.0.0/16": {
			givenNodesCidr: "10.250.0.0/16",
			givenZoneNames: []string{
				"eu-central-1a",
				"eu-central-1b",
				"eu-central-1c",
				"eu-central-1d",
				"eu-central-1e",
				"eu-central-1f",
				"eu-central-1g",
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
				{
					Name:     "eu-central-1d",
					Workers:  "10.250.192.0/22",
					Public:   "10.250.196.0/22",
					Internal: "10.250.200.0/22",
				},
				{
					Name:     "eu-central-1e",
					Workers:  "10.250.204.0/22",
					Public:   "10.250.208.0/22",
					Internal: "10.250.212.0/22",
				},
				{
					Name:     "eu-central-1f",
					Workers:  "10.250.216.0/22",
					Public:   "10.250.220.0/22",
					Internal: "10.250.224.0/22",
				},
				{
					Name:     "eu-central-1g",
					Workers:  "10.250.228.0/22",
					Public:   "10.250.232.0/22",
					Internal: "10.250.236.0/22",
				},
			},
		},
		"AWS eight zones and 10.250.0.0/16": {
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
				{
					Name:     "eu-central-1d",
					Workers:  "10.250.192.0/22",
					Public:   "10.250.196.0/22",
					Internal: "10.250.200.0/22",
				},
				{
					Name:     "eu-central-1e",
					Workers:  "10.250.204.0/22",
					Public:   "10.250.208.0/22",
					Internal: "10.250.212.0/22",
				},
				{
					Name:     "eu-central-1f",
					Workers:  "10.250.216.0/22",
					Public:   "10.250.220.0/22",
					Internal: "10.250.224.0/22",
				},
				{
					Name:     "eu-central-1g",
					Workers:  "10.250.228.0/22",
					Public:   "10.250.232.0/22",
					Internal: "10.250.236.0/22",
				},
				{
					Name:     "eu-central-1h",
					Workers:  "10.250.240.0/22",
					Public:   "10.250.244.0/22",
					Internal: "10.250.248.0/22",
				},
			},
		},
		"AWS one zone and 10.250.0.0/22": {
			givenNodesCidr: "10.250.0.0/22",
			givenZoneNames: []string{
				"eu-central-1a",
			},
			expectedAwsZones: []v1alpha1.Zone{
				{
					Name:     "eu-central-1a",
					Workers:  "10.250.0.0/25",
					Public:   "10.250.0.128/26",
					Internal: "10.250.0.192/26",
				},
			},
		},
		"AWS two zones and 10.250.0.0/22": {
			givenNodesCidr: "10.250.0.0/22",
			givenZoneNames: []string{
				"eu-central-1a",
				"eu-central-1b",
			},
			expectedAwsZones: []v1alpha1.Zone{
				{
					Name:     "eu-central-1a",
					Workers:  "10.250.0.0/25",
					Public:   "10.250.0.128/26",
					Internal: "10.250.0.192/26",
				},
				{
					Name:     "eu-central-1b",
					Workers:  "10.250.1.0/25",
					Public:   "10.250.1.128/26",
					Internal: "10.250.1.192/26",
				},
			},
		},
		"AWS three zones and 10.250.0.0/22": {
			givenNodesCidr: "10.250.0.0/22",
			givenZoneNames: []string{
				"eu-central-1a",
				"eu-central-1b",
				"eu-central-1c",
			},
			expectedAwsZones: []v1alpha1.Zone{
				{
					Name:     "eu-central-1a",
					Workers:  "10.250.0.0/25",
					Public:   "10.250.0.128/26",
					Internal: "10.250.0.192/26",
				},
				{
					Name:     "eu-central-1b",
					Workers:  "10.250.1.0/25",
					Public:   "10.250.1.128/26",
					Internal: "10.250.1.192/26",
				},
				{
					Name:     "eu-central-1c",
					Workers:  "10.250.2.0/25",
					Public:   "10.250.2.128/26",
					Internal: "10.250.2.192/26",
				},
			},
		},
		"AWS four zones and 10.250.0.0/22": {
			givenNodesCidr: "10.250.0.0/22",
			givenZoneNames: []string{
				"eu-central-1a",
				"eu-central-1b",
				"eu-central-1c",
				"eu-central-1d",
			},
			expectedAwsZones: []v1alpha1.Zone{
				{
					Name:     "eu-central-1a",
					Workers:  "10.250.0.0/25",
					Public:   "10.250.0.128/26",
					Internal: "10.250.0.192/26",
				},
				{
					Name:     "eu-central-1b",
					Workers:  "10.250.1.0/25",
					Public:   "10.250.1.128/26",
					Internal: "10.250.1.192/26",
				},
				{
					Name:     "eu-central-1c",
					Workers:  "10.250.2.0/25",
					Public:   "10.250.2.128/26",
					Internal: "10.250.2.192/26",
				},
				{
					Name:     "eu-central-1d",
					Workers:  "10.250.3.0/28",
					Public:   "10.250.3.16/28",
					Internal: "10.250.3.32/28",
				},
			},
		},
		"AWS five zones and 10.250.0.0/22": {
			givenNodesCidr: "10.250.0.0/22",
			givenZoneNames: []string{
				"eu-central-1a",
				"eu-central-1b",
				"eu-central-1c",
				"eu-central-1d",
				"eu-central-1e",
			},
			expectedAwsZones: []v1alpha1.Zone{
				{
					Name:     "eu-central-1a",
					Workers:  "10.250.0.0/25",
					Public:   "10.250.0.128/26",
					Internal: "10.250.0.192/26",
				},
				{
					Name:     "eu-central-1b",
					Workers:  "10.250.1.0/25",
					Public:   "10.250.1.128/26",
					Internal: "10.250.1.192/26",
				},
				{
					Name:     "eu-central-1c",
					Workers:  "10.250.2.0/25",
					Public:   "10.250.2.128/26",
					Internal: "10.250.2.192/26",
				},
				{
					Name:     "eu-central-1d",
					Workers:  "10.250.3.0/28",
					Public:   "10.250.3.16/28",
					Internal: "10.250.3.32/28",
				},
				{
					Name:     "eu-central-1e",
					Workers:  "10.250.3.48/28",
					Public:   "10.250.3.64/28",
					Internal: "10.250.3.80/28",
				},
			},
		},
		"AWS six zones and 10.250.0.0/22": {
			givenNodesCidr: "10.250.0.0/22",
			givenZoneNames: []string{
				"eu-central-1a",
				"eu-central-1b",
				"eu-central-1c",
				"eu-central-1d",
				"eu-central-1e",
				"eu-central-1f",
			},
			expectedAwsZones: []v1alpha1.Zone{
				{
					Name:     "eu-central-1a",
					Workers:  "10.250.0.0/25",
					Public:   "10.250.0.128/26",
					Internal: "10.250.0.192/26",
				},
				{
					Name:     "eu-central-1b",
					Workers:  "10.250.1.0/25",
					Public:   "10.250.1.128/26",
					Internal: "10.250.1.192/26",
				},
				{
					Name:     "eu-central-1c",
					Workers:  "10.250.2.0/25",
					Public:   "10.250.2.128/26",
					Internal: "10.250.2.192/26",
				},
				{
					Name:     "eu-central-1d",
					Workers:  "10.250.3.0/28",
					Public:   "10.250.3.16/28",
					Internal: "10.250.3.32/28",
				},
				{
					Name:     "eu-central-1e",
					Workers:  "10.250.3.48/28",
					Public:   "10.250.3.64/28",
					Internal: "10.250.3.80/28",
				},
				{
					Name:     "eu-central-1f",
					Workers:  "10.250.3.96/28",
					Public:   "10.250.3.112/28",
					Internal: "10.250.3.128/28",
				},
			},
		},
		"AWS seven zones and 10.250.0.0/22": {
			givenNodesCidr: "10.250.0.0/22",
			givenZoneNames: []string{
				"eu-central-1a",
				"eu-central-1b",
				"eu-central-1c",
				"eu-central-1d",
				"eu-central-1e",
				"eu-central-1f",
				"eu-central-1g",
			},
			expectedAwsZones: []v1alpha1.Zone{
				{
					Name:     "eu-central-1a",
					Workers:  "10.250.0.0/25",
					Public:   "10.250.0.128/26",
					Internal: "10.250.0.192/26",
				},
				{
					Name:     "eu-central-1b",
					Workers:  "10.250.1.0/25",
					Public:   "10.250.1.128/26",
					Internal: "10.250.1.192/26",
				},
				{
					Name:     "eu-central-1c",
					Workers:  "10.250.2.0/25",
					Public:   "10.250.2.128/26",
					Internal: "10.250.2.192/26",
				},
				{
					Name:     "eu-central-1d",
					Workers:  "10.250.3.0/28",
					Public:   "10.250.3.16/28",
					Internal: "10.250.3.32/28",
				},
				{
					Name:     "eu-central-1e",
					Workers:  "10.250.3.48/28",
					Public:   "10.250.3.64/28",
					Internal: "10.250.3.80/28",
				},
				{
					Name:     "eu-central-1f",
					Workers:  "10.250.3.96/28",
					Public:   "10.250.3.112/28",
					Internal: "10.250.3.128/28",
				},
				{
					Name:     "eu-central-1g",
					Workers:  "10.250.3.144/28",
					Public:   "10.250.3.160/28",
					Internal: "10.250.3.176/28",
				},
			},
		},
		"AWS eight zones and 10.250.0.0/22": {
			givenNodesCidr: "10.250.0.0/22",
			givenZoneNames: []string{
				"eu-central-1a",
				"eu-central-1b",
				"eu-central-1c",
				"eu-central-1d",
				"eu-central-1e",
				"eu-central-1f",
				"eu-central-1g",
				"eu-central-1h",
			},
			expectedAwsZones: []v1alpha1.Zone{
				{
					Name:     "eu-central-1a",
					Workers:  "10.250.0.0/25",
					Public:   "10.250.0.128/26",
					Internal: "10.250.0.192/26",
				},
				{
					Name:     "eu-central-1b",
					Workers:  "10.250.1.0/25",
					Public:   "10.250.1.128/26",
					Internal: "10.250.1.192/26",
				},
				{
					Name:     "eu-central-1c",
					Workers:  "10.250.2.0/25",
					Public:   "10.250.2.128/26",
					Internal: "10.250.2.192/26",
				},
				{
					Name:     "eu-central-1d",
					Workers:  "10.250.3.0/28",
					Public:   "10.250.3.16/28",
					Internal: "10.250.3.32/28",
				},
				{
					Name:     "eu-central-1e",
					Workers:  "10.250.3.48/28",
					Public:   "10.250.3.64/28",
					Internal: "10.250.3.80/28",
				},
				{
					Name:     "eu-central-1f",
					Workers:  "10.250.3.96/28",
					Public:   "10.250.3.112/28",
					Internal: "10.250.3.128/28",
				},
				{
					Name:     "eu-central-1g",
					Workers:  "10.250.3.144/28",
					Public:   "10.250.3.160/28",
					Internal: "10.250.3.176/28",
				},
				{
					Name:     "eu-central-1h",
					Workers:  "10.250.3.192/28",
					Public:   "10.250.3.208/28",
					Internal: "10.250.3.224/28",
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
		"AWS one zone and 10.250.0.0/23": {
			givenNodesCidr: "10.250.0.0/23",
			givenZoneNames: []string{
				"eu-central-1a",
			},
			expectedAwsZones: []v1alpha1.Zone{
				{
					Name:     "eu-central-1a",
					Workers:  "10.250.0.0/26",
					Public:   "10.250.0.64/27",
					Internal: "10.250.0.96/27",
				},
			},
		},
		"AWS two zones and 10.250.0.0/23": {
			givenNodesCidr: "10.250.0.0/23",
			givenZoneNames: []string{
				"eu-central-1a",
				"eu-central-1b",
			},
			expectedAwsZones: []v1alpha1.Zone{
				{
					Name:     "eu-central-1a",
					Workers:  "10.250.0.0/26",
					Public:   "10.250.0.64/27",
					Internal: "10.250.0.96/27",
				},
				{
					Name:     "eu-central-1b",
					Workers:  "10.250.0.128/26",
					Public:   "10.250.0.192/27",
					Internal: "10.250.0.224/27",
				},
			},
		},
		"AWS three zones and 10.250.0.0/23": {
			givenNodesCidr: "10.250.0.0/23",
			givenZoneNames: []string{
				"eu-central-1a",
				"eu-central-1b",
				"eu-central-1c",
			},
			expectedAwsZones: []v1alpha1.Zone{
				{
					Name:     "eu-central-1a",
					Workers:  "10.250.0.0/26",
					Public:   "10.250.0.64/27",
					Internal: "10.250.0.96/27",
				},
				{
					Name:     "eu-central-1b",
					Workers:  "10.250.0.128/26",
					Public:   "10.250.0.192/27",
					Internal: "10.250.0.224/27",
				},
				{
					Name:     "eu-central-1c",
					Workers:  "10.250.1.0/26",
					Public:   "10.250.1.64/27",
					Internal: "10.250.1.96/27",
				},
			},
		},
		"AWS four zones and 10.250.0.0/23": {
			givenNodesCidr: "10.250.0.0/23",
			givenZoneNames: []string{
				"eu-central-1a",
				"eu-central-1b",
				"eu-central-1c",
				"eu-central-1d",
			},
			expectedAwsZones: []v1alpha1.Zone{
				{
					Name:     "eu-central-1a",
					Workers:  "10.250.0.0/26",
					Public:   "10.250.0.64/27",
					Internal: "10.250.0.96/27",
				},
				{
					Name:     "eu-central-1b",
					Workers:  "10.250.0.128/26",
					Public:   "10.250.0.192/27",
					Internal: "10.250.0.224/27",
				},
				{
					Name:     "eu-central-1c",
					Workers:  "10.250.1.0/26",
					Public:   "10.250.1.64/27",
					Internal: "10.250.1.96/27",
				},
				{
					Name:     "eu-central-1d",
					Workers:  "10.250.1.128/29",
					Public:   "10.250.1.136/29",
					Internal: "10.250.1.144/29",
				},
			},
		},
		"AWS five zones and 10.250.0.0/23": {
			givenNodesCidr: "10.250.0.0/23",
			givenZoneNames: []string{
				"eu-central-1a",
				"eu-central-1b",
				"eu-central-1c",
				"eu-central-1d",
				"eu-central-1e",
			},
			expectedAwsZones: []v1alpha1.Zone{
				{
					Name:     "eu-central-1a",
					Workers:  "10.250.0.0/26",
					Public:   "10.250.0.64/27",
					Internal: "10.250.0.96/27",
				},
				{
					Name:     "eu-central-1b",
					Workers:  "10.250.0.128/26",
					Public:   "10.250.0.192/27",
					Internal: "10.250.0.224/27",
				},
				{
					Name:     "eu-central-1c",
					Workers:  "10.250.1.0/26",
					Public:   "10.250.1.64/27",
					Internal: "10.250.1.96/27",
				},
				{
					Name:     "eu-central-1d",
					Workers:  "10.250.1.128/29",
					Public:   "10.250.1.136/29",
					Internal: "10.250.1.144/29",
				},
				{
					Name:     "eu-central-1e",
					Workers:  "10.250.1.152/29",
					Public:   "10.250.1.160/29",
					Internal: "10.250.1.168/29",
				},
			},
		},
		"AWS six zones and 10.250.0.0/23": {
			givenNodesCidr: "10.250.0.0/23",
			givenZoneNames: []string{
				"eu-central-1a",
				"eu-central-1b",
				"eu-central-1c",
				"eu-central-1d",
				"eu-central-1e",
				"eu-central-1f",
			},
			expectedAwsZones: []v1alpha1.Zone{
				{
					Name:     "eu-central-1a",
					Workers:  "10.250.0.0/26",
					Public:   "10.250.0.64/27",
					Internal: "10.250.0.96/27",
				},
				{
					Name:     "eu-central-1b",
					Workers:  "10.250.0.128/26",
					Public:   "10.250.0.192/27",
					Internal: "10.250.0.224/27",
				},
				{
					Name:     "eu-central-1c",
					Workers:  "10.250.1.0/26",
					Public:   "10.250.1.64/27",
					Internal: "10.250.1.96/27",
				},
				{
					Name:     "eu-central-1d",
					Workers:  "10.250.1.128/29",
					Public:   "10.250.1.136/29",
					Internal: "10.250.1.144/29",
				},
				{
					Name:     "eu-central-1e",
					Workers:  "10.250.1.152/29",
					Public:   "10.250.1.160/29",
					Internal: "10.250.1.168/29",
				},
				{
					Name:     "eu-central-1f",
					Workers:  "10.250.1.176/29",
					Public:   "10.250.1.184/29",
					Internal: "10.250.1.192/29",
				},
			},
		},
		"AWS seven zones and 10.250.0.0/23": {
			givenNodesCidr: "10.250.0.0/23",
			givenZoneNames: []string{
				"eu-central-1a",
				"eu-central-1b",
				"eu-central-1c",
				"eu-central-1d",
				"eu-central-1e",
				"eu-central-1f",
				"eu-central-1g",
			},
			expectedAwsZones: []v1alpha1.Zone{
				{
					Name:     "eu-central-1a",
					Workers:  "10.250.0.0/26",
					Public:   "10.250.0.64/27",
					Internal: "10.250.0.96/27",
				},
				{
					Name:     "eu-central-1b",
					Workers:  "10.250.0.128/26",
					Public:   "10.250.0.192/27",
					Internal: "10.250.0.224/27",
				},
				{
					Name:     "eu-central-1c",
					Workers:  "10.250.1.0/26",
					Public:   "10.250.1.64/27",
					Internal: "10.250.1.96/27",
				},
				{
					Name:     "eu-central-1d",
					Workers:  "10.250.1.128/29",
					Public:   "10.250.1.136/29",
					Internal: "10.250.1.144/29",
				},
				{
					Name:     "eu-central-1e",
					Workers:  "10.250.1.152/29",
					Public:   "10.250.1.160/29",
					Internal: "10.250.1.168/29",
				},
				{
					Name:     "eu-central-1f",
					Workers:  "10.250.1.176/29",
					Public:   "10.250.1.184/29",
					Internal: "10.250.1.192/29",
				},
				{
					Name:     "eu-central-1g",
					Workers:  "10.250.1.200/29",
					Public:   "10.250.1.208/29",
					Internal: "10.250.1.216/29",
				},
			},
		},
		"AWS eight zones and 10.250.0.0/23": {
			givenNodesCidr: "10.250.0.0/23",
			givenZoneNames: []string{
				"eu-central-1a",
				"eu-central-1b",
				"eu-central-1c",
				"eu-central-1d",
				"eu-central-1e",
				"eu-central-1f",
				"eu-central-1g",
				"eu-central-1h",
			},
			expectedAwsZones: []v1alpha1.Zone{
				{
					Name:     "eu-central-1a",
					Workers:  "10.250.0.0/26",
					Public:   "10.250.0.64/27",
					Internal: "10.250.0.96/27",
				},
				{
					Name:     "eu-central-1b",
					Workers:  "10.250.0.128/26",
					Public:   "10.250.0.192/27",
					Internal: "10.250.0.224/27",
				},
				{
					Name:     "eu-central-1c",
					Workers:  "10.250.1.0/26",
					Public:   "10.250.1.64/27",
					Internal: "10.250.1.96/27",
				},
				{
					Name:     "eu-central-1d",
					Workers:  "10.250.1.128/29",
					Public:   "10.250.1.136/29",
					Internal: "10.250.1.144/29",
				},
				{
					Name:     "eu-central-1e",
					Workers:  "10.250.1.152/29",
					Public:   "10.250.1.160/29",
					Internal: "10.250.1.168/29",
				},
				{
					Name:     "eu-central-1f",
					Workers:  "10.250.1.176/29",
					Public:   "10.250.1.184/29",
					Internal: "10.250.1.192/29",
				},
				{
					Name:     "eu-central-1g",
					Workers:  "10.250.1.200/29",
					Public:   "10.250.1.208/29",
					Internal: "10.250.1.216/29",
				},
				{
					Name:     "eu-central-1h",
					Workers:  "10.250.1.224/29",
					Public:   "10.250.1.232/29",
					Internal: "10.250.1.240/29",
				},
			},
		},
		"AWS one zone and 10.250.0.0/24": {
			givenNodesCidr: "10.250.0.0/24",
			givenZoneNames: []string{
				"eu-central-1a",
			},
			expectedAwsZones: []v1alpha1.Zone{
				{
					Name:     "eu-central-1a",
					Workers:  "10.250.0.0/27",
					Public:   "10.250.0.32/28",
					Internal: "10.250.0.48/28",
				},
			},
		},
		"AWS two zones and 10.250.0.0/24": {
			givenNodesCidr: "10.250.0.0/24",
			givenZoneNames: []string{
				"eu-central-1a",
				"eu-central-1b",
			},
			expectedAwsZones: []v1alpha1.Zone{
				{
					Name:     "eu-central-1a",
					Workers:  "10.250.0.0/27",
					Public:   "10.250.0.32/28",
					Internal: "10.250.0.48/28",
				},
				{
					Name:     "eu-central-1b",
					Workers:  "10.250.0.64/27",
					Public:   "10.250.0.96/28",
					Internal: "10.250.0.112/28",
				},
			},
		},
		"AWS three zones and 10.250.0.0/24": {
			givenNodesCidr: "10.250.0.0/24",
			givenZoneNames: []string{
				"eu-central-1a",
				"eu-central-1b",
				"eu-central-1c",
			},
			expectedAwsZones: []v1alpha1.Zone{
				{
					Name:     "eu-central-1a",
					Workers:  "10.250.0.0/27",
					Public:   "10.250.0.32/28",
					Internal: "10.250.0.48/28",
				},
				{
					Name:     "eu-central-1b",
					Workers:  "10.250.0.64/27",
					Public:   "10.250.0.96/28",
					Internal: "10.250.0.112/28",
				},
				{
					Name:     "eu-central-1c",
					Workers:  "10.250.0.128/27",
					Public:   "10.250.0.160/28",
					Internal: "10.250.0.176/28",
				},
			},
		},
		"AWS four zones and 10.250.0.0/24": {
			givenNodesCidr: "10.250.0.0/24",
			givenZoneNames: []string{
				"eu-central-1a",
				"eu-central-1b",
				"eu-central-1c",
				"eu-central-1d",
			},
			expectedAwsZones: []v1alpha1.Zone{
				{
					Name:     "eu-central-1a",
					Workers:  "10.250.0.0/27",
					Public:   "10.250.0.32/28",
					Internal: "10.250.0.48/28",
				},
				{
					Name:     "eu-central-1b",
					Workers:  "10.250.0.64/27",
					Public:   "10.250.0.96/28",
					Internal: "10.250.0.112/28",
				},
				{
					Name:     "eu-central-1c",
					Workers:  "10.250.0.128/27",
					Public:   "10.250.0.160/28",
					Internal: "10.250.0.176/28",
				},
				{
					Name:     "eu-central-1d",
					Workers:  "10.250.0.192/30",
					Public:   "10.250.0.196/30",
					Internal: "10.250.0.200/30",
				},
			},
		},
		"AWS five zones and 10.250.0.0/24": {
			givenNodesCidr: "10.250.0.0/24",
			givenZoneNames: []string{
				"eu-central-1a",
				"eu-central-1b",
				"eu-central-1c",
				"eu-central-1d",
				"eu-central-1e",
			},
			expectedAwsZones: []v1alpha1.Zone{
				{
					Name:     "eu-central-1a",
					Workers:  "10.250.0.0/27",
					Public:   "10.250.0.32/28",
					Internal: "10.250.0.48/28",
				},
				{
					Name:     "eu-central-1b",
					Workers:  "10.250.0.64/27",
					Public:   "10.250.0.96/28",
					Internal: "10.250.0.112/28",
				},
				{
					Name:     "eu-central-1c",
					Workers:  "10.250.0.128/27",
					Public:   "10.250.0.160/28",
					Internal: "10.250.0.176/28",
				},
				{
					Name:     "eu-central-1d",
					Workers:  "10.250.0.192/30",
					Public:   "10.250.0.196/30",
					Internal: "10.250.0.200/30",
				},
				{
					Name:     "eu-central-1e",
					Workers:  "10.250.0.204/30",
					Public:   "10.250.0.208/30",
					Internal: "10.250.0.212/30",
				},
			},
		},
		"AWS six zones and 10.250.0.0/24": {
			givenNodesCidr: "10.250.0.0/24",
			givenZoneNames: []string{
				"eu-central-1a",
				"eu-central-1b",
				"eu-central-1c",
				"eu-central-1d",
				"eu-central-1e",
				"eu-central-1f",
			},
			expectedAwsZones: []v1alpha1.Zone{
				{
					Name:     "eu-central-1a",
					Workers:  "10.250.0.0/27",
					Public:   "10.250.0.32/28",
					Internal: "10.250.0.48/28",
				},
				{
					Name:     "eu-central-1b",
					Workers:  "10.250.0.64/27",
					Public:   "10.250.0.96/28",
					Internal: "10.250.0.112/28",
				},
				{
					Name:     "eu-central-1c",
					Workers:  "10.250.0.128/27",
					Public:   "10.250.0.160/28",
					Internal: "10.250.0.176/28",
				},
				{
					Name:     "eu-central-1d",
					Workers:  "10.250.0.192/30",
					Public:   "10.250.0.196/30",
					Internal: "10.250.0.200/30",
				},
				{
					Name:     "eu-central-1e",
					Workers:  "10.250.0.204/30",
					Public:   "10.250.0.208/30",
					Internal: "10.250.0.212/30",
				},
				{
					Name:     "eu-central-1f",
					Workers:  "10.250.0.216/30",
					Public:   "10.250.0.220/30",
					Internal: "10.250.0.224/30",
				},
			},
		},
		"AWS seven zones and 10.250.0.0/24": {
			givenNodesCidr: "10.250.0.0/24",
			givenZoneNames: []string{
				"eu-central-1a",
				"eu-central-1b",
				"eu-central-1c",
				"eu-central-1d",
				"eu-central-1e",
				"eu-central-1f",
				"eu-central-1g",
			},
			expectedAwsZones: []v1alpha1.Zone{
				{
					Name:     "eu-central-1a",
					Workers:  "10.250.0.0/27",
					Public:   "10.250.0.32/28",
					Internal: "10.250.0.48/28",
				},
				{
					Name:     "eu-central-1b",
					Workers:  "10.250.0.64/27",
					Public:   "10.250.0.96/28",
					Internal: "10.250.0.112/28",
				},
				{
					Name:     "eu-central-1c",
					Workers:  "10.250.0.128/27",
					Public:   "10.250.0.160/28",
					Internal: "10.250.0.176/28",
				},
				{
					Name:     "eu-central-1d",
					Workers:  "10.250.0.192/30",
					Public:   "10.250.0.196/30",
					Internal: "10.250.0.200/30",
				},
				{
					Name:     "eu-central-1e",
					Workers:  "10.250.0.204/30",
					Public:   "10.250.0.208/30",
					Internal: "10.250.0.212/30",
				},
				{
					Name:     "eu-central-1f",
					Workers:  "10.250.0.216/30",
					Public:   "10.250.0.220/30",
					Internal: "10.250.0.224/30",
				},
				{
					Name:     "eu-central-1g",
					Workers:  "10.250.0.228/30",
					Public:   "10.250.0.232/30",
					Internal: "10.250.0.236/30",
				},
			},
		},
		"AWS eight zones and 10.250.0.0/24": {
			givenNodesCidr: "10.250.0.0/24",
			givenZoneNames: []string{
				"eu-central-1a",
				"eu-central-1b",
				"eu-central-1c",
				"eu-central-1d",
				"eu-central-1e",
				"eu-central-1f",
				"eu-central-1g",
				"eu-central-1h",
			},
			expectedAwsZones: []v1alpha1.Zone{
				{
					Name:     "eu-central-1a",
					Workers:  "10.250.0.0/27",
					Public:   "10.250.0.32/28",
					Internal: "10.250.0.48/28",
				},
				{
					Name:     "eu-central-1b",
					Workers:  "10.250.0.64/27",
					Public:   "10.250.0.96/28",
					Internal: "10.250.0.112/28",
				},
				{
					Name:     "eu-central-1c",
					Workers:  "10.250.0.128/27",
					Public:   "10.250.0.160/28",
					Internal: "10.250.0.176/28",
				},
				{
					Name:     "eu-central-1d",
					Workers:  "10.250.0.192/30",
					Public:   "10.250.0.196/30",
					Internal: "10.250.0.200/30",
				},
				{
					Name:     "eu-central-1e",
					Workers:  "10.250.0.204/30",
					Public:   "10.250.0.208/30",
					Internal: "10.250.0.212/30",
				},
				{
					Name:     "eu-central-1f",
					Workers:  "10.250.0.216/30",
					Public:   "10.250.0.220/30",
					Internal: "10.250.0.224/30",
				},
				{
					Name:     "eu-central-1g",
					Workers:  "10.250.0.228/30",
					Public:   "10.250.0.232/30",
					Internal: "10.250.0.236/30",
				},
				{
					Name:     "eu-central-1h",
					Workers:  "10.250.0.240/30",
					Public:   "10.250.0.244/30",
					Internal: "10.250.0.248/30",
				},
			},
		},
	} {
		t.Run(tname, func(t *testing.T) {
			zones, err := generateAWSZones(tcase.givenNodesCidr, tcase.givenZoneNames)

			assert.NoError(t, err)
			assert.Equal(t, len(tcase.expectedAwsZones), len(zones))

			for i, expectedZone := range tcase.expectedAwsZones {
				assertAWSZoneNetworkIPRanges(t, tcase.givenNodesCidr, expectedZone, zones[i])
			}
		})
	}

	for tname, tcase := range map[string]struct {
		givenNodesCidr string
		givenZoneNames []string
		message        string
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
			message: "Number of networking zones must be between 1 and 8",
		},
		"AWS should return error when duplicated zones names are provided": {
			givenNodesCidr: "10.250.0.0/16",
			givenZoneNames: []string{
				"eu-central-1a",
				"eu-central-1a",
			},
			message: "zone name eu-central-1a is duplicated",
		},
		"AWS should return error when 0 zones are provided": {
			givenNodesCidr: "10.180.0.0/23",
			givenZoneNames: []string{},
			message:        "Number of networking zones must be between 1 and 8",
		},

		"AWS should return error when cannot parse nodes CIDR": {
			givenNodesCidr: "888.888.888.0/77",
			givenZoneNames: []string{
				"eu-central-1a",
			},
			message: "failed to parse worker network CIDR",
		},
		"AWS should return error when prefix is too big for ex 10.250.0.0/25": {
			givenNodesCidr: "10.250.0.0/25",
			givenZoneNames: []string{
				"eu-central-1a",
			},
			message: "CIDR prefix length must be between 16 and 24",
		},
		"AWS should return error when prefix is too small for ex 10.250.0.0/15": {
			givenNodesCidr: "10.250.0.0/15",
			givenZoneNames: []string{
				"eu-central-1a",
			},
			message: "CIDR prefix length must be between 16 and 24",
		},
	} {
		t.Run(tname, func(t *testing.T) {
			zones, err := generateAWSZones(tcase.givenNodesCidr, tcase.givenZoneNames)

			assert.Error(t, err)
			assert.Contains(t, err.Error(), tcase.message)
			assert.Equal(t, 0, len(zones))
		})
	}
}

func assertAWSZoneNetworkIPRanges(t *testing.T, nodesCIDR string, expectedZone v1alpha1.Zone, checked v1alpha1.Zone) {
	result, err := networking.IsSubnetInsideWorkerCIDR(nodesCIDR, checked.Internal)
	assert.NoError(t, err)
	assert.True(t, result)

	result, err = networking.IsSubnetInsideWorkerCIDR(nodesCIDR, checked.Workers)
	assert.NoError(t, err)
	assert.True(t, result)

	result, err = networking.IsSubnetInsideWorkerCIDR(nodesCIDR, checked.Public)
	assert.NoError(t, err)
	assert.True(t, result)

	assert.Equal(t, expectedZone.Internal, checked.Internal)
	assert.Equal(t, expectedZone.Workers, checked.Workers)
	assert.Equal(t, expectedZone.Public, checked.Public)
	assert.Equal(t, expectedZone.Name, checked.Name)
}
