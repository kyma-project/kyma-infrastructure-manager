package aws

import (
	"github.com/pkg/errors"
	"math/big"
	"net/netip"

	"github.com/gardener/gardener-extension-provider-aws/pkg/apis/aws/v1alpha1"
)

const (
	workersBits      = 3
	lastBitNumber    = 31
	maxNumberOfZones = 8
)

/*
*
generateAWSZones - creates a list of AWSZoneInput objects which contains a proper IP ranges.
It generates subnets - the subnets in AZ must be inside of the cidr block and non overlapping. example values:
cidr: 10.250.0.0/16
  - name: eu-central-1a
    workers: 10.250.0.0/19
    public: 10.250.32.0/20
    internal: 10.250.48.0/20
  - name: eu-central-1b
    workers: 10.250.64.0/19
    public: 10.250.96.0/20
    internal: 10.250.112.0/20
  - name: eu-central-1c
    workers: 10.250.128.0/19
    public: 10.250.160.0/20
    internal: 10.250.176.0/20
*/
func generateAWSZones(workerCidr string, zoneNames []string) ([]v1alpha1.Zone, error) {

	numZones := len(zoneNames)
	if numZones < 1 || numZones > maxNumberOfZones {
		return nil, errors.New("Number of networking zones must be between 1 and 8")
	}

	var zones []v1alpha1.Zone

	cidr, _ := netip.ParsePrefix(workerCidr)
	workerPrefixLength := cidr.Bits() + workersBits
	workerPrefix, _ := cidr.Addr().Prefix(workerPrefixLength)

	// delta - it is the difference between "public" and "internal" CIDRs, for example:
	//    WorkerCidr:   "10.250.0.0/19",
	//    PublicCidr:   "10.250.32.0/20",
	//    InternalCidr: "10.250.48.0/20",
	// 4 * delta  - difference between two worker (zone) CIDRs
	delta := big.NewInt(1)
	delta.Lsh(delta, uint(lastBitNumber-workerPrefixLength))

	// base - it is an integer, which is based on IP bytes
	base := new(big.Int).SetBytes(workerPrefix.Addr().AsSlice())

	for _, name := range zoneNames {
		zoneWorkerIP, _ := netip.AddrFromSlice(base.Bytes())
		zoneWorkerCidr := netip.PrefixFrom(zoneWorkerIP, workerPrefixLength)

		base.Add(base, delta)
		base.Add(base, delta)
		publicIP, _ := netip.AddrFromSlice(base.Bytes())
		public := netip.PrefixFrom(publicIP, workerPrefixLength+1)

		base.Add(base, delta)
		internalIP, _ := netip.AddrFromSlice(base.Bytes())
		internalPrefix := netip.PrefixFrom(internalIP, workerPrefixLength+1)

		zones = append(zones, v1alpha1.Zone{
			Name:     name,
			Workers:  zoneWorkerCidr.String(),
			Public:   public.String(),
			Internal: internalPrefix.String(),
		})

		base.Add(base, delta)
	}

	return zones, nil
}
