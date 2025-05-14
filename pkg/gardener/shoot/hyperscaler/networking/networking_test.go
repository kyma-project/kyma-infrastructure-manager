package networking

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestIsSubnetInsideWorkerCIDR(t *testing.T) {
	for tname, tcase := range map[string]struct {
		givenNodesCidr  string
		givenSubnetCidr string
		expected        bool
	}{
		"Should return true when subnet CIDR is inside Nodes CIDR": {
			givenNodesCidr:  "10.250.0.0/16",
			givenSubnetCidr: "10.250.0.0/19",
			expected:        true,
		},
		"Should return false when subnet CIDR is outside Nodes CIDR": {
			givenNodesCidr:  "10.250.0.0/16",
			givenSubnetCidr: "10.251.0.0/19",
			expected:        false,
		},
	} {
		t.Run(tname, func(t *testing.T) {
			result, err := IsSubnetInsideWorkerCIDR(tcase.givenNodesCidr, tcase.givenSubnetCidr)
			assert.NoError(t, err)

			if tcase.expected {
				assert.True(t, result)
			} else {
				assert.False(t, result)
			}
		})
	}
}
