package provider

import (
	awsinfra "github.com/gardener/gardener-extension-provider-aws/pkg/apis/aws/v1alpha1"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/extender/testutils"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/yaml"

	"testing"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/hyperscaler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidations(t *testing.T) {
	t.Run("Return error for unknown provider", func(t *testing.T) {
		// given
		shoot := testutils.FixEmptyGardenerShoot("cluster", "kcp-system")
		rt := imv1.Runtime{
			Spec: imv1.RuntimeSpec{
				Shoot: imv1.RuntimeShoot{
					Provider: imv1.Provider{
						Type: "unknown",
					},
				},
			},
		}

		// when
		extender := NewProviderExtenderForCreateOperation(false, "", "")
		err := extender(rt, &shoot)

		// then
		require.Error(t, err)
	})
}

func TestFixAlignWorkerZonesWithGardener(t *testing.T) {
	t.Run("The single node worker pool specified in the Runtime CR refers to zone that is different on the existing shoot", func(t *testing.T) {
		// given
		shoot := testutils.FixEmptyGardenerShoot("cluster", "kcp-system")

		runtime := imv1.Runtime{
			Spec: imv1.RuntimeSpec{
				Shoot: imv1.RuntimeShoot{
					Provider: fixProviderWithMultipleWorkersAndConfig(hyperscaler.TypeAWS, fixMultipleWorkers([]workerConfig{
						{"main-worker", "m6i.large", "gardenlinux", "1310.4.0", 1, 3, []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}},
						{"additional", "m7i.large", "gardenlinux", "1311.2.0", 2, 4, []string{"eu-central-1a"}},
					}), fixAWSInfrastructureConfig(t, "10.250.0.0/22", []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}), fixAWSControlPlaneConfig()),
					Networking: imv1.Networking{
						Nodes: "10.250.0.0/22",
					},
				},
			},
		}

		currentWorkers := fixMultipleWorkers([]workerConfig{
			{"main-worker", "m6i.large", "gardenlinux", "1310.4.0", 1, 3, []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}},
			{"additional", "m7i.large", "gardenlinux", "1311.2.0", 2, 4, []string{"eu-central-1b"}},
		})

		// when
		extender := NewProviderExtenderPatchOperation(false, "gardenlinux", "1311.2.0", currentWorkers, fixAWSInfrastructureConfig(t, "10.250.0.0/22", []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}), fixAWSControlPlaneConfig())
		err := extender(runtime, &shoot)

		// then
		require.NoError(t, err)

		assert.Equal(t, []string{"eu-central-1b", "eu-central-1a"}, shoot.Spec.Provider.Workers[1].Zones)
	})
}

func TestGetAllWorkersZones(t *testing.T) {
	tests := []struct {
		name     string
		workers  []gardener.Worker
		expected []string
	}{
		{
			name: "Single worker with zones",
			workers: []gardener.Worker{
				{
					Name:  "worker1",
					Zones: []string{"zone1", "zone2"},
				},
			},
			expected: []string{"zone1", "zone2"},
		},
		{
			name: "Multiple workers with same zones",
			workers: []gardener.Worker{
				{
					Name:  "worker1",
					Zones: []string{"zone1", "zone2"},
				},
				{
					Name:  "worker2",
					Zones: []string{"zone1", "zone2"},
				},
			},
			expected: []string{"zone1", "zone2"},
		},
		{
			name: "Multiple workers with different zones",
			workers: []gardener.Worker{
				{
					Name:  "worker1",
					Zones: []string{"zone1", "zone2"},
				},
				{
					Name:  "worker2",
					Zones: []string{"zone1"},
				},
			},
			expected: []string{"zone1", "zone2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			zones := getNetworkingZonesFromWorkers(tt.workers)
			assert.Equal(t, tt.expected, zones)
		})
	}
}

type workerConfig struct {
	Name                string
	MachineType         string
	MachineImageName    string
	MachineImageVersion string
	hypescalerMin       int32
	hyperscalerMax      int32
	Zones               []string
}

