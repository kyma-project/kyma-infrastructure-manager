package gdch

import (
	"errors"
	"fmt"
	"testing"

	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/hyperscaler/networking"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateGDCHZones_HappyPaths(t *testing.T) {
	tests := map[string]struct {
		nodeCIDR  string
		zoneNames []string
		want      []Zone
	}{
		"from /24 to three /26": {
			nodeCIDR:  "10.72.0.0/24",
			zoneNames: []string{"us-west16-b", "us-west16-c", "us-west16-d"},
			want: []Zone{
				{Name: "us-west16-b", CIDR: "10.72.0.0/26"},
				{Name: "us-west16-c", CIDR: "10.72.0.64/26"},
				{Name: "us-west16-d", CIDR: "10.72.0.128/26"},
			},
		},
		"from /16 to three /18": {
			nodeCIDR:  "10.180.0.0/16",
			zoneNames: []string{"a", "b", "c"},
			want: []Zone{
				{Name: "a", CIDR: "10.180.0.0/18"},
				{Name: "b", CIDR: "10.180.64.0/18"},
				{Name: "c", CIDR: "10.180.128.0/18"},
			},
		},
		"from /22 to three /24": {
			nodeCIDR:  "10.72.0.0/22",
			zoneNames: []string{"a", "b", "c"},
			want: []Zone{
				{Name: "a", CIDR: "10.72.0.0/24"},
				{Name: "b", CIDR: "10.72.1.0/24"},
				{Name: "c", CIDR: "10.72.2.0/24"},
			},
		},
		"input order preserved (reversed names)": {
			nodeCIDR:  "10.72.0.0/24",
			zoneNames: []string{"us-west16-d", "us-west16-c", "us-west16-b"},
			want: []Zone{
				{Name: "us-west16-d", CIDR: "10.72.0.0/26"},
				{Name: "us-west16-c", CIDR: "10.72.0.64/26"},
				{Name: "us-west16-b", CIDR: "10.72.0.128/26"},
			},
		},
		"N=3 boundary /30 to three /32": {
			nodeCIDR:  "10.72.0.0/30",
			zoneNames: []string{"a", "b", "c"},
			want: []Zone{
				{Name: "a", CIDR: "10.72.0.0/32"},
				{Name: "b", CIDR: "10.72.0.1/32"},
				{Name: "c", CIDR: "10.72.0.2/32"},
			},
		},
		"N=1 single zone at landscape default /19": {
			nodeCIDR:  "10.72.0.0/19",
			zoneNames: []string{"a"},
			want: []Zone{
				{Name: "a", CIDR: "10.72.0.0/19"},
			},
		},
		"N=1 preserves full /24 block": {
			nodeCIDR:  "10.72.0.0/24",
			zoneNames: []string{"a"},
			want: []Zone{
				{Name: "a", CIDR: "10.72.0.0/24"},
			},
		},
		"N=2 landscape default /19 to two /20": {
			nodeCIDR:  "10.72.0.0/19",
			zoneNames: []string{"a", "b"},
			want: []Zone{
				{Name: "a", CIDR: "10.72.0.0/20"},
				{Name: "b", CIDR: "10.72.16.0/20"},
			},
		},
		"N=2 /24 to two /25": {
			nodeCIDR:  "10.72.0.0/24",
			zoneNames: []string{"a", "b"},
			want: []Zone{
				{Name: "a", CIDR: "10.72.0.0/25"},
				{Name: "b", CIDR: "10.72.0.128/25"},
			},
		},
		"N=3 landscape default /19 to three /21": {
			nodeCIDR:  "10.72.0.0/19",
			zoneNames: []string{"a", "b", "c"},
			want: []Zone{
				{Name: "a", CIDR: "10.72.0.0/21"},
				{Name: "b", CIDR: "10.72.8.0/21"},
				{Name: "c", CIDR: "10.72.16.0/21"},
			},
		},
		"N=2 boundary /31 to two /32": {
			nodeCIDR:  "10.72.0.0/31",
			zoneNames: []string{"a", "b"},
			want: []Zone{
				{Name: "a", CIDR: "10.72.0.0/32"},
				{Name: "b", CIDR: "10.72.0.1/32"},
			},
		},
		"N=1 boundary /32 to single /32": {
			nodeCIDR:  "10.72.0.0/32",
			zoneNames: []string{"a"},
			want: []Zone{
				{Name: "a", CIDR: "10.72.0.0/32"},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := generateGDCHZones(tc.nodeCIDR, tc.zoneNames)
			require.NoError(t, err)
			require.Len(t, got, len(tc.want))

			for i, wantZone := range tc.want {
				assert.Equal(t, wantZone.Name, got[i].Name, "zone %d name", i)
				assert.Equal(t, wantZone.CIDR, got[i].CIDR, "zone %d CIDR", i)

				contained, cerr := networking.IsSubnetInsideWorkerCIDR(tc.nodeCIDR, got[i].CIDR)
				require.NoError(t, cerr)
				assert.True(t, contained,
					"zone %d CIDR %s is not inside nodeCIDR %s",
					i, got[i].CIDR, tc.nodeCIDR)
			}
		})
	}
}

