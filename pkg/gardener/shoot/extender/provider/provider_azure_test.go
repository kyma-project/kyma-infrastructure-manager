package provider

import (
	"encoding/json"
	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/extender/testutils"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/hyperscaler"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/hyperscaler/azure"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/yaml"
	"testing"
)

func TestProviderExtenderForCreateAzure(t *testing.T) {
	// tests of ProviderExtenderForCreateOperation for Azure provider
	for tname, tc := range map[string]struct {
		Runtime                     imv1.Runtime
		DefaultMachineImageVersion  string
		ExpectedMachineImageVersion string
		DefaultMachineImageName     string
		ExpectedMachineImageName    string
		CurrentZonesConfig          []string
		ExpectedZonesCount          int
	}{
		"Create provider specific config for Azure without worker config and one zone": {
			Runtime: imv1.Runtime{
				Spec: imv1.RuntimeSpec{
					Shoot: imv1.RuntimeShoot{
						Provider: fixProvider(hyperscaler.TypeAzure, "ubuntu", "18.04", []string{"1"}),
						Networking: imv1.Networking{
							Nodes: "10.250.0.0/22",
						},
					},
				},
			},
			DefaultMachineImageVersion:  "18.04-LTS",
			ExpectedMachineImageVersion: "18.04",
			ExpectedMachineImageName:    "ubuntu",
			ExpectedZonesCount:          1,
		},
		"Create provider specific config for Azure without worker config and two zones": {
			Runtime: imv1.Runtime{
				Spec: imv1.RuntimeSpec{
					Shoot: imv1.RuntimeShoot{
						Provider: fixProvider(hyperscaler.TypeAzure, "ubuntu", "18.04", []string{"1", "2"}),
						Networking: imv1.Networking{
							Nodes: "10.250.0.0/22",
						},
					},
				},
			},
			DefaultMachineImageVersion:  "18.04-LTS",
			ExpectedMachineImageVersion: "18.04",
			ExpectedMachineImageName:    "ubuntu",
			ExpectedZonesCount:          2,
		},
		"Create provider specific config for Azure without worker config and three zones": {
			Runtime: imv1.Runtime{
				Spec: imv1.RuntimeSpec{
					Shoot: imv1.RuntimeShoot{
						Provider: fixProvider(hyperscaler.TypeAzure, "ubuntu", "18.04", []string{"1", "2", "3"}),
						Networking: imv1.Networking{
							Nodes: "10.250.0.0/22",
						},
					},
				},
			},
			DefaultMachineImageVersion:  "18.04-LTS",
			ExpectedMachineImageVersion: "18.04",
			ExpectedMachineImageName:    "ubuntu",
			ExpectedZonesCount:          3,
		},
	} {
		t.Run(tname, func(t *testing.T) {
			// given
			shoot := testutils.FixEmptyGardenerShoot("cluster", "kcp-system")

			// when

			extender := NewProviderExtenderForCreateOperation(false, tc.DefaultMachineImageName, tc.DefaultMachineImageVersion)
			err := extender(tc.Runtime, &shoot)

			// then
			require.NoError(t, err)

			assertProvider(t, tc.Runtime.Spec.Shoot, shoot, false, tc.ExpectedMachineImageName, tc.ExpectedMachineImageVersion)
			assertProviderSpecificConfigAzure(t, shoot, tc.ExpectedZonesCount)
		})
	}
}