func TestProviderExtenderForCreateMultipleWorkersAWS(t *testing.T) {
	// tests of NewProviderExtenderForCreateOperation for workers create operation
	for tname, tc := range map[string]struct {
		Runtime                    imv1.Runtime
		EnableIMDSv2               bool
		DefaultMachineImageVersion string
		DefaultMachineImageName    string
		CurrentShootWorkers        []gardener.Worker
		ExistingInfraConfig        *runtime.RawExtension
		ExistingControlPlaneConfig *runtime.RawExtension
		ExpectedShootWorkers       []gardener.Worker
		ExpectedZonesCount         int
	}{
		"Create provider config for multiple workers without AWS worker config": {
			Runtime: imv1.Runtime{
				Spec: imv1.RuntimeSpec{
					Shoot: imv1.RuntimeShoot{
						Provider: fixProviderWithMultipleWorkers(hyperscaler.TypeAWS, fixMultipleWorkers([]workerConfig{
							{"main-worker", "m6i.large", "gardenlinux", "1310.4.0", 1, 3, []string{"eu-central-1a"}},
							{"additional", "m7i.large", "gardenlinux", "1311.2.0", 2, 4, []string{"eu-central-1b", "eu-central-1c"}},
							{"another", "m8i.large", "gardenlinux", "1312.2.0", 3, 5, []string{"eu-central-1c"}},
						})),
						Networking: imv1.Networking{
							Nodes: "10.250.0.0/22",
						},
					},
				},
			},
			EnableIMDSv2:               false,
			DefaultMachineImageName:    "gardenlinux",
			DefaultMachineImageVersion: "1312.3.0",
			ExpectedZonesCount:         3,
			ExpectedShootWorkers: fixMultipleWorkers([]workerConfig{
				{"main-worker", "m6i.large", "gardenlinux", "1310.4.0", 1, 3, []string{"eu-central-1a"}},
				{"additional", "m7i.large", "gardenlinux", "1311.2.0", 2, 4, []string{"eu-central-1b", "eu-central-1c"}},
				{"another", "m8i.large", "gardenlinux", "1312.2.0", 3, 5, []string{"eu-central-1c"}}}),
		},
		"Create provider config for multiple workers with AWS worker config": {
			Runtime: imv1.Runtime{
				Spec: imv1.RuntimeSpec{
					Shoot: imv1.RuntimeShoot{
						Provider: fixProviderWithMultipleWorkers(hyperscaler.TypeAWS, fixMultipleWorkers([]workerConfig{
							{"main-worker", "m6i.large", "gardenlinux", "1310.4.0", 1, 3, []string{"eu-central-1a"}},
							{"additional", "m7i.large", "gardenlinux", "1311.2.0", 2, 4, []string{"eu-central-1b", "eu-central-1c"}},
							{"another", "m8i.large", "gardenlinux", "1312.2.0", 3, 5, []string{"eu-central-1c"}},
						})),
						Networking: imv1.Networking{
							Nodes: "10.250.0.0/22",
						},
					},
				},
			},
			EnableIMDSv2:               true,
			DefaultMachineImageName:    "gardenlinux",
			DefaultMachineImageVersion: "1312.3.0",
			ExpectedZonesCount:         3,
			ExpectedShootWorkers: fixMultipleWorkers([]workerConfig{
				{"main-worker", "m6i.large", "gardenlinux", "1310.4.0", 1, 3, []string{"eu-central-1a"}},
				{"additional", "m7i.large", "gardenlinux", "1311.2.0", 2, 4, []string{"eu-central-1b", "eu-central-1c"}},
				{"another", "m8i.large", "gardenlinux", "1312.2.0", 3, 5, []string{"eu-central-1c"}}}),
		},
	} {
		t.Run(tname, func(t *testing.T) {
			// given
			shoot := testutils.FixEmptyGardenerShoot("cluster", "kcp-system")

			// when
			extender := NewProviderExtenderForCreateOperation(tc.EnableIMDSv2, tc.DefaultMachineImageName, tc.DefaultMachineImageVersion)
			err := extender(tc.Runtime, &shoot)

			// then
			require.NoError(t, err)

			assertProviderMultipleWorkers(t, tc.Runtime.Spec.Shoot, shoot, tc.EnableIMDSv2, tc.ExpectedShootWorkers)
			assertProviderSpecificConfigAWS(t, shoot, tc.ExpectedZonesCount)
		})
	}
}

