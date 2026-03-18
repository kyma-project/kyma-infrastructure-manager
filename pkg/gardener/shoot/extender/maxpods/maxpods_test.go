/*
Copyright 2023.

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
	"math"
	"testing"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"
)

func TestMaxPodsFromPodsCIDR(t *testing.T) {
	tests := []struct {
		name        string
		podsCIDR    string
		expected    int32
		expectError bool
	}{
		{
			name:     "mask 24 returns 254 (256 minus network and broadcast)",
			podsCIDR: "100.64.0.0/24",
			expected: 254,
		},
		{
			name:     "mask 26 returns 62",
			podsCIDR: "100.64.0.0/26",
			expected: 62,
		},
		{
			name:     "mask 16 returns 65534",
			podsCIDR: "100.64.0.0/16",
			expected: 65534,
		},
		{
			name:     "mask 28 returns 14",
			podsCIDR: "10.96.0.0/28",
			expected: 14,
		},
		{
			name:     "mask 31 returns 2 (point-to-point, RFC 3021)",
			podsCIDR: "10.0.0.0/31",
			expected: 2,
		},
		{
			name:     "mask 32 returns 1 (host route)",
			podsCIDR: "10.0.0.1/32",
			expected: 1,
		},
		{
			name:     "mask 2 returns 1073741822 (2^30 minus network and broadcast)",
			podsCIDR: "0.0.0.0/2",
			expected: 1073741822,
		},
		{
			name:        "invalid CIDR",
			podsCIDR:    "invalid",
			expectError: true,
		},
		{
			name:     "IPv6 mask 128 returns 1 (host route)",
			podsCIDR: "2001:db8::1/128",
			expected: 1,
		},
		{
			name:     "IPv6 mask 127 returns 2 (point-to-point, RFC 3021)",
			podsCIDR: "2001:db8::/127",
			expected: 2,
		},
		{
			name:     "IPv6 mask 120 returns 254 (256 minus network and broadcast)",
			podsCIDR: "2001:db8::/120",
			expected: 254,
		},
		{
			name:     "IPv6 mask 32 returns capped at MaxInt32",
			podsCIDR: "2001:db8::/32",
			expected: math.MaxInt32,
		},
		{
			name:     "IPv6 mask 65 returns capped at MaxInt32 (exercises non-overflow path)",
			podsCIDR: "2001:db8::/65",
			expected: math.MaxInt32,
		},
		{
			name:     "IPv6 mask 64 returns capped at MaxInt32",
			podsCIDR: "2001:db8::/64",
			expected: math.MaxInt32,
		},
		{
			name:        "IPv6 mask 1 invalid",
			podsCIDR:    "2001:db8::/1",
			expectError: true,
		},
		{
			name:        "mask 0 too large for pod CIDR",
			podsCIDR:    "0.0.0.0/0",
			expectError: true,
		},
		{
			name:        "mask 1 too large for pod CIDR",
			podsCIDR:    "128.0.0.0/1",
			expectError: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MaxPodsFromPodsCIDR(tt.podsCIDR)
			if tt.expectError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestApplyMaxPodsWithTotalCap(t *testing.T) {
	tests := []struct {
		name        string
		workers     []gardener.Worker
		totalIPs    int32
		expectedMax []*int32
		expectError bool
	}{
		{
			name: "nil maxPods unchanged",
			workers: []gardener.Worker{
				{Kubernetes: nil},
				{Kubernetes: &gardener.WorkerKubernetes{Kubelet: nil}},
				{Kubernetes: &gardener.WorkerKubernetes{Kubelet: &gardener.KubeletConfig{MaxPods: nil}}},
			},
			totalIPs:    256,
			expectedMax: []*int32{nil, nil, nil},
		},
		{
			name: "sum within totalIPs unchanged",
			workers: []gardener.Worker{
				{
					Kubernetes: &gardener.WorkerKubernetes{
						Kubelet: &gardener.KubeletConfig{MaxPods: ptr.To(int32(100))},
					},
				},
				{
					Kubernetes: &gardener.WorkerKubernetes{
						Kubelet: &gardener.KubeletConfig{MaxPods: ptr.To(int32(125))},
					},
				},
			},
			totalIPs:    256,
			expectedMax: []*int32{ptr.To(int32(100)), ptr.To(int32(125))},
		},
		{
			name: "sum exceeds totalIPs clamp last worker",
			workers: []gardener.Worker{
				{
					Kubernetes: &gardener.WorkerKubernetes{
						Kubelet: &gardener.KubeletConfig{MaxPods: ptr.To(int32(100))},
					},
				},
				{
					Kubernetes: &gardener.WorkerKubernetes{
						Kubelet: &gardener.KubeletConfig{MaxPods: ptr.To(int32(200))},
					},
				},
			},
			totalIPs:    256,
			expectedMax: []*int32{ptr.To(int32(100)), ptr.To(int32(156))},
		},
		{
			name: "single worker exceeds totalIPs clamped to totalIPs",
			workers: []gardener.Worker{
				{
					Kubernetes: &gardener.WorkerKubernetes{
						Kubelet: &gardener.KubeletConfig{MaxPods: ptr.To(int32(500))},
					},
				},
			},
			totalIPs:    254, // matches MaxPodsFromPodsCIDR("x.x.x.x/24")
			expectedMax: []*int32{ptr.To(int32(254))},
		},
		{
			name: "multiple workers clamp from last backward",
			workers: []gardener.Worker{
				{
					Kubernetes: &gardener.WorkerKubernetes{
						Kubelet: &gardener.KubeletConfig{MaxPods: ptr.To(int32(100))},
					},
				},
				{
					Kubernetes: &gardener.WorkerKubernetes{
						Kubelet: &gardener.KubeletConfig{MaxPods: ptr.To(int32(50))},
					},
				},
				{
					Kubernetes: &gardener.WorkerKubernetes{
						Kubelet: &gardener.KubeletConfig{MaxPods: ptr.To(int32(100))},
					},
				},
			},
			totalIPs:    200,
			expectedMax: []*int32{ptr.To(int32(100)), ptr.To(int32(50)), ptr.To(int32(50))},
		},
		{
			name: "workers with nil maxPods skipped",
			workers: []gardener.Worker{
				{Kubernetes: nil},
				{
					Kubernetes: &gardener.WorkerKubernetes{
						Kubelet: &gardener.KubeletConfig{MaxPods: ptr.To(int32(256))},
					},
				},
			},
			totalIPs:    256,
			expectedMax: []*int32{nil, ptr.To(int32(256))},
		},
		{
			name: "totalIPs less than 1 returns error",
			workers: []gardener.Worker{
				{
					Kubernetes: &gardener.WorkerKubernetes{
						Kubelet: &gardener.KubeletConfig{MaxPods: ptr.To(int32(10))},
					},
				},
			},
			totalIPs:    0,
			expectedMax: []*int32{ptr.To(int32(10))},
			expectError: true,
		},
		{
			name: "constraint cannot be satisfied returns error",
			workers: []gardener.Worker{
				{
					Kubernetes: &gardener.WorkerKubernetes{
						Kubelet: &gardener.KubeletConfig{MaxPods: ptr.To(int32(1))},
					},
				},
				{
					Kubernetes: &gardener.WorkerKubernetes{
						Kubelet: &gardener.KubeletConfig{MaxPods: ptr.To(int32(1))},
					},
				},
			},
			totalIPs:    1,
			expectedMax: []*int32{ptr.To(int32(1)), ptr.To(int32(1))},
			expectError: true,
		},
		{
			name: "sum overflow with true int64 excess clamps both workers",
			workers: []gardener.Worker{
				{
					Kubernetes: &gardener.WorkerKubernetes{
						Kubelet: &gardener.KubeletConfig{MaxPods: ptr.To(int32(math.MaxInt32))},
					},
				},
				{
					Kubernetes: &gardener.WorkerKubernetes{
						Kubelet: &gardener.KubeletConfig{MaxPods: ptr.To(int32(math.MaxInt32))},
					},
				},
			},
			totalIPs:    100,
			expectedMax: []*int32{ptr.To(int32(99)), ptr.To(int32(1))},
		},
		{
			name: "worker with maxPods 1 unchanged during clamp",
			workers: []gardener.Worker{
				{
					Kubernetes: &gardener.WorkerKubernetes{
						Kubelet: &gardener.KubeletConfig{MaxPods: ptr.To(int32(100))},
					},
				},
				{
					Kubernetes: &gardener.WorkerKubernetes{
						Kubelet: &gardener.KubeletConfig{MaxPods: ptr.To(int32(1))},
					},
				},
			},
			totalIPs:    50,
			expectedMax: []*int32{ptr.To(int32(49)), ptr.To(int32(1))},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ApplyMaxPodsWithTotalCap(tt.workers, tt.totalIPs)
			if tt.expectError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			for i, w := range tt.workers {
				var got *int32
				if w.Kubernetes != nil && w.Kubernetes.Kubelet != nil {
					got = w.Kubernetes.Kubelet.MaxPods
				}
				if tt.expectedMax[i] == nil {
					assert.Nil(t, got)
				} else {
					require.NotNil(t, got)
					assert.Equal(t, *tt.expectedMax[i], *got)
				}
			}
		})
	}
}