// tests of NewProviderExtenderPatch for workers update operation on Azure provider
func TestProviderExtenderForPatchWorkersUpdateAzure(t *testing.T) {
	for tname, tc := range map[string]struct {
		Runtime                    imv1.Runtime
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
						Provider: fixProviderWithMultipleWorkers(hyperscaler.TypeAzure, fixMultipleWorkers([]workerConfig{
							{"main-worker", "azure.small", "gardenlinux", "1312.4.0", 1, 3, []string{"1", "2", "3"}},
							{"next-worker", "azure.large", "gardenlinux", "1312.2.0", 1, 3, []string{"1", "2", "3"}},
						})),
						Networking: imv1.Networking{
							Nodes: "10.250.0.0/22",
						},
					},
				},
			},
			DefaultMachineImageName:    "gardenlinux",
			DefaultMachineImageVersion: "1312.3.0",
			ExpectedZonesCount:         3,
			CurrentShootWorkers:        fixWorkers("main-worker", "m6i.large", "gardenlinux", "1312.4.0", 1, 3, []string{"1", "2", "3"}),
			ExpectedShootWorkers: fixMultipleWorkers([]workerConfig{
				{"main-worker", "azure.small", "gardenlinux", "1312.4.0", 1, 3, []string{"1", "2", "3"}},
				{"next-worker", "azure.large", "gardenlinux", "1312.2.0", 1, 3, []string{"1", "2", "3"}}}),
			ExistingInfraConfig:        fixAzureInfrastructureConfig(t, "10.250.0.0/22", []string{"1", "2", "3"}),
			ExistingControlPlaneConfig: fixAzureControlPlaneConfig(),
		},
		"Add additional worker with new zone": {
			Runtime: imv1.Runtime{
				Spec: imv1.RuntimeSpec{
					Shoot: imv1.RuntimeShoot{
						Provider: fixProviderWithMultipleWorkers(hyperscaler.TypeAzure, fixMultipleWorkers([]workerConfig{
							{"main-worker", "azure.small", "gardenlinux", "1312.4.0", 1, 3, []string{"1", "2", "3"}},
							{"next-worker", "azure.large", "gardenlinux", "1312.2.0", 1, 3, []string{"1", "2", "4"}},
						})),
						Networking: imv1.Networking{
							Nodes: "10.250.0.0/22",
						},
					},
				},
			},
			DefaultMachineImageName:    "gardenlinux",
			DefaultMachineImageVersion: "1312.3.0",
			ExpectedZonesCount:         4,
			CurrentShootWorkers:        fixWorkers("main-worker", "m6i.large", "gardenlinux", "1312.4.0", 1, 3, []string{"1", "2", "3"}),
			ExpectedShootWorkers: fixMultipleWorkers([]workerConfig{
				{"main-worker", "azure.small", "gardenlinux", "1312.4.0", 1, 3, []string{"1", "2", "3"}},
				{"next-worker", "azure.large", "gardenlinux", "1312.2.0", 1, 3, []string{"1", "2", "4"}}}),
			ExistingInfraConfig:        fixAzureInfrastructureConfig(t, "10.250.0.0/22", []string{"1", "2", "3"}),
			ExistingControlPlaneConfig: fixAzureControlPlaneConfig(),
		},
		"Add additional worker - extend existing additional worker from non HA setup to HA setup by adding more zones (use zone already specified in the existing shoot)": {
			Runtime: imv1.Runtime{
				Spec: imv1.RuntimeSpec{
					Shoot: imv1.RuntimeShoot{
						Provider: fixProviderWithMultipleWorkers(hyperscaler.TypeAzure, fixMultipleWorkers([]workerConfig{
							{"main-worker", "m6i.large", "gardenlinux", "1312.4.0", 1, 3, []string{"1", "2", "3"}},
							{"additional", "m6i.large", "gardenlinux", "1312.2.0", 1, 3, []string{"1", "2", "3"}},
						})),
						Networking: imv1.Networking{
							Nodes: "10.250.0.0/22",
						},
					},
				},
			},
			DefaultMachineImageName:    "gardenlinux",
			DefaultMachineImageVersion: "1312.3.0",
			ExpectedZonesCount:         3,
			CurrentShootWorkers: fixMultipleWorkers([]workerConfig{
				{"main-worker", "m6i.large", "gardenlinux", "1312.4.0", 1, 3, []string{"1", "2", "3"}},
				{"additional", "m6i.large", "gardenlinux", "1312.2.0", 1, 3, []string{"2"}}}),
			ExpectedShootWorkers: fixMultipleWorkers([]workerConfig{
				{"main-worker", "m6i.large", "gardenlinux", "1312.4.0", 1, 3, []string{"1", "2", "3"}},
				{"additional", "m6i.large", "gardenlinux", "1312.2.0", 1, 3, []string{"2", "1", "3"}}}),
			ExistingInfraConfig:        fixAzureInfrastructureConfig(t, "10.250.0.0/22", []string{"1", "2", "3"}),
			ExistingControlPlaneConfig: fixAzureControlPlaneConfig(),
		},
		"Add additional worker - extend existing additional worker from non HA setup to HA setup by adding more zones (use zone not specified in the existing shoot)": {
			Runtime: imv1.Runtime{
				Spec: imv1.RuntimeSpec{
					Shoot: imv1.RuntimeShoot{
						Provider: fixProviderWithMultipleWorkers(hyperscaler.TypeAzure, fixMultipleWorkers([]workerConfig{
							{"main-worker", "m6i.large", "gardenlinux", "1312.4.0", 1, 3, []string{"1", "2", "3"}},
							{"additional", "m6i.large", "gardenlinux", "1312.2.0", 1, 3, []string{"1", "2", "4"}},
						})),
						Networking: imv1.Networking{
							Nodes: "10.250.0.0/22",
						},
					},
				},
			},
			DefaultMachineImageName:    "gardenlinux",
			DefaultMachineImageVersion: "1312.3.0",
			ExpectedZonesCount:         4,
			CurrentShootWorkers: fixMultipleWorkers([]workerConfig{
				{"main-worker", "m6i.large", "gardenlinux", "1312.4.0", 1, 3, []string{"1", "2", "3"}},
				{"additional", "m6i.large", "gardenlinux", "1312.2.0", 1, 3, []string{"2"}}}),
			ExpectedShootWorkers: fixMultipleWorkers([]workerConfig{
				{"main-worker", "m6i.large", "gardenlinux", "1312.4.0", 1, 3, []string{"1", "2", "3"}},
				{"additional", "m6i.large", "gardenlinux", "1312.2.0", 1, 3, []string{"2", "1", "4"}}}),
			ExistingInfraConfig:        fixAzureInfrastructureConfig(t, "10.250.0.0/22", []string{"1", "2", "3"}),
			ExistingControlPlaneConfig: fixAzureControlPlaneConfig(),
		},
		"Add additional worker - extend existing additional worker from non HA setup to HA setup by adding more zones, infrastructureConfig has no zones - legacy azure-lite case": {
			Runtime: imv1.Runtime{
				Spec: imv1.RuntimeSpec{
					Shoot: imv1.RuntimeShoot{
						Provider: fixProviderWithMultipleWorkers(hyperscaler.TypeAzure, fixMultipleWorkers([]workerConfig{
							{"main-worker", "m6i.large", "gardenlinux", "1312.4.0", 1, 3, []string{"1", "2", "3"}},
							{"additional", "m6i.large", "gardenlinux", "1312.2.0", 1, 3, []string{"1", "2", "3"}},
						})),
						Networking: imv1.Networking{
							Nodes: "10.250.0.0/22",
						},
					},
				},
			},
			DefaultMachineImageName:    "gardenlinux",
			DefaultMachineImageVersion: "1312.3.0",
			ExpectedZonesCount:         0,
			CurrentShootWorkers: fixMultipleWorkers([]workerConfig{
				{"main-worker", "m6i.large", "gardenlinux", "1312.4.0", 1, 3, []string{"1", "2", "3"}},
				{"additional", "m6i.large", "gardenlinux", "1312.2.0", 1, 3, []string{"2"}}}),
			ExpectedShootWorkers: fixMultipleWorkers([]workerConfig{
				{"main-worker", "m6i.large", "gardenlinux", "1312.4.0", 1, 3, []string{"1", "2", "3"}},
				{"additional", "m6i.large", "gardenlinux", "1312.2.0", 1, 3, []string{"2", "1", "3"}}}),
			ExistingInfraConfig:        fixAzureInfrastructureConfig(t, "10.250.0.0/22", []string{}),
			ExistingControlPlaneConfig: fixAzureControlPlaneConfig(),
		},
		"Add additional worker - extend existing additional worker from non HA setup to HA setup by adding zone not specified in the shoot, infrastructureConfig has no zones - legacy azure-lite case": {
			Runtime: imv1.Runtime{
				Spec: imv1.RuntimeSpec{
					Shoot: imv1.RuntimeShoot{
						Provider: fixProviderWithMultipleWorkers(hyperscaler.TypeAzure, fixMultipleWorkers([]workerConfig{
							{"main-worker", "m6i.large", "gardenlinux", "1312.4.0", 1, 3, []string{"1", "2", "3"}},
							{"additional", "m6i.large", "gardenlinux", "1312.2.0", 1, 3, []string{"1", "2", "3", "4"}},
						})),
						Networking: imv1.Networking{
							Nodes: "10.250.0.0/22",
						},
					},
				},
			},
			DefaultMachineImageName:    "gardenlinux",
			DefaultMachineImageVersion: "1312.3.0",
			ExpectedZonesCount:         0,
			CurrentShootWorkers: fixMultipleWorkers([]workerConfig{
				{"main-worker", "m6i.large", "gardenlinux", "1312.4.0", 1, 3, []string{"1", "2", "3"}},
				{"additional", "m6i.large", "gardenlinux", "1312.2.0", 1, 3, []string{"2"}}}),
			ExpectedShootWorkers: fixMultipleWorkers([]workerConfig{
				{"main-worker", "m6i.large", "gardenlinux", "1312.4.0", 1, 3, []string{"1", "2", "3"}},
				{"additional", "m6i.large", "gardenlinux", "1312.2.0", 1, 3, []string{"2", "1", "3", "4"}}}),
			ExistingInfraConfig:        fixAzureInfrastructureConfig(t, "10.250.0.0/22", []string{}),
			ExistingControlPlaneConfig: fixAzureControlPlaneConfig(),
		},
		"Remove additional worker from existing set of workers": {
			Runtime: imv1.Runtime{
				Spec: imv1.RuntimeSpec{
					Shoot: imv1.RuntimeShoot{
						Provider: fixProviderWithMultipleWorkers(hyperscaler.TypeAzure, fixMultipleWorkers([]workerConfig{
							{"main-worker", "azure.small", "gardenlinux", "1312.4.0", 1, 3, []string{"1", "2", "3"}}})),
						Networking: imv1.Networking{
							Nodes: "10.250.0.0/22",
						},
					},
				},
			},
			DefaultMachineImageName:    "gardenlinux",
			DefaultMachineImageVersion: "1312.3.0",
			ExpectedZonesCount:         3,
			CurrentShootWorkers: fixMultipleWorkers([]workerConfig{
				{"main-worker", "azure.small", "gardenlinux", "1312.4.0", 1, 3, []string{"1", "2", "3"}},
				{"next-worker", "azure.large", "gardenlinux", "1312.2.0", 1, 3, []string{"1", "2", "3"}}}),
			ExpectedShootWorkers: fixMultipleWorkers([]workerConfig{
				{"main-worker", "azure.small", "gardenlinux", "1312.4.0", 1, 3, []string{"1", "2", "3"}}}),
			ExistingInfraConfig:        fixAzureInfrastructureConfig(t, "10.250.0.0/22", []string{"1", "2", "3"}),
			ExistingControlPlaneConfig: fixAzureControlPlaneConfig(),
		},
		"Update machine type and image name and version in multiple workers separately": {
			Runtime: imv1.Runtime{
				Spec: imv1.RuntimeSpec{
					Shoot: imv1.RuntimeShoot{
						Provider: fixProviderWithMultipleWorkers(hyperscaler.TypeAzure, fixMultipleWorkers([]workerConfig{
							{"main-worker", "azure.large", "gardenlinux", "1313.4.0", 1, 3, []string{"1", "2", "3"}},
							{"next-worker", "azure.big", "gardenlinux", "1313.2.0", 1, 3, []string{"1", "2", "3"}},
						})),
						Networking: imv1.Networking{
							Nodes: "10.250.0.0/22",
						},
					},
				},
			},
			DefaultMachineImageName:    "gardenlinux",
			DefaultMachineImageVersion: "1312.3.0",
			CurrentShootWorkers: fixMultipleWorkers([]workerConfig{
				{"main-worker", "azure.small", "gardenlinux", "1312.4.0", 1, 3, []string{"1", "2", "3"}},
				{"next-worker", "azure.small", "gardenlinux", "1312.2.0", 1, 3, []string{"1", "2", "3"}}}),
			ExpectedShootWorkers: fixMultipleWorkers([]workerConfig{
				{"main-worker", "azure.large", "gardenlinux", "1313.4.0", 1, 3, []string{"1", "2", "3"}},
				{"next-worker", "azure.big", "gardenlinux", "1313.2.0", 1, 3, []string{"1", "2", "3"}}}),
			ExpectedZonesCount:         3,
			ExistingInfraConfig:        fixAzureInfrastructureConfig(t, "10.250.0.0/22", []string{"1", "2", "3"}),
			ExistingControlPlaneConfig: fixAzureControlPlaneConfig(),
		},
		"Remove worker from existing set of workers networking zones set in infrastructureConfig should not change": {
			Runtime: imv1.Runtime{
				Spec: imv1.RuntimeSpec{
					Shoot: imv1.RuntimeShoot{
						Provider: fixProviderWithMultipleWorkers(hyperscaler.TypeAzure, fixMultipleWorkers([]workerConfig{
							{"main-worker", "m6i.large", "gardenlinux", "1313.4.0", 1, 3, []string{"1"}},
						})),
						Networking: imv1.Networking{
							Nodes: "10.250.0.0/22",
						},
					},
				},
			},
			DefaultMachineImageName:    "gardenlinux",
			DefaultMachineImageVersion: "1312.3.0",
			CurrentShootWorkers: fixMultipleWorkers([]workerConfig{
				{"main-worker", "m6i.large", "gardenlinux", "1312.4.0", 1, 3, []string{"1"}},
				{"next-worker", "m6i.large", "gardenlinux", "1312.2.0", 1, 3, []string{"1", "2", "3"}}}),
			ExpectedShootWorkers: fixMultipleWorkers([]workerConfig{
				{"main-worker", "m6i.large", "gardenlinux", "1313.4.0", 1, 3, []string{"1"}}}),
			ExpectedZonesCount:         3,
			ExistingInfraConfig:        fixAzureInfrastructureConfig(t, "10.250.0.0/22", []string{"1", "2", "3"}),
			ExistingControlPlaneConfig: fixAzureControlPlaneConfig(),
		},
	} {
		t.Run(tname, func(t *testing.T) {
			// given
			shoot := testutils.FixEmptyGardenerShoot("cluster", "kcp-system")

			// when
			extender := NewProviderExtenderPatchOperation(false, tc.DefaultMachineImageName, tc.DefaultMachineImageVersion, tc.CurrentShootWorkers, tc.ExistingInfraConfig, tc.ExistingControlPlaneConfig)
			err := extender(tc.Runtime, &shoot)

			// then
			require.NoError(t, err)

			assertProviderMultipleWorkers(t, tc.Runtime.Spec.Shoot, shoot, false, tc.ExpectedShootWorkers)
			assertProviderSpecificConfigAzure(t, shoot, tc.ExpectedZonesCount)
			assertExistingZonesAzureInfrastructureNotModified(t, tc.ExistingInfraConfig, shoot.Spec.Provider.InfrastructureConfig)
		})
	}
}