func TestGenerateGDCHZones_ErrorPaths(t *testing.T) {
	tests := map[string]struct {
		nodeCIDR  string
		zoneNames []string
		wantErr   error
	}{
		"malformed CIDR - nonsense string": {
			nodeCIDR:  "not a cidr",
			zoneNames: []string{"a", "b", "c"},
			wantErr:   errInvalidCIDR,
		},
		"malformed CIDR - out-of-range octets": {
			nodeCIDR:  "888.888.888.0/77",
			zoneNames: []string{"a", "b", "c"},
			wantErr:   errInvalidCIDR,
		},
		"IPv6 rejected": {
			nodeCIDR:  "2001:db8::/48",
			zoneNames: []string{"a", "b", "c"},
			wantErr:   errInvalidCIDR,
		},
		"zero zones": {
			nodeCIDR:  "10.72.0.0/24",
			zoneNames: []string{},
			wantErr:   errInvalidZoneCount,
		},
		"nil zones": {
			nodeCIDR:  "10.72.0.0/24",
			zoneNames: nil,
			wantErr:   errInvalidZoneCount,
		},
		"four zones": {
			nodeCIDR:  "10.72.0.0/24",
			zoneNames: []string{"a", "b", "c", "d"},
			wantErr:   errInvalidZoneCount,
		},
		"duplicate zones": {
			nodeCIDR:  "10.72.0.0/24",
			zoneNames: []string{"b", "c", "b"},
			wantErr:   errDuplicateZone,
		},
		"duplicate zones – 2 zones": {
			nodeCIDR:  "10.72.0.0/24",
			zoneNames: []string{"a", "a"},
			wantErr:   errDuplicateZone,
		},
		"N=3 CIDR too small – /31": {
			nodeCIDR:  "10.72.0.0/31",
			zoneNames: []string{"a", "b", "c"},
			wantErr:   errCIDRTooSmall,
		},
		"N=3 CIDR too small – /32": {
			nodeCIDR:  "10.72.0.0/32",
			zoneNames: []string{"a", "b", "c"},
			wantErr:   errCIDRTooSmall,
		},
		"N=2 CIDR too small – /32": {
			nodeCIDR:  "10.72.0.0/32",
			zoneNames: []string{"a", "b"},
			wantErr:   errCIDRTooSmall,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := generateGDCHZones(tc.nodeCIDR, tc.zoneNames)
			require.Error(t, err)
			assert.Nil(t, got)
			assert.True(t, errors.Is(err, tc.wantErr),
				"error %v does not match sentinel error %v", err, tc.wantErr)
		})
	}
}

func TestSubnetShiftForZoneCount(t *testing.T) {
	cases := map[int]int{
		1: 0,
		2: 1,
		3: 2,
	}
	for n, wantShift := range cases {
		t.Run(fmt.Sprintf("n=%d", n), func(t *testing.T) {
			assert.Equal(t, wantShift, subnetShiftForZoneCount(n))
		})
	}
}
