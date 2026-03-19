/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package maxpods

import (
	"fmt"
	"math"
	"net/netip"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/pkg/errors"
	"k8s.io/utils/ptr"
)

// MaxPodsFromPodsCIDR parses the pods CIDR (IPv4 only, e.g. "100.64.0.0/24") and returns
// the number of usable pod IPs, accounting for reserved network and broadcast addresses.
// IPv4: /32: 1 usable (host route). /31: 2 usable (point-to-point, RFC 3021).
// /30 and larger: 2^(32-mask) - 2 (network + broadcast reserved).
// Results exceeding math.MaxInt32 are capped at math.MaxInt32.
// Returns error if the CIDR is invalid, the mask is out of range (2-32), or the CIDR is IPv6.
func MaxPodsFromPodsCIDR(podsCIDR string) (int64, error) {
	prefix, err := netip.ParsePrefix(podsCIDR)
	if err != nil {
		return 0, errors.Wrap(err, "failed to parse pods CIDR")
	}

	bits := prefix.Bits()

	if prefix.Addr().Is4() {
		// Valid range [2, 32]: bits < 2 rejects overly large CIDRs (/0, /1) that are impractical for pod allocation; bits > 32 invalid for IPv4
		if bits < 2 || bits > 32 {
			return 0, fmt.Errorf("pods CIDR mask must be between 2 and 32 for IPv4, got %d", bits)
		}
		switch bits {
		case 32:
			return 1, nil // host route
		case 31:
			return 2, nil // point-to-point, both addresses usable (RFC 3021)
		default:
			count := uint64(1<<uint(32-bits)) - 2
			if count > math.MaxInt32 {
				return math.MaxInt32, nil
			}
			return int64(count), nil // network and broadcast addresses reserved
		}
	}

	return 0, fmt.Errorf("maxPods calculation supports IPv4 only, got IPv6 CIDR")
}

// ApplyMaxPodsWithTotalCap ensures sum(worker maxPods) <= totalIPs.
// Workers keep their declared maxPods unless the sum exceeds totalIPs.
// When sum > totalIPs, clamps from the last worker backward until sum <= totalIPs.
// Workers without maxPods set are skipped (they use Kubernetes default at runtime).
// The guarantee sum(maxPods) <= totalIPs only holds when all workers have maxPods explicitly set.
// totalIPs must be at least 512.
// Returns error if totalIPs < 512 or if the constraint cannot be satisfied (e.g. more workers with maxPods=1 than totalIPs).
func ApplyMaxPodsWithTotalCap(workers []gardener.Worker, totalIPs int64) error {
	if totalIPs < 512 {
		return fmt.Errorf("totalIPs must be at least 512, got %d", totalIPs)
	}
	currentSum, indicesWithMaxPods := collectMaxPodsIndices(workers)
	if currentSum <= totalIPs {
		return nil
	}
	excess := currentSum - totalIPs
	for i := len(indicesWithMaxPods) - 1; i >= 0 && excess > 0; i-- {
		worker := &workers[indicesWithMaxPods[i]]
		reduction := reduceWorkerMaxPods(worker, excess)
		excess -= reduction
	}
	if excess > 0 {
		return fmt.Errorf("cannot satisfy maxPods constraint: %d workers with maxPods set (minimum 1 each) exceed totalIPs %d", len(indicesWithMaxPods), totalIPs)
	}
	return nil
}

// collectMaxPodsIndices returns the sum of maxPods across workers and the indices of workers that have maxPods set.
// Sum is accumulated in int64 to avoid overflow when multiple workers have large maxPods.
func collectMaxPodsIndices(workers []gardener.Worker) (int64, []int) {
	var sum int64
	indices := make([]int, 0, len(workers))
	for i := range workers {
		worker := &workers[i]
		if worker.Kubernetes != nil && worker.Kubernetes.Kubelet != nil && worker.Kubernetes.Kubelet.MaxPods != nil {
			sum += int64(*worker.Kubernetes.Kubelet.MaxPods)
			indices = append(indices, i)
		}
	}
	return sum, indices
}

// reduceWorkerMaxPods reduces the worker's maxPods by up to excess, but not below 1.
// Returns the actual amount reduced. When current is 1 or less (invalid), returns 0 without modifying.
func reduceWorkerMaxPods(w *gardener.Worker, excess int64) int64 {
	current := *w.Kubernetes.Kubelet.MaxPods
	if current <= 1 {
		return 0
	}
	reduction := min(excess, int64(current-1))
	w.Kubernetes.Kubelet.MaxPods = ptr.To(current - int32(reduction))
	return reduction
}
