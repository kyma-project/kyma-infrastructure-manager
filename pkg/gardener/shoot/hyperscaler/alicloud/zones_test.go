package alicloud

import (
	"github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud/v1alpha1"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/hyperscaler/networking"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestAlicloudZonesWithCustomNodeIPRange(t *testing.T) {
	for tname, tcase := range map[string]struct {
		givenNodesCidr string
		givenZoneNames []string
		expectedZones  []v1alpha1.Zone
	}{
		"Alicloud one zone and 10.250.0.0/16": {
			givenNodesCidr: "10.250.0.0/16",
			givenZoneNames: []string{
				"eu-central-1a",
			},
			expectedZones: []v1alpha1.Zone{
				{
					Name:    "eu-central-1a",
					Workers: "10.250.0.0/19",
				},
			},
		},
		"Alicloud two zones and 10.250.0.0/16": {
			givenNodesCidr: "10.250.0.0/16",
			givenZoneNames: []string{
				"eu-central-1a",
				"eu-central-1b",
			},
			expectedZones: []v1alpha1.Zone{
				{
					Name:    "eu-central-1a",
					Workers: "10.250.0.0/19",
				},
				{
					Name:    "eu-central-1b",
					Workers: "10.250.64.0/19",
				},
			},
		},
		"Alicloud three zones and 10.250.0.0/16": {
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
					Workers: "10.250.64.0/19",
				},
				{
					Name:    "eu-central-1c",
					Workers: "10.250.128.0/19",
				},
			},
		},
		"Alicloud four zones and 10.250.0.0/16": {
			givenNodesCidr: "10.250.0.0/16",
			givenZoneNames: []string{
				"eu-central-1a",
				"eu-central-1b",
				"eu-central-1c",
				"eu-central-1d",
			},
			expectedZones: []v1alpha1.Zone{
				{
					Name:    "eu-central-1a",
					Workers: "10.250.0.0/19",
				},
				{
					Name:    "eu-central-1b",
					Workers: "10.250.64.0/19",
				},
				{
					Name:    "eu-central-1c",
					Workers: "10.250.128.0/19",
				},
				{
					Name:    "eu-central-1d",
					Workers: "10.250.192.0/22",
				},
			},
		},
		"Alicloud five zones and 10.250.0.0/16": {
			givenNodesCidr: "10.250.0.0/16",
			givenZoneNames: []string{
				"eu-central-1a",
				"eu-central-1b",
				"eu-central-1c",
				"eu-central-1d",
				"eu-central-1e",
			},
			expectedZones: []v1alpha1.Zone{
				{
					Name:    "eu-central-1a",
					Workers: "10.250.0.0/19",
				},
				{
					Name:    "eu-central-1b",
					Workers: "10.250.64.0/19",
				},
				{
					Name:    "eu-central-1c",
					Workers: "10.250.128.0/19",
				},
				{
					Name:    "eu-central-1d",
					Workers: "10.250.192.0/22",
				},
				{
					Name:    "eu-central-1e",
					Workers: "10.250.204.0/22",
				},
			},
		},
		"Alicloud six zones and 10.250.0.0/16": {
			givenNodesCidr: "10.250.0.0/16",
			givenZoneNames: []string{
				"eu-central-1a",
				"eu-central-1b",
				"eu-central-1c",
				"eu-central-1d",
				"eu-central-1e",
				"eu-central-1f",
			},
			expectedZones: []v1alpha1.Zone{
				{
					Name:    "eu-central-1a",
					Workers: "10.250.0.0/19",
				},
				{
					Name:    "eu-central-1b",
					Workers: "10.250.64.0/19",
				},
				{
					Name:    "eu-central-1c",
					Workers: "10.250.128.0/19",
				},
				{
					Name:    "eu-central-1d",
					Workers: "10.250.192.0/22",
				},
				{
					Name:    "eu-central-1e",
					Workers: "10.250.204.0/22",
				},
				{
					Name:    "eu-central-1f",
					Workers: "10.250.216.0/22",
				},
			},
		},
		"Alicloud seven zones and 10.250.0.0/16": {
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
			expectedZones: []v1alpha1.Zone{
				{
					Name:    "eu-central-1a",
					Workers: "10.250.0.0/19",
				},
				{
					Name:    "eu-central-1b",
					Workers: "10.250.64.0/19",
				},
				{
					Name:    "eu-central-1c",
					Workers: "10.250.128.0/19",
				},
				{
					Name:    "eu-central-1d",
					Workers: "10.250.192.0/22",
				},
				{
					Name:    "eu-central-1e",
					Workers: "10.250.204.0/22",
				},
				{
					Name:    "eu-central-1f",
					Workers: "10.250.216.0/22",
				},
				{
					Name:    "eu-central-1g",
					Workers: "10.250.228.0/22",
				},
			},
		},
		"Alicloud eight zones and 10.250.0.0/16": {
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
			expectedZones: []v1alpha1.Zone{
				{
					Name:    "eu-central-1a",
					Workers: "10.250.0.0/19",
				},
				{
					Name:    "eu-central-1b",
					Workers: "10.250.64.0/19",
				},
				{
					Name:    "eu-central-1c",
					Workers: "10.250.128.0/19",
				},
				{
					Name:    "eu-central-1d",
					Workers: "10.250.192.0/22",
				},
				{
					Name:    "eu-central-1e",
					Workers: "10.250.204.0/22",
				},
				{
					Name:    "eu-central-1f",
					Workers: "10.250.216.0/22",
				},
				{
					Name:    "eu-central-1g",
					Workers: "10.250.228.0/22",
				},
				{
					Name:    "eu-central-1h",
					Workers: "10.250.240.0/22",
				},
			},
		},
		"Alicloud one zone and 10.250.0.0/22": {
			givenNodesCidr: "10.250.0.0/22",
			givenZoneNames: []string{
				"eu-central-1a",
			},
			expectedZones: []v1alpha1.Zone{
				{
					Name:    "eu-central-1a",
					Workers: "10.250.0.0/25",
				},
			},
		},
		"Alicloud two zones and 10.250.0.0/22": {
			givenNodesCidr: "10.250.0.0/22",
			givenZoneNames: []string{
				"eu-central-1a",
				"eu-central-1b",
			},
			expectedZones: []v1alpha1.Zone{
				{
					Name:    "eu-central-1a",
					Workers: "10.250.0.0/25",
				},
				{
					Name:    "eu-central-1b",
					Workers: "10.250.1.0/25",
				},
			},
		},
		"Alicloud three zones and 10.250.0.0/22": {
			givenNodesCidr: "10.250.0.0/22",
			givenZoneNames: []string{
				"eu-central-1a",
				"eu-central-1b",
				"eu-central-1c",
			},
			expectedZones: []v1alpha1.Zone{
				{
					Name:    "eu-central-1a",
					Workers: "10.250.0.0/25",
				},
				{
					Name:    "eu-central-1b",
					Workers: "10.250.1.0/25",
				},
				{
					Name:    "eu-central-1c",
					Workers: "10.250.2.0/25",
				},
			},
		},
		"Alicloud four zones and 10.250.0.0/22": {
			givenNodesCidr: "10.250.0.0/22",
			givenZoneNames: []string{
				"eu-central-1a",
				"eu-central-1b",
				"eu-central-1c",
				"eu-central-1d",
			},
			expectedZones: []v1alpha1.Zone{
				{
					Name:    "eu-central-1a",
					Workers: "10.250.0.0/25",
				},
				{
					Name:    "eu-central-1b",
					Workers: "10.250.1.0/25",
				},
				{
					Name:    "eu-central-1c",
					Workers: "10.250.2.0/25",
				},
				{
					Name:    "eu-central-1d",
					Workers: "10.250.3.0/28",
				},
			},
		},
		"Alicloud five zones and 10.250.0.0/22": {
			givenNodesCidr: "10.250.0.0/22",
			givenZoneNames: []string{
				"eu-central-1a",
				"eu-central-1b",
				"eu-central-1c",
				"eu-central-1d",
				"eu-central-1e",
			},
			expectedZones: []v1alpha1.Zone{
				{
					Name:    "eu-central-1a",
					Workers: "10.250.0.0/25",
				},
				{
					Name:    "eu-central-1b",
					Workers: "10.250.1.0/25",
				},
				{
					Name:    "eu-central-1c",
					Workers: "10.250.2.0/25",
				},
				{
					Name:    "eu-central-1d",
					Workers: "10.250.3.0/28",
				},
				{
					Name:    "eu-central-1e",
					Workers: "10.250.3.48/28",
				},
			},
		},
		"Alicloud six zones and 10.250.0.0/22": {
			givenNodesCidr: "10.250.0.0/22",
			givenZoneNames: []string{
				"eu-central-1a",
				"eu-central-1b",
				"eu-central-1c",
				"eu-central-1d",
				"eu-central-1e",
				"eu-central-1f",
			},
			expectedZones: []v1alpha1.Zone{
				{
					Name:    "eu-central-1a",
					Workers: "10.250.0.0/25",
				},
				{
					Name:    "eu-central-1b",
					Workers: "10.250.1.0/25",
				},
				{
					Name:    "eu-central-1c",
					Workers: "10.250.2.0/25",
				},
				{
					Name:    "eu-central-1d",
					Workers: "10.250.3.0/28",
				},
				{
					Name:    "eu-central-1e",
					Workers: "10.250.3.48/28",
				},
				{
					Name:    "eu-central-1f",
					Workers: "10.250.3.96/28",
				},
			},
		},
		"Alicloud seven zones and 10.250.0.0/22": {
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
			expectedZones: []v1alpha1.Zone{
				{
					Name:    "eu-central-1a",
					Workers: "10.250.0.0/25",
				},
				{
					Name:    "eu-central-1b",
					Workers: "10.250.1.0/25",
				},
				{
					Name:    "eu-central-1c",
					Workers: "10.250.2.0/25",
				},
				{
					Name:    "eu-central-1d",
					Workers: "10.250.3.0/28",
				},
				{
					Name:    "eu-central-1e",
					Workers: "10.250.3.48/28",
				},
				{
					Name:    "eu-central-1f",
					Workers: "10.250.3.96/28",
				},
				{
					Name:    "eu-central-1g",
					Workers: "10.250.3.144/28",
				},
			},
		},
		"Alicloud eight zones and 10.250.0.0/22": {
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
			expectedZones: []v1alpha1.Zone{
				{
					Name:    "eu-central-1a",
					Workers: "10.250.0.0/25",
				},
				{
					Name:    "eu-central-1b",
					Workers: "10.250.1.0/25",
				},
				{
					Name:    "eu-central-1c",
					Workers: "10.250.2.0/25",
				},
				{
					Name:    "eu-central-1d",
					Workers: "10.250.3.0/28",
				},
				{
					Name:    "eu-central-1e",
					Workers: "10.250.3.48/28",
				},
				{
					Name:    "eu-central-1f",
					Workers: "10.250.3.96/28",
				},
				{
					Name:    "eu-central-1g",
					Workers: "10.250.3.144/28",
				},
				{
					Name:    "eu-central-1h",
					Workers: "10.250.3.192/28",
				},
			},
		},
		"Alicloud three zones 10.180.0.0/23": {
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
					Workers: "10.180.0.128/26",
				},
				{
					Name:    "eu-central-1c",
					Workers: "10.180.1.0/26",
				},
			},
		},
		"Alicloud one zone and 10.250.0.0/23": {
			givenNodesCidr: "10.250.0.0/23",
			givenZoneNames: []string{
				"eu-central-1a",
			},
			expectedZones: []v1alpha1.Zone{
				{
					Name:    "eu-central-1a",
					Workers: "10.250.0.0/26",
				},
			},
		},
		"Alicloud two zones and 10.250.0.0/23": {
			givenNodesCidr: "10.250.0.0/23",
			givenZoneNames: []string{
				"eu-central-1a",
				"eu-central-1b",
			},
			expectedZones: []v1alpha1.Zone{
				{
					Name:    "eu-central-1a",
					Workers: "10.250.0.0/26",
				},
				{
					Name:    "eu-central-1b",
					Workers: "10.250.0.128/26",
				},
			},
		},
		"Alicloud three zones and 10.250.0.0/23": {
			givenNodesCidr: "10.250.0.0/23",
			givenZoneNames: []string{
				"eu-central-1a",
				"eu-central-1b",
				"eu-central-1c",
			},
			expectedZones: []v1alpha1.Zone{
				{
					Name:    "eu-central-1a",
					Workers: "10.250.0.0/26",
				},
				{
					Name:    "eu-central-1b",
					Workers: "10.250.0.128/26",
				},
				{
					Name:    "eu-central-1c",
					Workers: "10.250.1.0/26",
				},
			},
		},
		"Alicloud four zones and 10.250.0.0/23": {
			givenNodesCidr: "10.250.0.0/23",
			givenZoneNames: []string{
				"eu-central-1a",
				"eu-central-1b",
				"eu-central-1c",
				"eu-central-1d",
			},
			expectedZones: []v1alpha1.Zone{
				{
					Name:    "eu-central-1a",
					Workers: "10.250.0.0/26",
				},
				{
					Name:    "eu-central-1b",
					Workers: "10.250.0.128/26",
				},
				{
					Name:    "eu-central-1c",
					Workers: "10.250.1.0/26",
				},
				{
					Name:    "eu-central-1d",
					Workers: "10.250.1.128/29",
				},
			},
		},
		"Alicloud five zones and 10.250.0.0/23": {
			givenNodesCidr: "10.250.0.0/23",
			givenZoneNames: []string{
				"eu-central-1a",
				"eu-central-1b",
				"eu-central-1c",
				"eu-central-1d",
				"eu-central-1e",
			},
			expectedZones: []v1alpha1.Zone{
				{
					Name:    "eu-central-1a",
					Workers: "10.250.0.0/26",
				},
				{
					Name:    "eu-central-1b",
					Workers: "10.250.0.128/26",
				},
				{
					Name:    "eu-central-1c",
					Workers: "10.250.1.0/26",
				},
				{
					Name:    "eu-central-1d",
					Workers: "10.250.1.128/29",
				},
				{
					Name:    "eu-central-1e",
					Workers: "10.250.1.152/29",
				},
			},
		},
		"Alicloud six zones and 10.250.0.0/23": {
			givenNodesCidr: "10.250.0.0/23",
			givenZoneNames: []string{
				"eu-central-1a",
				"eu-central-1b",
				"eu-central-1c",
				"eu-central-1d",
				"eu-central-1e",
				"eu-central-1f",
			},
			expectedZones: []v1alpha1.Zone{
				{
					Name:    "eu-central-1a",
					Workers: "10.250.0.0/26",
				},
				{
					Name:    "eu-central-1b",
					Workers: "10.250.0.128/26",
				},
				{
					Name:    "eu-central-1c",
					Workers: "10.250.1.0/26",
				},
				{
					Name:    "eu-central-1d",
					Workers: "10.250.1.128/29",
				},
				{
					Name:    "eu-central-1e",
					Workers: "10.250.1.152/29",
				},
				{
					Name:    "eu-central-1f",
					Workers: "10.250.1.176/29",
				},
			},
		},
		"Alicloud seven zones and 10.250.0.0/23": {
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
			expectedZones: []v1alpha1.Zone{
				{
					Name:    "eu-central-1a",
					Workers: "10.250.0.0/26",
				},
				{
					Name:    "eu-central-1b",
					Workers: "10.250.0.128/26",
				},
				{
					Name:    "eu-central-1c",
					Workers: "10.250.1.0/26",
				},
				{
					Name:    "eu-central-1d",
					Workers: "10.250.1.128/29",
				},
				{
					Name:    "eu-central-1e",
					Workers: "10.250.1.152/29",
				},
				{
					Name:    "eu-central-1f",
					Workers: "10.250.1.176/29",
				},
				{
					Name:    "eu-central-1g",
					Workers: "10.250.1.200/29",
				},
			},
		},
		"Alicloud eight zones and 10.250.0.0/23": {
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
			expectedZones: []v1alpha1.Zone{
				{
					Name:    "eu-central-1a",
					Workers: "10.250.0.0/26",
				},
				{
					Name:    "eu-central-1b",
					Workers: "10.250.0.128/26",
				},
				{
					Name:    "eu-central-1c",
					Workers: "10.250.1.0/26",
				},
				{
					Name:    "eu-central-1d",
					Workers: "10.250.1.128/29",
				},
				{
					Name:    "eu-central-1e",
					Workers: "10.250.1.152/29",
				},
				{
					Name:    "eu-central-1f",
					Workers: "10.250.1.176/29",
				},
				{
					Name:    "eu-central-1g",
					Workers: "10.250.1.200/29",
				},
				{
					Name:    "eu-central-1h",
					Workers: "10.250.1.224/29",
				},
			},
		},
		"Alicloud one zone and 10.250.0.0/24": {
			givenNodesCidr: "10.250.0.0/24",
			givenZoneNames: []string{
				"eu-central-1a",
			},
			expectedZones: []v1alpha1.Zone{
				{
					Name:    "eu-central-1a",
					Workers: "10.250.0.0/27",
				},
			},
		},
		"Alicloud two zones and 10.250.0.0/24": {
			givenNodesCidr: "10.250.0.0/24",
			givenZoneNames: []string{
				"eu-central-1a",
				"eu-central-1b",
			},
			expectedZones: []v1alpha1.Zone{
				{
					Name:    "eu-central-1a",
					Workers: "10.250.0.0/27",
				},
				{
					Name:    "eu-central-1b",
					Workers: "10.250.0.64/27",
				},
			},
		},
		"Alicloud three zones and 10.250.0.0/24": {
			givenNodesCidr: "10.250.0.0/24",
			givenZoneNames: []string{
				"eu-central-1a",
				"eu-central-1b",
				"eu-central-1c",
			},
			expectedZones: []v1alpha1.Zone{
				{
					Name:    "eu-central-1a",
					Workers: "10.250.0.0/27",
				},
				{
					Name:    "eu-central-1b",
					Workers: "10.250.0.64/27",
				},
				{
					Name:    "eu-central-1c",
					Workers: "10.250.0.128/27",
				},
			},
		},
		"Alicloud four zones and 10.250.0.0/24": {
			givenNodesCidr: "10.250.0.0/24",
			givenZoneNames: []string{
				"eu-central-1a",
				"eu-central-1b",
				"eu-central-1c",
				"eu-central-1d",
			},
			expectedZones: []v1alpha1.Zone{
				{
					Name:    "eu-central-1a",
					Workers: "10.250.0.0/27",
				},
				{
					Name:    "eu-central-1b",
					Workers: "10.250.0.64/27",
				},
				{
					Name:    "eu-central-1c",
					Workers: "10.250.0.128/27",
				},
				{
					Name:    "eu-central-1d",
					Workers: "10.250.0.192/30",
				},
			},
		},
		"Alicloud five zones and 10.250.0.0/24": {
			givenNodesCidr: "10.250.0.0/24",
			givenZoneNames: []string{
				"eu-central-1a",
				"eu-central-1b",
				"eu-central-1c",
				"eu-central-1d",
				"eu-central-1e",
			},
			expectedZones: []v1alpha1.Zone{
				{
					Name:    "eu-central-1a",
					Workers: "10.250.0.0/27",
				},
				{
					Name:    "eu-central-1b",
					Workers: "10.250.0.64/27",
				},
				{
					Name:    "eu-central-1c",
					Workers: "10.250.0.128/27",
				},
				{
					Name:    "eu-central-1d",
					Workers: "10.250.0.192/30",
				},
				{
					Name:    "eu-central-1e",
					Workers: "10.250.0.204/30",
				},
			},
		},
		"Alicloud six zones and 10.250.0.0/24": {
			givenNodesCidr: "10.250.0.0/24",
			givenZoneNames: []string{
				"eu-central-1a",
				"eu-central-1b",
				"eu-central-1c",
				"eu-central-1d",
				"eu-central-1e",
				"eu-central-1f",
			},
			expectedZones: []v1alpha1.Zone{
				{
					Name:    "eu-central-1a",
					Workers: "10.250.0.0/27",
				},
				{
					Name:    "eu-central-1b",
					Workers: "10.250.0.64/27",
				},
				{
					Name:    "eu-central-1c",
					Workers: "10.250.0.128/27",
				},
				{
					Name:    "eu-central-1d",
					Workers: "10.250.0.192/30",
				},
				{
					Name:    "eu-central-1e",
					Workers: "10.250.0.204/30",
				},
				{
					Name:    "eu-central-1f",
					Workers: "10.250.0.216/30",
				},
			},
		},
		"Alicloud seven zones and 10.250.0.0/24": {
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
			expectedZones: []v1alpha1.Zone{
				{
					Name:    "eu-central-1a",
					Workers: "10.250.0.0/27",
				},
				{
					Name:    "eu-central-1b",
					Workers: "10.250.0.64/27",
				},
				{
					Name:    "eu-central-1c",
					Workers: "10.250.0.128/27",
				},
				{
					Name:    "eu-central-1d",
					Workers: "10.250.0.192/30",
				},
				{
					Name:    "eu-central-1e",
					Workers: "10.250.0.204/30",
				},
				{
					Name:    "eu-central-1f",
					Workers: "10.250.0.216/30",
				},
				{
					Name:    "eu-central-1g",
					Workers: "10.250.0.228/30",
				},
			},
		},
		"Alicloud eight zones and 10.250.0.0/24": {
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
			expectedZones: []v1alpha1.Zone{
				{
					Name:    "eu-central-1a",
					Workers: "10.250.0.0/27",
				},
				{
					Name:    "eu-central-1b",
					Workers: "10.250.0.64/27",
				},
				{
					Name:    "eu-central-1c",
					Workers: "10.250.0.128/27",
				},
				{
					Name:    "eu-central-1d",
					Workers: "10.250.0.192/30",
				},
				{
					Name:    "eu-central-1e",
					Workers: "10.250.0.204/30",
				},
				{
					Name:    "eu-central-1f",
					Workers: "10.250.0.216/30",
				},
				{
					Name:    "eu-central-1g",
					Workers: "10.250.0.228/30",
				},
				{
					Name:    "eu-central-1h",
					Workers: "10.250.0.240/30",
				},
			},
		},
	} {
		t.Run(tname, func(t *testing.T) {
			zones, err := generateZones(tcase.givenNodesCidr, tcase.givenZoneNames)

			assert.NoError(t, err)
			assert.Equal(t, len(tcase.expectedZones), len(zones))

			for i, expectedZone := range tcase.expectedZones {
				assertAlicloudZoneNetworkIPRanges(t, tcase.givenNodesCidr, expectedZone, zones[i])
			}
		})
	}

	for tname, tcase := range map[string]struct {
		givenNodesCidr string
		givenZoneNames []string
		message        string
	}{
		"Alicloud should return error when more than 8 zones provided": {
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
		"Alicloud should return error when duplicated zones names are provided": {
			givenNodesCidr: "10.250.0.0/16",
			givenZoneNames: []string{
				"eu-central-1a",
				"eu-central-1a",
			},
			message: "zone name eu-central-1a is duplicated",
		},
		"Alicloud should return error when 0 zones are provided": {
			givenNodesCidr: "10.180.0.0/23",
			givenZoneNames: []string{},
			message:        "Number of networking zones must be between 1 and 8",
		},

		"Alicloud should return error when cannot parse nodes CIDR": {
			givenNodesCidr: "888.888.888.0/77",
			givenZoneNames: []string{
				"eu-central-1a",
			},
			message: "failed to parse worker network CIDR",
		},
		"Alicloud should return error when prefix is too big for ex 10.250.0.0/25": {
			givenNodesCidr: "10.250.0.0/25",
			givenZoneNames: []string{
				"eu-central-1a",
			},
			message: "CIDR prefix length must be between 16 and 24",
		},
		"Alicloud should return error when prefix is too small for ex 10.250.0.0/15": {
			givenNodesCidr: "10.250.0.0/15",
			givenZoneNames: []string{
				"eu-central-1a",
			},
			message: "CIDR prefix length must be between 16 and 24",
		},
	} {
		t.Run(tname, func(t *testing.T) {
			zones, err := generateZones(tcase.givenNodesCidr, tcase.givenZoneNames)

			assert.Error(t, err)
			assert.Contains(t, err.Error(), tcase.message)
			assert.Equal(t, 0, len(zones))
		})
	}
}

func assertAlicloudZoneNetworkIPRanges(t *testing.T, nodesCIDR string, expectedZone v1alpha1.Zone, checked v1alpha1.Zone) {
	result, err := networking.IsSubnetInsideWorkerCIDR(nodesCIDR, checked.Workers)
	assert.NoError(t, err)
	assert.True(t, result)

	assert.Equal(t, expectedZone.Workers, checked.Workers)
	assert.Equal(t, expectedZone.Name, checked.Name)
}
