package networking

import (
	"github.com/pkg/errors"
	"net/netip"
)

// isSubnetInsideWorkerCIDR verifies if the given subnet CIDR is within the worker CIDR.
func IsSubnetInsideWorkerCIDR(workerCIDR string, subnetCIDR string) (bool, error) {
	workerPrefix, err := netip.ParsePrefix(workerCIDR)
	if err != nil {
		return false, errors.Wrap(err, "failed to parse worker CIDR")
	}

	subnetPrefix, err := netip.ParsePrefix(subnetCIDR)
	if err != nil {
		return false, errors.Wrap(err, "failed to parse subnet CIDR")
	}

	// Check if the subnet is contained within the worker CIDR
	return workerPrefix.Contains(subnetPrefix.Addr()), nil
}
