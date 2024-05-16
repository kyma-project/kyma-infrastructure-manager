package azure

import (
	"math/big"
	"math/rand"
	"net/netip"
)

func generateRandomAzureZones(zonesCount int) []int {
	zones := []int{1, 2, 3}
	if zonesCount > 3 {
		zonesCount = 3
	}

	rand.Shuffle(len(zones), func(i, j int) { zones[i], zones[j] = zones[j], zones[i] })
	return zones[:zonesCount]
}

func generateAzureZones(workerCidr string, zoneNames []int) []*Zone {
	var zones []*Zone

	cidr, _ := netip.ParsePrefix(workerCidr)
	workerPrefixLength := cidr.Bits() + 3
	workerPrefix, _ := cidr.Addr().Prefix(workerPrefixLength)
	// delta - it is the difference between CIDRs of two zones:
	//    zone1:   "10.250.0.0/19",
	//    zone2:   "10.250.32.0/19",
	delta := big.NewInt(1)
	delta.Lsh(delta, uint(32-workerPrefixLength))

	// zoneIPValue - it is an integer, which is based on IP bytes
	zoneIPValue := new(big.Int).SetBytes(workerPrefix.Addr().AsSlice())

	for _, name := range zoneNames {
		zoneWorkerIP, _ := netip.AddrFromSlice(zoneIPValue.Bytes())
		zoneWorkerCidr := netip.PrefixFrom(zoneWorkerIP, workerPrefixLength)
		zoneIPValue.Add(zoneIPValue, delta)
		zones = append(zones, &Zone{
			Name: name,
			CIDR: zoneWorkerCidr.String(),
		})
	}
	return zones
}
