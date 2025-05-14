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
	minNumberOfZones = 1
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
	if numZones < minNumberOfZones || numZones > maxNumberOfZones {
		return nil, errors.New("Number of networking zones must be between 1 and 8")
	}

	var zones []v1alpha1.Zone

	cidr, err := netip.ParsePrefix(workerCidr)
	if err != nil {
		return zones, errors.Wrap(err, "failed to parse worker CIDR")
	}

	orgWorkerPrefixLength := cidr.Bits() + workersBits
	workerPrefix, err := cidr.Addr().Prefix(orgWorkerPrefixLength)
	if err != nil {
		return zones, errors.Wrap(err, "failed to get worker prefix")
	}

	// delta - it is the difference between "public" and "internal" CIDRs, for example:
	//    WorkerCidr:   "10.250.0.0/19",
	//    PublicCidr:   "10.250.32.0/20",
	//    InternalCidr: "10.250.48.0/20",
	// 4 * delta  - difference between two worker (zone) CIDRs 4096 hosts
	// small_delta and smallPrefixLength are used for subnets created for last 5 (from 4th to 8th) zones
	// small_delta  - difference between two subnetworks CIDRs for last 5 (from 4th to 8th) zones 1024 hosts
	//    WorkerCidr:   "10.250.192.0/22",
	//    PublicCidr:   "10.250.196.0/22",
	//    InternalCidr: "10.250.200.0/22",

	smallPrefixLength := orgWorkerPrefixLength + 3

	delta := big.NewInt(1)
	delta.Lsh(delta, uint(lastBitNumber-orgWorkerPrefixLength))
	small_delta := big.NewInt(1)
	small_delta.Rsh(delta, 2)

	// base - it is an integer, which is based on IP bytes
	base := new(big.Int).SetBytes(workerPrefix.Addr().AsSlice())

	processed := make(map[string]bool)

	for i, name := range zoneNames {
		if _, ok := processed[name]; ok {
			return nil, errors.Errorf("zone name %s is duplicated", name)
		}
		processed[name] = true

		var workPrefixLength, publicPrefixLength, internalPrefixLength int
		var deltaStep *big.Int

		if i < 3 {
			// first 3 zones are using bigger subnet size 4096 hosts
			workPrefixLength = orgWorkerPrefixLength
			publicPrefixLength = orgWorkerPrefixLength + 1
			internalPrefixLength = orgWorkerPrefixLength + 1
			deltaStep = delta
		} else {
			// last 5 zones are using smaller subnet size 1024 hosts
			workPrefixLength = smallPrefixLength
			publicPrefixLength = smallPrefixLength
			internalPrefixLength = smallPrefixLength
			deltaStep = small_delta
		}

		zoneWorkerIP, _ := netip.AddrFromSlice(base.Bytes())
		zoneWorkerCidr := netip.PrefixFrom(zoneWorkerIP, workPrefixLength)
		base.Add(base, deltaStep)

		if i < 3 {
			base.Add(base, deltaStep)
		}

		publicIP, _ := netip.AddrFromSlice(base.Bytes())
		public := netip.PrefixFrom(publicIP, publicPrefixLength)
		base.Add(base, deltaStep)

		internalIP, _ := netip.AddrFromSlice(base.Bytes())
		internalPrefix := netip.PrefixFrom(internalIP, internalPrefixLength)
		base.Add(base, deltaStep)

		zones = append(zones, v1alpha1.Zone{
			Name:     name,
			Workers:  zoneWorkerCidr.String(),
			Public:   public.String(),
			Internal: internalPrefix.String(),
		})
	}

	return zones, nil
}