func TestProviderExtenderForPatchWorkersUpdateAWS(t *testing.T) {
	// tests of NewProviderExtenderPatch for workers update operation
	for tname, tc := range map[string]struct {
		Runtime                    imv1.Runtime
		EnableIMDSv2               bool
		DefaultMachineImageVersion string
		DefaultMachineImageName    string
		CurrentShootWorkers        []gardener.Worker
		ExistingInfraConfig        *runtime.RawExtension
		ExistingControlPlaneConfig *runtime.RawExtension
		ExpectedShootWorkers       []gardener.Worker
		ExpectedZonesCount         int
	}{
		"Add additional worker": {
			Runtime: imv1.Runtime{
				Spec: imv1.RuntimeSpec{
					Shoot: imv1.RuntimeShoot{
						Provider: fixProviderWithMultipleWorkers(hyperscaler.TypeAWS, fixMultipleWorkers([]workerConfig{
							{"main-worker", "m6i.large", "gardenlinux", "1312.4.0", 1, 3, []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}},
							{"additional", "m6i.large", "gardenlinux", "1312.2.0", 1, 3, []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}},
						})),
						Networking: imv1.Networking{
							Nodes: "10.250.0.0/22",
						},
					},
				},
			},
			EnableIMDSv2:               false,
			DefaultMachineImageName:    "gardenlinux",
			DefaultMachineImageVersion: "1312.3.0",
			ExpectedZonesCount:         3,
			CurrentShootWorkers:        fixWorkers("main-worker", "m6i.large", "gardenlinux", "1312.4.0", 1, 3, []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}),
			ExpectedShootWorkers: fixMultipleWorkers([]workerConfig{
				{"main-worker", "m6i.large", "gardenlinux", "1312.4.0", 1, 3, []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}},
				{"additional", "m6i.large", "gardenlinux", "1312.2.0", 1, 3, []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}}}),
			ExistingInfraConfig:        fixAWSInfrastructureConfig(t, "10.250.0.0/22", []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}),
			ExistingControlPlaneConfig: fixAWSControlPlaneConfig(),
		},
		"Add additional worker with new zone": {
			Runtime: imv1.Runtime{
				Spec: imv1.RuntimeSpec{
					Shoot: imv1.RuntimeShoot{
						Provider: fixProviderWithMultipleWorkers(hyperscaler.TypeAWS, fixMultipleWorkers([]workerConfig{
							{"main-worker", "m6i.large", "gardenlinux", "1312.4.0", 1, 3, []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}},
							{"additional", "m6i.large", "gardenlinux", "1312.2.0", 1, 3, []string{"eu-central-1a", "eu-central-1b", "eu-central-1d"}},
						})),
						Networking: imv1.Networking{
							Nodes: "10.250.0.0/22",
						},
					},
				},
			},
			EnableIMDSv2:               false,
			DefaultMachineImageName:    "gardenlinux",
			DefaultMachineImageVersion: "1312.3.0",
			ExpectedZonesCount:         4,
			CurrentShootWorkers:        fixWorkers("main-worker", "m6i.large", "gardenlinux", "1312.4.0", 1, 3, []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}),
			ExpectedShootWorkers: fixMultipleWorkers([]workerConfig{
				{"main-worker", "m6i.large", "gardenlinux", "1312.4.0", 1, 3, []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}},
				{"additional", "m6i.large", "gardenlinux", "1312.2.0", 1, 3, []string{"eu-central-1a", "eu-central-1b", "eu-central-1d"}}}),
			ExistingInfraConfig:        fixAWSInfrastructureConfig(t, "10.250.0.0/22", []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}),
			ExistingControlPlaneConfig: fixAWSControlPlaneConfig(),
		},
		"Edit additional worker - extend existing additional worker from non HA setup to HA setup (use zone already specified in the existing shoot)": {
			Runtime: imv1.Runtime{
				Spec: imv1.RuntimeSpec{
					Shoot: imv1.RuntimeShoot{
						Provider: fixProviderWithMultipleWorkers(hyperscaler.TypeAWS, fixMultipleWorkers([]workerConfig{
							{"main-worker", "m6i.large", "gardenlinux", "1312.4.0", 1, 3, []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}},
							{"additional", "m6i.large", "gardenlinux", "1312.2.0", 1, 3, []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}},
						})),
						Networking: imv1.Networking{
							Nodes: "10.250.0.0/22",
						},
					},
				},
			},
			EnableIMDSv2:               false,
			DefaultMachineImageName:    "gardenlinux",
			DefaultMachineImageVersion: "1312.3.0",
			ExpectedZonesCount:         3,
			CurrentShootWorkers: fixMultipleWorkers([]workerConfig{
				{"main-worker", "m6i.large", "gardenlinux", "1312.4.0", 1, 3, []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}},
				{"additional", "m6i.large", "gardenlinux", "1312.2.0", 1, 3, []string{"eu-central-1a"}}}),
			ExpectedShootWorkers: fixMultipleWorkers([]workerConfig{
				{"main-worker", "m6i.large", "gardenlinux", "1312.4.0", 1, 3, []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}},
				{"additional", "m6i.large", "gardenlinux", "1312.2.0", 1, 3, []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}}}),
			ExistingInfraConfig:        fixAWSInfrastructureConfig(t, "10.250.0.0/22", []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}),
			ExistingControlPlaneConfig: fixAWSControlPlaneConfig(),
		},
		"Edit additional worker - extend existing additional worker from non HA setup to HA setup (use zone not specified in the existing shoot)": {
			Runtime: imv1.Runtime{
				Spec: imv1.RuntimeSpec{
					Shoot: imv1.RuntimeShoot{
						Provider: fixProviderWithMultipleWorkers(hyperscaler.TypeAWS, fixMultipleWorkers([]workerConfig{
							{"main-worker", "m6i.large", "gardenlinux", "1312.4.0", 1, 3, []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}},
							{"additional", "m6i.large", "gardenlinux", "1312.2.0", 1, 3, []string{"eu-central-1a", "eu-central-1b", "eu-central-1d"}},
						})),
						Networking: imv1.Networking{
							Nodes: "10.250.0.0/22",
						},
					},
				},
			},
			EnableIMDSv2:               false,
			DefaultMachineImageName:    "gardenlinux",
			DefaultMachineImageVersion: "1312.3.0",
			ExpectedZonesCount:         4,
			CurrentShootWorkers: fixMultipleWorkers([]workerConfig{
				{"main-worker", "m6i.large", "gardenlinux", "1312.4.0", 1, 3, []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}},
				{"additional", "m6i.large", "gardenlinux", "1312.2.0", 1, 3, []string{"eu-central-1a"}}}),
			ExpectedShootWorkers: fixMultipleWorkers([]workerConfig{
				{"main-worker", "m6i.large", "gardenlinux", "1312.4.0", 1, 3, []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}},
				{"additional", "m6i.large", "gardenlinux", "1312.2.0", 1, 3, []string{"eu-central-1a", "eu-central-1b", "eu-central-1d"}}}),
			ExistingInfraConfig:        fixAWSInfrastructureConfig(t, "10.250.0.0/22", []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}),
			ExistingControlPlaneConfig: fixAWSControlPlaneConfig(),
		},
		"Edit additional worker - extend existing additional worker from non HA setup to HA setup (use zone not specified in the existing shoot, and infrastructure config provided in the Runtime CR)": {
			Runtime: imv1.Runtime{
				Spec: imv1.RuntimeSpec{
					Shoot: imv1.RuntimeShoot{
						Provider: fixProviderWithMultipleWorkersAndConfig(hyperscaler.TypeAWS, fixMultipleWorkers([]workerConfig{
							{"main-worker", "m6i.large", "gardenlinux", "1312.4.0", 1, 3, []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}},
							{"additional", "m6i.large", "gardenlinux", "1312.2.0", 1, 3, []string{"eu-central-1a", "eu-central-1b", "eu-central-1d"}},
						}), fixAWSInfrastructureConfig(t, "10.250.0.0/22", []string{"1", "2", "3"}), fixAWSControlPlaneConfig()),
						Networking: imv1.Networking{
							Nodes: "10.250.0.0/22",
						},
					},
				},
			},
			EnableIMDSv2:               false,
			DefaultMachineImageName:    "gardenlinux",
			DefaultMachineImageVersion: "1312.3.0",
			ExpectedZonesCount:         4,
			CurrentShootWorkers: fixMultipleWorkers([]workerConfig{
				{"main-worker", "m6i.large", "gardenlinux", "1312.4.0", 1, 3, []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}},
				{"additional", "m6i.large", "gardenlinux", "1312.2.0", 1, 3, []string{"eu-central-1a"}}}),
			ExpectedShootWorkers: fixMultipleWorkers([]workerConfig{
				{"main-worker", "m6i.large", "gardenlinux", "1312.4.0", 1, 3, []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}},
				{"additional", "m6i.large", "gardenlinux", "1312.2.0", 1, 3, []string{"eu-central-1a", "eu-central-1b", "eu-central-1d"}}}),
			ExistingInfraConfig:        fixAWSInfrastructureConfig(t, "10.250.0.0/22", []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}),
			ExistingControlPlaneConfig: fixAWSControlPlaneConfig(),
		},
		"Edit additional worker - modify existing additional worker from HA setup to non HA setup (not allowed in BTP, the KIM is supposed to not modify the zones list)": {
			Runtime: imv1.Runtime{
				Spec: imv1.RuntimeSpec{
					Shoot: imv1.RuntimeShoot{
						Provider: fixProviderWithMultipleWorkers(hyperscaler.TypeAWS, fixMultipleWorkers([]workerConfig{
							{"main-worker", "m6i.large", "gardenlinux", "1312.4.0", 1, 3, []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}},
							{"additional", "m6i.large", "gardenlinux", "1312.2.0", 1, 3, []string{"eu-central-1a"}},
						})),
						Networking: imv1.Networking{
							Nodes: "10.250.0.0/22",
						},
					},
				},
			},
			EnableIMDSv2:               false,
			DefaultMachineImageName:    "gardenlinux",
			DefaultMachineImageVersion: "1312.3.0",
			ExpectedZonesCount:         3,
			CurrentShootWorkers: fixMultipleWorkers([]workerConfig{
				{"main-worker", "m6i.large", "gardenlinux", "1312.4.0", 1, 3, []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}},
				{"additional", "m6i.large", "gardenlinux", "1312.2.0", 1, 3, []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}}}),
			ExpectedShootWorkers: fixMultipleWorkers([]workerConfig{
				{"main-worker", "m6i.large", "gardenlinux", "1312.4.0", 1, 3, []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}},
				{"additional", "m6i.large", "gardenlinux", "1312.2.0", 1, 3, []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}}}),
			ExistingInfraConfig:        fixAWSInfrastructureConfig(t, "10.250.0.0/22", []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}),
			ExistingControlPlaneConfig: fixAWSControlPlaneConfig(),
		},
		"Remove additional worker from existing set of workers": {
			Runtime: imv1.Runtime{
				Spec: imv1.RuntimeSpec{
					Shoot: imv1.RuntimeShoot{
						Provider: fixProviderWithMultipleWorkers(hyperscaler.TypeAWS, fixMultipleWorkers([]workerConfig{
							{"main-worker", "m6i.large", "gardenlinux", "1312.4.0", 1, 3, []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}}})),
						Networking: imv1.Networking{
							Nodes: "10.250.0.0/22",
						},
					},
				},
			},
			EnableIMDSv2:               false,
			DefaultMachineImageName:    "gardenlinux",
			DefaultMachineImageVersion: "1312.3.0",
			ExpectedZonesCount:         3,
			CurrentShootWorkers: fixMultipleWorkers([]workerConfig{
				{"main-worker", "m6i.large", "gardenlinux", "1312.4.0", 1, 3, []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}},
				{"additional", "m6i.large", "gardenlinux", "1312.2.0", 1, 3, []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}}}),
			ExpectedShootWorkers: fixMultipleWorkers([]workerConfig{
				{"main-worker", "m6i.large", "gardenlinux", "1312.4.0", 1, 3, []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}}}),
			ExistingInfraConfig:        fixAWSInfrastructureConfig(t, "10.250.0.0/22", []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}),
			ExistingControlPlaneConfig: fixAWSControlPlaneConfig(),
		},
		"Update machine image name and version in multiple workers separately": {
			Runtime: imv1.Runtime{
				Spec: imv1.RuntimeSpec{
					Shoot: imv1.RuntimeShoot{
						Provider: fixProviderWithMultipleWorkers(hyperscaler.TypeAWS, fixMultipleWorkers([]workerConfig{
							{"main-worker", "m6i.large", "gardenlinux", "1313.4.0", 1, 3, []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}},
							{"additional", "m6i.large", "gardenlinux", "1313.2.0", 1, 3, []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}},
						})),
						Networking: imv1.Networking{
							Nodes: "10.250.0.0/22",
						},
					},
				},
			},
			EnableIMDSv2:               false,
			DefaultMachineImageName:    "gardenlinux",
			DefaultMachineImageVersion: "1312.3.0",
			CurrentShootWorkers: fixMultipleWorkers([]workerConfig{
				{"main-worker", "m6i.large", "gardenlinux", "1312.4.0", 1, 3, []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}},
				{"additional", "m6i.large", "gardenlinux", "1312.2.0", 1, 3, []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}}}),
			ExpectedShootWorkers: fixMultipleWorkers([]workerConfig{
				{"main-worker", "m6i.large", "gardenlinux", "1313.4.0", 1, 3, []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}},
				{"additional", "m6i.large", "gardenlinux", "1313.2.0", 1, 3, []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}}}),
			ExpectedZonesCount:         3,
			ExistingInfraConfig:        fixAWSInfrastructureConfig(t, "10.250.0.0/22", []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}),
			ExistingControlPlaneConfig: fixAWSControlPlaneConfig(),
		},
		"Remove worker from existing set of workers networking zones set in infrastructureConfig should not change": {
			Runtime: imv1.Runtime{
				Spec: imv1.RuntimeSpec{
					Shoot: imv1.RuntimeShoot{
						Provider: fixProviderWithMultipleWorkers(hyperscaler.TypeAWS, fixMultipleWorkers([]workerConfig{
							{"main-worker", "m6i.large", "gardenlinux", "1313.4.0", 1, 3, []string{"eu-central-1a"}},
						})),
						Networking: imv1.Networking{
							Nodes: "10.250.0.0/22",
						},
					},
				},
			},
			EnableIMDSv2:               false,
			DefaultMachineImageName:    "gardenlinux",
			DefaultMachineImageVersion: "1312.3.0",
			CurrentShootWorkers: fixMultipleWorkers([]workerConfig{
				{"main-worker", "m6i.large", "gardenlinux", "1312.4.0", 1, 3, []string{"eu-central-1a"}},
				{"additional", "m6i.large", "gardenlinux", "1312.2.0", 1, 3, []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}}}),
			ExpectedShootWorkers: fixMultipleWorkers([]workerConfig{
				{"main-worker", "m6i.large", "gardenlinux", "1313.4.0", 1, 3, []string{"eu-central-1a"}}}),
			ExpectedZonesCount:         3,
			ExistingInfraConfig:        fixAWSInfrastructureConfig(t, "10.250.0.0/22", []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}),
			ExistingControlPlaneConfig: fixAWSControlPlaneConfig(),
		},
		"Edit infrastructure config with value provided externally with 3 zones": {
			Runtime: imv1.Runtime{
				Spec: imv1.RuntimeSpec{
					Shoot: imv1.RuntimeShoot{
						Provider: fixProviderWithMultipleWorkersAndConfig(hyperscaler.TypeAWS, fixMultipleWorkers([]workerConfig{
							{"main-worker", "m6i.large", "gardenlinux", "1313.4.0", 1, 3, []string{"eu-central-1a"}},
							{"additional", "m6i.large", "gardenlinux", "1313.2.0", 1, 3, []string{"eu-central-1a", "eu-central-1b"}},
						}), fixAWSInfrastructureConfig(t, "10.250.0.0/22", []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}), fixAWSControlPlaneConfig()),
						Networking: imv1.Networking{
							Nodes: "10.250.0.0/22",
						},
					},
				},
			},
			EnableIMDSv2:               false,
			DefaultMachineImageName:    "gardenlinux",
			DefaultMachineImageVersion: "1312.3.0",
			CurrentShootWorkers: fixMultipleWorkers([]workerConfig{
				{"main-worker", "m6i.large", "gardenlinux", "1312.4.0", 1, 3, []string{"eu-central-1a"}},
				{"additional", "m6i.large", "gardenlinux", "1312.2.0", 1, 3, []string{"eu-central-1a", "eu-central-1b"}}}),
			ExpectedShootWorkers: fixMultipleWorkers([]workerConfig{
				{"main-worker", "m6i.large", "gardenlinux", "1313.4.0", 1, 3, []string{"eu-central-1a"}},
				{"additional", "m6i.large", "gardenlinux", "1313.2.0", 1, 3, []string{"eu-central-1a", "eu-central-1b"}}}),
			ExpectedZonesCount:         3,
			ExistingInfraConfig:        fixAWSInfrastructureConfig(t, "10.250.0.0/22", []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}),
			ExistingControlPlaneConfig: fixAWSControlPlaneConfig(),
		},
	} {
		t.Run(tname, func(t *testing.T) {
			// given
			shoot := testutils.FixEmptyGardenerShoot("cluster", "kcp-system")

			// when
			extender := NewProviderExtenderPatchOperation(tc.EnableIMDSv2, tc.DefaultMachineImageName, tc.DefaultMachineImageVersion, tc.CurrentShootWorkers, tc.ExistingInfraConfig, tc.ExistingControlPlaneConfig)
			err := extender(tc.Runtime, &shoot)

			// then
			require.NoError(t, err)

			assertProviderMultipleWorkers(t, tc.Runtime.Spec.Shoot, shoot, tc.EnableIMDSv2, tc.ExpectedShootWorkers)
			assertProviderSpecificConfigAWS(t, shoot, tc.ExpectedZonesCount)
			assertExistingZonesAWSInfrastructureNotModified(t, tc.ExistingInfraConfig, shoot.Spec.Provider.InfrastructureConfig)
		})
	}
}

