package aws

import (
	"math/big"
	"net/netip"
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
func generateAWSZones(workerCidr string, zoneNames []string) []Zone {
	var zones []Zone

	cidr, _ := netip.ParsePrefix(workerCidr)
	workerPrefixLength := cidr.Bits() + 3
	workerPrefix, _ := cidr.Addr().Prefix(workerPrefixLength)

	// delta - it is the difference between "public" and "internal" CIDRs, for example:
	//    WorkerCidr:   "10.250.0.0/19",
	//    PublicCidr:   "10.250.32.0/20",
	//    InternalCidr: "10.250.48.0/20",
	// 4 * delta  - difference between two worker (zone) CIDRs
	delta := big.NewInt(1)
	delta.Lsh(delta, uint(31-workerPrefixLength))

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

		zones = append(zones, Zone{
			Name:     name,
			Workers:  zoneWorkerCidr.String(),
			Public:   public.String(),
			Internal: internalPrefix.String(),
		})

		base.Add(base, delta)
	}

	return zones
}
