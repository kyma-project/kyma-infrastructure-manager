package alicloud

import (
	"fmt"
	"github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud/v1alpha1"
	"github.com/pkg/errors"
	"math/big"
	"net/netip"
)

const (
	subNetworkBitsSize              = 3
	cidrLength                      = 32
	maxNumberOfZones                = 8
)

func generateZones(workerCidr string, zoneNames []string) ([]v1alpha1.Zone, error) {
	numZones := len(zoneNames)
	if numZones > maxNumberOfZones {
		return nil, errors.New("Number of networking zones must be between 0 and 8")
	}

	var zones []v1alpha1.Zone

	cidr, err := netip.ParsePrefix(workerCidr)
	if err != nil {
		return zones, errors.Wrap(err, "failed to parse worker network CIDR")
	}

	prefixLength := cidr.Bits()

	if prefixLength > 24 {
		return nil, errors.New("CIDR prefix length must be less than or equal to 24")
	}

	if prefixLength < 16 {
		return nil, errors.New("CIDR prefix length must be bigger than or equal to 16")
	}


	workerNetworkPrefixLength := prefixLength + subNetworkBitsSize
	workerPrefix, err := cidr.Addr().Prefix(workerNetworkPrefixLength)
	if err != nil {
		return nil, err
	}

	// delta - it is the difference between CIDRs of two zones:
	//    zone1:   "10.250.0.0/19",
	//    zone2:   "10.250.32.0/19",
	delta := big.NewInt(1)
	delta.Lsh(delta, uint(cidrLength-workerNetworkPrefixLength))

	// zoneIPValue - it is an integer, which is based on IP bytes
	zoneIPValue := new(big.Int).SetBytes(workerPrefix.Addr().AsSlice())

	processed := make(map[string]bool)

	for _, name := range zoneNames {
		if _, ok := processed[name]; ok {
			return nil, errors.Errorf("zone name %v is duplicated", name)
		}
		processed[name] = true

		zoneWorkerIP, _ := netip.AddrFromSlice(zoneIPValue.Bytes())
		zoneWorkerCidr := netip.PrefixFrom(zoneWorkerIP, workerNetworkPrefixLength)

		if !cidr.Contains(zoneWorkerCidr.Addr()) {
			return nil, errors.Errorf("calculated subnet CIDR %s is not contained in main worker CIDR %s", zoneWorkerCidr.String(), cidr.String())
		}

		zoneIPValue.Add(zoneIPValue, delta)
		zones = append(zones, v1alpha1.Zone{
			Name: fmt.Sprintf("%v", name),
			Workers: zoneWorkerCidr.String(),
		})
	}
	return zones, nil
}