func assertExistingZonesAzureInfrastructureNotModified(t *testing.T, infraConfigBeforeUpdate *runtime.RawExtension, infraConfigToUpdate *runtime.RawExtension) {
	var existingInfraConfig azure.InfrastructureConfig
	err := yaml.Unmarshal(infraConfigBeforeUpdate.Raw, &existingInfraConfig)
	require.NoError(t, err)

	var newInfraConfig azure.InfrastructureConfig
	err = yaml.Unmarshal(infraConfigToUpdate.Raw, &newInfraConfig)
	require.NoError(t, err)

	for i := 0; i < len(existingInfraConfig.Networks.Zones); i++ {
		existingZone := existingInfraConfig.Networks.Zones[i]
		require.Equal(t, existingZone, newInfraConfig.Networks.Zones[i])
	}
	require.Equal(t, existingInfraConfig.ResourceGroup, newInfraConfig.ResourceGroup)
	require.Equal(t, existingInfraConfig.Zoned, newInfraConfig.Zoned)
	require.Equal(t, existingInfraConfig.Networks.Workers, newInfraConfig.Networks.Workers)
	require.Equal(t, existingInfraConfig.Networks.NatGateway, newInfraConfig.Networks.NatGateway)
	require.Equal(t, existingInfraConfig.Networks.ServiceEndpoints, newInfraConfig.Networks.ServiceEndpoints)
	require.Equal(t, existingInfraConfig.Networks.VNet, newInfraConfig.Networks.VNet)
}

