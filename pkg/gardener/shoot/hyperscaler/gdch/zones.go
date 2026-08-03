package gdch

import (
	"errors"
	"fmt"
	"net/netip"
)

const (
	minZoneCount = 1
	maxZoneCount = 3
)

var (
	errInvalidCIDR      = errors.New("invalid nodeCIDR: must be a parseable IPv4 CIDR")
	errInvalidZoneCount = errors.New("zone count must be between 1 and 3")
	errDuplicateZone    = errors.New("zone names must be unique")
	errCIDRTooSmall     = errors.New("nodeCIDR prefix too small for requested zone count")
)

func subnetShiftForZoneCount(n int) int {
	switch n {
	case 1:
		return 0
	case 2:
		return 1
	case 3:
		return 2
	}

	return 0
}

func generateGDCHZones(nodeCIDR string, zoneNames []string) ([]Zone, error) {
	n := len(zoneNames)
	if n < minZoneCount || n > maxZoneCount {
		return nil, fmt.Errorf("got %d: %w", n, errInvalidZoneCount)
	}

	prefix, err := netip.ParsePrefix(nodeCIDR)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", nodeCIDR, errInvalidCIDR)
	}
	if !prefix.Addr().Is4() {
		return nil, fmt.Errorf("%s is not IPv4: %w", nodeCIDR, errInvalidCIDR)
	}

	shift := subnetShiftForZoneCount(n)
	newBits := prefix.Bits() + shift
	if newBits > 32 {
		return nil, fmt.Errorf("prefix /%d too small for %d zones: %w", prefix.Bits(), n, errCIDRTooSmall)
	}

	step := uint32(1) << (32 - newBits)
	addr := prefix.Masked().Addr()

	zones := make([]Zone, 0, n)
	processed := make(map[string]struct{}, n)
	for _, name := range zoneNames {
		if _, dup := processed[name]; dup {
			return nil, fmt.Errorf("%q repeated: %w", name, errDuplicateZone)
		}
		processed[name] = struct{}{}

		zones = append(zones, Zone{
			Name: name,
			CIDR: netip.PrefixFrom(addr, newBits).String(),
		})
		addr = addAddrOffset(addr, step)
	}
	return zones, nil
}

func addAddrOffset(a netip.Addr, offset uint32) netip.Addr {
	b := a.As4()
	carry := offset
	for i := 3; i >= 0 && carry > 0; i-- {
		sum := uint32(b[i]) + (carry & 0xff)
		b[i] = byte(sum)
		carry = (carry >> 8) + (sum >> 8)
	}
	return netip.AddrFrom4(b)
}