func TestProviderExtenderForPatchWorkersUpdateErrors(t *testing.T) {
	for tname, tc := range map[string]struct {
		Runtime                    imv1.Runtime
		CurrentShootWorkers        []gardener.Worker
		ExistingInfraConfig        *runtime.RawExtension
		ExistingControlPlaneConfig *runtime.RawExtension
	}{
		"Existing InfrastructureConfig required": {
			Runtime: imv1.Runtime{
				Spec: imv1.RuntimeSpec{
					Shoot: imv1.RuntimeShoot{
						Provider: fixProviderWithMultipleWorkers(hyperscaler.TypeAWS, fixMultipleWorkers([]workerConfig{
							{"main-worker", "m6i.large", "gardenlinux", "1313.4.0", 1, 3, []string{"eu-central-1a"}},
						})),
						Networking: imv1.Networking{
							Nodes: "10.250.0.0/22",
						},
					},
				},
			},
			CurrentShootWorkers: fixMultipleWorkers([]workerConfig{
				{"main-worker", "m6i.large", "gardenlinux", "1312.4.0", 1, 3, []string{"eu-central-1a"}}}),
			ExistingInfraConfig:        nil,
			ExistingControlPlaneConfig: fixAWSControlPlaneConfig(),
		},
		"Main worker not defined": {
			Runtime: imv1.Runtime{
				Spec: imv1.RuntimeSpec{
					Shoot: imv1.RuntimeShoot{
						Provider: imv1.Provider{
							Type: "aws",
						},
						Networking: imv1.Networking{
							Nodes: "10.250.0.0/22",
						},
					},
				},
			},
			CurrentShootWorkers: fixMultipleWorkers([]workerConfig{
				{"main-worker", "m6i.large", "gardenlinux", "1312.4.0", 1, 3, []string{"eu-central-1a"}}}),
			ExistingInfraConfig:        fixAWSInfrastructureConfig(t, "10.250.0.0/22", []string{"eu-central-1a"}),
			ExistingControlPlaneConfig: fixAWSControlPlaneConfig(),
		},
	} {
		t.Run(tname, func(t *testing.T) {
			// given
			shoot := testutils.FixEmptyGardenerShoot("cluster", "kcp-system")

			// when
			extender := NewProviderExtenderPatchOperation(false, "", "", tc.CurrentShootWorkers, tc.ExistingInfraConfig, tc.ExistingControlPlaneConfig)
			err := extender(tc.Runtime, &shoot)

			// then
			require.Error(t, err)
		})
	}
}