func TestProviderExtenderForCreateMultipleWorkersAzure(t *testing.T) {
	// tests of NewProviderExtenderForCreateOperation for workers create operation
	for tname, tc := range map[string]struct {
		Runtime                    imv1.Runtime
		DefaultMachineImageVersion string
		DefaultMachineImageName    string
		CurrentShootWorkers        []gardener.Worker
		ExistingInfraConfig        *runtime.RawExtension
		ExistingControlPlaneConfig *runtime.RawExtension
		ExpectedShootWorkers       []gardener.Worker
		ExpectedZonesCount         int
	}{
		"Create multiple workers without worker config": {
			Runtime: imv1.Runtime{
				Spec: imv1.RuntimeSpec{
					Shoot: imv1.RuntimeShoot{
						Provider: fixProviderWithMultipleWorkers(hyperscaler.TypeAzure, fixMultipleWorkers([]workerConfig{
							{"main-worker", "m6i.large", "gardenlinux", "1310.4.0", 1, 3, []string{"1"}},
							{"additional", "m7i.large", "gardenlinux", "1311.2.0", 2, 4, []string{"2", "3"}},
							{"another", "m8i.large", "gardenlinux", "1312.2.0", 3, 5, []string{"3"}},
						})),
						Networking: imv1.Networking{
							Nodes: "10.250.0.0/22",
						},
					},
				},
			},
			DefaultMachineImageName:    "gardenlinux",
			DefaultMachineImageVersion: "1312.3.0",
			ExpectedZonesCount:         3,
			ExpectedShootWorkers: fixMultipleWorkers([]workerConfig{
				{"main-worker", "m6i.large", "gardenlinux", "1310.4.0", 1, 3, []string{"1"}},
				{"additional", "m7i.large", "gardenlinux", "1311.2.0", 2, 4, []string{"2", "3"}},
				{"another", "m8i.large", "gardenlinux", "1312.2.0", 3, 5, []string{"3"}}}),
		},
	} {
		t.Run(tname, func(t *testing.T) {
			// given
			shoot := testutils.FixEmptyGardenerShoot("cluster", "kcp-system")

			// when
			extender := NewProviderExtenderForCreateOperation(false, tc.DefaultMachineImageName, tc.DefaultMachineImageVersion)
			err := extender(tc.Runtime, &shoot)

			// then
			require.NoError(t, err)

			assertProviderMultipleWorkers(t, tc.Runtime.Spec.Shoot, shoot, false, tc.ExpectedShootWorkers)
			assertProviderSpecificConfigAzure(t, shoot, tc.ExpectedZonesCount)
		})
	}
}

