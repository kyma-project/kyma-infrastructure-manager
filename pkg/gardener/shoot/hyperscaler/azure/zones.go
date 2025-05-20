package azure

import (
	"github.com/pkg/errors"
	"math/big"
	"net/netip"
	"strconv"
)

const (
	defaultConnectionTimeOutMinutes = 4
	subNetworkBitsSize              = 3
	cidrLength                      = 32
	maxNumberOfZones                = 8
	minNumberOfZones                = 1
)

func generateAzureZones(workerCidr string, zoneNames []string) ([]Zone, error) {
	numZones := len(zoneNames)
	// old Azure lite clusters have no zones in InfrastructureConfig
	if numZones > maxNumberOfZones {
		return nil, errors.New("Number of networking zones must be between 0 and 8")
	}

	var zones []Zone

	cidr, err := netip.ParsePrefix(workerCidr)
	if err != nil {
		return zones, errors.Wrap(err, "failed to parse worker network CIDR")
	}

	prefixLength := cidr.Bits()

	if prefixLength > 24 || prefixLength < 16 {
		return nil, errors.New("CIDR prefix length must be between 16 and 24")
	}

	workerNetworkPrefixLength := cidr.Bits() + subNetworkBitsSize
	workerPrefix, _ := cidr.Addr().Prefix(workerNetworkPrefixLength)
	// delta - it is the difference between CIDRs of two zones:
	//    zone1:   "10.250.0.0/19",
	//    zone2:   "10.250.32.0/19",
	delta := big.NewInt(1)
	delta.Lsh(delta, uint(cidrLength-workerNetworkPrefixLength))

	// zoneIPValue - it is an integer, which is based on IP bytes
	zoneIPValue := new(big.Int).SetBytes(workerPrefix.Addr().AsSlice())

	processed := make(map[int]bool)

	for _, name := range convertZoneNames(zoneNames) {

		if _, ok := processed[name]; ok {
			return nil, errors.Errorf("zone name %d is duplicated", name)
		}
		processed[name] = true

		zoneWorkerIP, _ := netip.AddrFromSlice(zoneIPValue.Bytes())
		zoneWorkerCidr := netip.PrefixFrom(zoneWorkerIP, workerNetworkPrefixLength)
		zoneIPValue.Add(zoneIPValue, delta)
		zones = append(zones, Zone{
			Name: name,
			CIDR: zoneWorkerCidr.String(),
			NatGateway: &NatGateway{
				// There are existing Azure clusters which were created before NAT gateway support,
				// and they were migrated to HA with all zones having enableNatGateway: false .
				// But for new Azure runtimes, enableNatGateway for all zones is always true
				Enabled:                      true,
				IdleConnectionTimeoutMinutes: defaultConnectionTimeOutMinutes,
			},
		})
	}
	return zones, nil
}

func convertZoneNames(zoneNames []string) []int {
	var zones []int
	for _, inputZone := range zoneNames {
		zone, err := strconv.Atoi(inputZone)
		if err != nil || zone < minNumberOfZones || zone > maxNumberOfZones {
			continue
		}
		zones = append(zones, zone)
	}

	return zones
}