func assertExistingZonesAWSInfrastructureNotModified(t *testing.T, infraConfigBeforeUpdate *runtime.RawExtension, infraConfigToUpdate *runtime.RawExtension) {
	var existingInfraConfig awsinfra.InfrastructureConfig
	err := yaml.Unmarshal(infraConfigBeforeUpdate.Raw, &existingInfraConfig)
	require.NoError(t, err)

	var newInfraConfig awsinfra.InfrastructureConfig
	err = yaml.Unmarshal(infraConfigToUpdate.Raw, &newInfraConfig)
	require.NoError(t, err)

	for i := 0; i < len(existingInfraConfig.Networks.Zones); i++ {
		existingZone := existingInfraConfig.Networks.Zones[i]
		require.Equal(t, existingZone, newInfraConfig.Networks.Zones[i])
	}
	require.Equal(t, existingInfraConfig.EnableECRAccess, newInfraConfig.EnableECRAccess)
	require.Equal(t, existingInfraConfig.DualStack, newInfraConfig.DualStack)
	require.Equal(t, existingInfraConfig.IgnoreTags, newInfraConfig.IgnoreTags)
	require.Equal(t, newInfraConfig.Networks.VPC.ID, existingInfraConfig.Networks.VPC.ID)
	require.Equal(t, newInfraConfig.Networks.VPC.CIDR, existingInfraConfig.Networks.VPC.CIDR)
	require.Equal(t, newInfraConfig.Networks.VPC.GatewayEndpoints, existingInfraConfig.Networks.VPC.GatewayEndpoints)
}

