package azure

import (
	"github.com/pkg/errors"
	"math/big"
	"net/netip"
	"strconv"
)

const (
	defaultConnectionTimeOutMinutes = 4
	workersBits                     = 3
	cidrLength                      = 32
	maxNumberOfZones                = 8
)

func generateAzureZones(workerCidr string, zoneNames []string) ([]Zone, error) {
	numZones := len(zoneNames)
	if numZones < 1 || numZones > maxNumberOfZones {
		return nil, errors.New("Number of networking zones must be between 1 and 8")
	}

	var zones []Zone

	cidr, _ := netip.ParsePrefix(workerCidr)
	workerPrefixLength := cidr.Bits() + workersBits
	workerPrefix, _ := cidr.Addr().Prefix(workerPrefixLength)
	// delta - it is the difference between CIDRs of two zones:
	//    zone1:   "10.250.0.0/19",
	//    zone2:   "10.250.32.0/19",
	delta := big.NewInt(1)
	delta.Lsh(delta, uint(cidrLength-workerPrefixLength))

	// zoneIPValue - it is an integer, which is based on IP bytes
	zoneIPValue := new(big.Int).SetBytes(workerPrefix.Addr().AsSlice())

	for _, name := range convertZoneNames(zoneNames) {
		zoneWorkerIP, _ := netip.AddrFromSlice(zoneIPValue.Bytes())
		zoneWorkerCidr := netip.PrefixFrom(zoneWorkerIP, workerPrefixLength)
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
		if err != nil || zone < 1 || zone > 8 {
			continue
		}
		zones = append(zones, zone)
	}

	return zones
}
