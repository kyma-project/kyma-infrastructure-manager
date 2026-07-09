package gdch

import (
	"errors"
	"fmt"
	"net/netip"
)

const (
	zoneCount       = 3
	subnetBitsShift = 2
)

var (
	errInvalidCIDR      = errors.New("invalid nodeCIDR: must be a parseable IPv4 CIDR")
	errInvalidZoneCount = errors.New("exactly 3 zone names required")
	errDuplicateZone    = errors.New("zone names must be unique")
	errCIDRTooSmall     = errors.New("nodeCIDR prefix length must be <= 30")
)

func generateGDCHZones(nodeCIDR string, zoneNames []string) ([]Zones, error) {
	if len(zoneNames) != zoneCount {
		return nil, fmt.Errorf("got %d: %w", len(zoneNames), errInvalidZoneCount)
	}

	prefix, err := netip.ParsePrefix(nodeCIDR)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", nodeCIDR, errInvalidCIDR)
	}
	if !prefix.Addr().Is4() {
		return nil, fmt.Errorf("%s is not IPv4: %w", nodeCIDR, errInvalidCIDR)
	}

	newBits := prefix.Bits() + subnetBitsShift
	if newBits > 32 {
		return nil, fmt.Errorf("prefix /%d too small: %w", prefix.Bits(), errCIDRTooSmall)
	}

	step := uint32(1) << (32 - newBits)
	addr := prefix.Masked().Addr()

	zones := make([]Zones, 0, zoneCount)
	processed := make(map[string]struct{}, zoneCount)
	for _, name := range zoneNames {
		if _, dup := processed[name]; dup {
			return nil, fmt.Errorf("%q repeated: %w", name, errDuplicateZone)
		}
		processed[name] = struct{}{}

		zones = append(zones, Zones{
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