func fixProvider(providerType string, machineImageName, machineImageVersion string, zones []string) imv1.Provider {
	return imv1.Provider{
		Type: providerType,
		Workers: []gardener.Worker{
			{
				Name: "worker",
				Machine: gardener.Machine{
					Type:  "m6i.large",
					Image: fixMachineImage(machineImageName, machineImageVersion),
				},
				Minimum: 1,
				Maximum: 3,
				Zones:   zones,
			},
		},
	}
}

func fixWorkers(name, machineType, machineImageName, machineImageVersion string, min, max int32, zones []string) []gardener.Worker {
	return []gardener.Worker{
		{
			Name: name,
			Machine: gardener.Machine{
				Type: machineType,
				Image: &gardener.ShootMachineImage{
					Name:    machineImageName,
					Version: &machineImageVersion,
				},
			},
			Minimum: min,
			Maximum: max,
			Zones:   zones,
		},
	}
}

func fixMultipleWorkers(workers []workerConfig) []gardener.Worker {
	var result []gardener.Worker
	for _, w := range workers {
		result = append(result, gardener.Worker{
			Name: w.Name,
			Machine: gardener.Machine{
				Type:  w.MachineType,
				Image: fixMachineImage(w.MachineImageName, w.MachineImageVersion),
			},
			Minimum: w.hypescalerMin,
			Maximum: w.hyperscalerMax,
			Zones:   w.Zones,
		})
	}
	return result
}

