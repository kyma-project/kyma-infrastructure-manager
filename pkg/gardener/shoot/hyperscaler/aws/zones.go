package aws

import (
	"github.com/pkg/errors"
	"math/big"
	"net/netip"

	"github.com/gardener/gardener-extension-provider-aws/pkg/apis/aws/v1alpha1"
)

const (
	subNetworkBitsSize   = 3
	lastBitNumber        = 31
	maxNumberOfZones     = 8
	minNumberOfZones     = 1
	maxPrefixSize        = 24
	minPrefixSize        = 16
	addSmallSubnetPrefix = 3
	kymaWorkerPoolAZs    = 3 //first 3 zones are reserved for Kyma worker pool and bigger dimensioned than later added AZs
)

/*

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

As we use 3 bits for the workers, we have max 8 AZ available, each one allocating 3 non-overlapping subnets (public, private and internal).
The first 3 AZs are using the bigger subnet size for kyma Worker pool (4096 hosts).
The last 5 subnets are using the smaller subnet size (1024 hosts).
*/

func generateAWSZones(workerCidr string, zoneNames []string) ([]v1alpha1.Zone, error) {
	numZones := len(zoneNames)
	if numZones < minNumberOfZones || numZones > maxNumberOfZones {
		return nil, errors.New("Number of networking zones must be between 1 and 8")
	}

	var zones []v1alpha1.Zone

	cidr, err := netip.ParsePrefix(workerCidr)
	if err != nil {
		return zones, errors.Wrap(err, "failed to parse worker network CIDR")
	}

	// CIDR prefix length ex from "10.250.0.0/16" is 16
	prefixLength := cidr.Bits()

	if prefixLength > maxPrefixSize || prefixLength < minPrefixSize {
		return nil, errors.New("CIDR prefix length must be between 16 and 24")
	}

	kymaWorkerNetworkPrefixLength := prefixLength + subNetworkBitsSize
	workerPrefix, err := cidr.Addr().Prefix(kymaWorkerNetworkPrefixLength)
	if err != nil {
		return zones, errors.Wrap(err, "failed to get worker prefix")
	}
	// base - it is an integer, which is based on IP bytes
	base := new(big.Int).SetBytes(workerPrefix.Addr().AsSlice())

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

	kymaWorkerNetworkDelta := big.NewInt(1)
	kymaWorkerNetworkDelta.Lsh(kymaWorkerNetworkDelta, uint(lastBitNumber-kymaWorkerNetworkPrefixLength))

	// initialize for additional Networks for last 5 zones
	// additionalWorkerNetworkPrefixLength - prefix length for additional networks for last 5 zones,
	// addSmallSubnetPrefix is used to calculate prefix length for additional networks as small subnet
	// additionalWorkerNetworkDelta - difference between two subnetworks CIDRs for last 5 zones
	additionalWorkerNetworkPrefixLength := kymaWorkerNetworkPrefixLength + addSmallSubnetPrefix
	additionalWorkerNetworkDelta := big.NewInt(1)
	additionalWorkerNetworkDelta.Rsh(kymaWorkerNetworkDelta, 2)

	processed := make(map[string]bool)

	for i, name := range zoneNames {
		if _, ok := processed[name]; ok {
			return nil, errors.Errorf("zone name %s is duplicated", name)
		}
		processed[name] = true

		var workPrefixLength, publicPrefixLength, internalPrefixLength int
		var deltaStep *big.Int

		if i < kymaWorkerPoolAZs {
			// first 3 zones are using bigger subnet sizes.
			workPrefixLength = kymaWorkerNetworkPrefixLength
			publicPrefixLength = kymaWorkerNetworkPrefixLength + 1
			internalPrefixLength = kymaWorkerNetworkPrefixLength + 1
			deltaStep = kymaWorkerNetworkDelta
		} else {
			// last 5 zones are using smaller subnet sizes
			workPrefixLength = additionalWorkerNetworkPrefixLength
			publicPrefixLength = additionalWorkerNetworkPrefixLength
			internalPrefixLength = additionalWorkerNetworkPrefixLength
			deltaStep = additionalWorkerNetworkDelta
		}

		zoneWorkerCIDR, err := getCIDRFromInt(base, workPrefixLength, cidr)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get worker CIDR for zone %s", name)
		}

		base.Add(base, deltaStep)

		if i < kymaWorkerPoolAZs { // additional step for the first 3 AZs
			base.Add(base, deltaStep)
		}

		zonePublicCIDR, err := getCIDRFromInt(base, publicPrefixLength, cidr)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get public CIDR for zone %s", name)
		}

		base.Add(base, deltaStep)

		zoneInternalCIDR, err := getCIDRFromInt(base, internalPrefixLength, cidr)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get internal CIDR for zone %s", name)
		}

		base.Add(base, deltaStep)

		zones = append(zones, v1alpha1.Zone{
			Name:     name,
			Workers:  zoneWorkerCIDR.String(),
			Public:   zonePublicCIDR.String(),
			Internal: zoneInternalCIDR.String(),
		})
	}

	return zones, nil
}

func getCIDRFromInt(base *big.Int, prefixLength int, mainCIDR netip.Prefix) (netip.Prefix, error) {
	addr, _ := netip.AddrFromSlice(base.Bytes())
	resultCIDR := netip.PrefixFrom(addr, prefixLength)

	if !mainCIDR.Contains(resultCIDR.Addr()) {
		return netip.Prefix{}, errors.Errorf("calculated subnet CIDR %s is not contained in main worker CIDR %s", resultCIDR.String(), mainCIDR.String())
	}

	return resultCIDR, nil
}