func fixAzureInfrastructureConfig(t *testing.T, workersCIDR string, zones []string) *runtime.RawExtension {
	infraConfig, err := azure.NewInfrastructureConfig(workersCIDR, zones)

	require.NoError(t, err)

	infraConfig.ResourceGroup = &azure.ResourceGroup{
		Name: "resource-group",
	}
	infraConfig.Networks.VNet.ResourceGroup = ptr.To("resource-group")
	infraConfig.Networks.VNet.Name = ptr.To("vnet")
	infraConfig.Networks.Workers = ptr.To("workers")
	infraConfig.Networks.NatGateway = &azure.NatGateway{
		Enabled:                      true,
		IdleConnectionTimeoutMinutes: 1,
		Zone:                         1,
		IPAddresses: []azure.PublicIPReference{{
			Name:          "public-ip",
			ResourceGroup: "resource-group",
			Zone:          1,
		},
		},
	}

	for i := 0; i < len(zones); i++ {
		infraConfig.Networks.Zones[i].ServiceEndpoints = []string{"service-endpoint"}
		infraConfig.Networks.Zones[i].NatGateway.Zone = 1
		infraConfig.Networks.Zones[i].NatGateway.IPAddresses = []azure.PublicIPReference{
			{
				Name:          "public-ip",
				ResourceGroup: "resource-group",
				Zone:          1,
			},
		}
	}

	infraConfigBytes, err := json.Marshal(infraConfig)
	require.NoError(t, err)

	return &runtime.RawExtension{Raw: infraConfigBytes}
}

func fixAzureControlPlaneConfig() *runtime.RawExtension {
	infraConfig, _ := azure.GetControlPlaneConfig([]string{})
	return &runtime.RawExtension{Raw: infraConfig}
}

func assertProviderSpecificConfigAzure(t *testing.T, shoot gardener.Shoot, expectedZonesCount int) {
	var infrastructureConfig azure.InfrastructureConfig

	err := json.Unmarshal(shoot.Spec.Provider.InfrastructureConfig.Raw, &infrastructureConfig)
	require.NoError(t, err)

	assert.Equal(t, expectedZonesCount, len(infrastructureConfig.Networks.Zones))
}