func fixProviderWithMultipleWorkers(providerType string, workers []gardener.Worker) imv1.Provider {
	if len(workers) < 1 {
		return imv1.Provider{
			Type: providerType,
		}
	}

	var addPtr *[]gardener.Worker

	if len(workers) > 1 {
		additional := workers[1:]
		addPtr = &additional
	}

	return imv1.Provider{
		Type:              providerType,
		Workers:           workers[:1],
		AdditionalWorkers: addPtr,
	}
}

func fixProviderWithMultipleWorkersAndConfig(providerType string, workers []gardener.Worker, infraConfig, ctrlPlaneConfig *runtime.RawExtension) imv1.Provider {
	provider := fixProviderWithMultipleWorkers(providerType, workers)
	provider.InfrastructureConfig = infraConfig
	provider.ControlPlaneConfig = ctrlPlaneConfig
	return provider
}

func fixMachineImage(machineImageName, machineImageVersion string) *gardener.ShootMachineImage {
	if machineImageVersion != "" {
		return &gardener.ShootMachineImage{
			Version: &machineImageVersion,
			Name:    machineImageName,
		}
	}

	return &gardener.ShootMachineImage{}
}

func assertProvider(t *testing.T, runtimeShoot imv1.RuntimeShoot, shoot gardener.Shoot, expectWorkerConfig bool, expectedMachineImageName, expectedMachineImageVersion string) {
	assert.Equal(t, runtimeShoot.Provider.Type, shoot.Spec.Provider.Type)
	assert.Equal(t, runtimeShoot.Provider.Workers, shoot.Spec.Provider.Workers)
	assert.Equal(t, false, shoot.Spec.Provider.WorkersSettings.SSHAccess.Enabled)
	assert.NotEmpty(t, shoot.Spec.Provider.InfrastructureConfig)
	assert.NotEmpty(t, shoot.Spec.Provider.InfrastructureConfig.Raw)
	assert.NotEmpty(t, shoot.Spec.Provider.ControlPlaneConfig)
	assert.NotEmpty(t, shoot.Spec.Provider.ControlPlaneConfig.Raw)

	for _, worker := range shoot.Spec.Provider.Workers {
		if expectWorkerConfig {
			assert.NotEmpty(t, worker.ProviderConfig)
			assert.NotEmpty(t, worker.ProviderConfig.Raw)
		} else {
			assert.Empty(t, worker.ProviderConfig)
		}
		assert.Equal(t, expectedMachineImageVersion, *worker.Machine.Image.Version)
		assert.Equal(t, expectedMachineImageName, worker.Machine.Image.Name)
	}
}

