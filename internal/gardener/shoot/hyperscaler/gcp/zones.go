package gcp

import (
	"fmt"
	"math/rand"
)

func ZonesForGCPRegion(region string, zonesCount int) []string {
	zoneCodes := []string{"a", "b", "c"}
	var zones []string
	rand.Shuffle(len(zoneCodes), func(i, j int) { zoneCodes[i], zoneCodes[j] = zoneCodes[j], zoneCodes[i] })

	for i := 0; i < zonesCount && i < len(zoneCodes); i++ {
		zones = append(zones, fmt.Sprintf("%s-%s", region, zoneCodes[i]))
	}

	return zones
}