func assertProviderMultipleWorkers(t *testing.T, runtimeShoot imv1.RuntimeShoot, shoot gardener.Shoot, expectWorkerConfig bool, expectedWorkers []gardener.Worker) {
	assert.Equal(t, len(expectedWorkers), len(shoot.Spec.Provider.Workers))
	assert.Equal(t, shoot.Spec.Provider.Type, runtimeShoot.Provider.Type)
	assert.Equal(t, false, shoot.Spec.Provider.WorkersSettings.SSHAccess.Enabled)
	assert.NotEmpty(t, shoot.Spec.Provider.InfrastructureConfig)
	assert.NotEmpty(t, shoot.Spec.Provider.InfrastructureConfig.Raw)
	assert.NotEmpty(t, shoot.Spec.Provider.ControlPlaneConfig)
	assert.NotEmpty(t, shoot.Spec.Provider.ControlPlaneConfig.Raw)

	for i, worker := range shoot.Spec.Provider.Workers {
		if expectWorkerConfig {
			assert.NotEmpty(t, worker.ProviderConfig)
			assert.NotEmpty(t, worker.ProviderConfig.Raw)
		} else {
			assert.Empty(t, worker.ProviderConfig)
		}

		expected := expectedWorkers[i]
		assert.Equal(t, expected.Name, worker.Name)
		assert.Equal(t, expected.Zones, worker.Zones)
		assert.Equal(t, expected.Minimum, worker.Minimum)
		assert.Equal(t, expected.Maximum, worker.Maximum)
		assert.Equal(t, expected.Machine.Type, worker.Machine.Type)

		if expected.MaxSurge != nil {
			assert.NotNil(t, worker.MaxSurge)
			assert.Equal(t, *expected.MaxSurge, *worker.MaxSurge)
		}

		if expected.MaxUnavailable != nil {
			assert.NotNil(t, worker.MaxUnavailable)
			assert.Equal(t, *expected.MaxUnavailable, *worker.MaxUnavailable)
		}

		if expected.Machine.Architecture != nil {
			assert.NotNil(t, worker.Machine.Architecture)
			assert.Equal(t, *expected.Machine.Architecture, *worker.Machine.Architecture)
		}

		assert.NotNil(t, worker.Machine.Image)
		assert.Equal(t, expected.Machine.Image.Name, worker.Machine.Image.Name)
		assert.NotEmpty(t, expected.Machine.Image.Version)
		assert.NotEmpty(t, worker.Machine.Image.Version)
		assert.Equal(t, *expected.Machine.Image.Version, *worker.Machine.Image.Version)
	}
}
