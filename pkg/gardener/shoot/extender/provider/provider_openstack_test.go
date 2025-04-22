package provider

import (
	"encoding/json"
	ostext "github.com/gardener/gardener-extension-provider-openstack/pkg/apis/openstack/v1alpha1"
	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/extender/testutils"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/hyperscaler"
	ops "github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/hyperscaler/openstack"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	"testing"
)

func TestProviderExtenderForCreateOpenstack(t *testing.T) {
	// tests of NewProviderExtenderForCreateOperation for workers create operation
	for tname, tc := range map[string]struct {
		Runtime                    imv1.Runtime
		DefaultMachineImageVersion string
		DefaultMachineImageName    string
		CurrentShootWorkers        []gardener.Worker
		ExistingInfraConfig        *runtime.RawExtension
		ExistingControlPlaneConfig *runtime.RawExtension
		ExpectedShootWorkers       []gardener.Worker
		ExpectedInfraConfigCIDR    string
	}{
		"Create single Openstack worker": {
			Runtime: imv1.Runtime{
				Spec: imv1.RuntimeSpec{
					Shoot: imv1.RuntimeShoot{
						Provider: fixProviderWithMultipleWorkers(hyperscaler.TypeOpenStack, fixMultipleWorkers([]workerConfig{
							{"main-worker", "openstack.large", "gardenlinux", "1310.4.0", 1, 3, []string{"eu-de-1a", "eu-de-1b", "eu-de-1c"}},
						})),
						Networking: imv1.Networking{
							Nodes: "10.250.0.0/22",
						},
					},
				},
			},
			DefaultMachineImageName:    "gardenlinux",
			DefaultMachineImageVersion: "1312.3.0",
			ExpectedShootWorkers: fixMultipleWorkers([]workerConfig{
				{"main-worker", "openstack.large", "gardenlinux", "1310.4.0", 1, 3, []string{"eu-de-1a", "eu-de-1b", "eu-de-1c"}},
			}),
			ExpectedInfraConfigCIDR: "10.250.0.0/22",
		},
		"Create multiple Openstack workers": {
			Runtime: imv1.Runtime{
				Spec: imv1.RuntimeSpec{
					Shoot: imv1.RuntimeShoot{
						Provider: fixProviderWithMultipleWorkers(hyperscaler.TypeOpenStack, fixMultipleWorkers([]workerConfig{
							{"main-worker", "openstack.small", "gardenlinux", "1310.4.0", 1, 3, []string{"eu-de-1a"}},
							{"additional", "openstack.big", "gardenlinux", "1311.2.0", 2, 4, []string{"eu-de-1b", "eu-de-1c"}},
							{"another", "openstack.large", "gardenlinux", "1312.2.0", 3, 5, []string{"eu-de-1c"}},
						})),
						Networking: imv1.Networking{
							Nodes: "10.250.0.0/22",
						},
					},
				},
			},
			DefaultMachineImageName:    "gardenlinux",
			DefaultMachineImageVersion: "1312.3.0",
			ExpectedInfraConfigCIDR:    "10.250.0.0/22",
			ExpectedShootWorkers: fixMultipleWorkers([]workerConfig{
				{"main-worker", "openstack.small", "gardenlinux", "1310.4.0", 1, 3, []string{"eu-de-1a"}},
				{"additional", "openstack.big", "gardenlinux", "1311.2.0", 2, 4, []string{"eu-de-1b", "eu-de-1c"}},
				{"another", "openstack.large", "gardenlinux", "1312.2.0", 3, 5, []string{"eu-de-1c"}}}),
		},
		"Create multiple Openstack workers with custom CIDR Infrastructure Config": {
			Runtime: imv1.Runtime{
				Spec: imv1.RuntimeSpec{
					Shoot: imv1.RuntimeShoot{
						Provider: fixProviderWithMultipleWorkersAndConfig(hyperscaler.TypeOpenStack, fixMultipleWorkers([]workerConfig{
							{"main-worker", "openstack.small", "gardenlinux", "1310.4.0", 1, 3, []string{"eu-de-1a"}},
							{"additional", "openstack.big", "gardenlinux", "1311.2.0", 2, 4, []string{"eu-de-1b", "eu-de-1c"}},
							{"another", "openstack.large", "gardenlinux", "1312.2.0", 3, 5, []string{"eu-de-1c"}},
						}), fixOpenstackInfrastructureConfig("10.250.0.0/16"), fixOpenstackControlPlaneConfig()),
						Networking: imv1.Networking{
							Nodes: "10.250.0.0/22",
						},
					},
				},
			},
			DefaultMachineImageName:    "gardenlinux",
			DefaultMachineImageVersion: "1312.3.0",
			ExpectedInfraConfigCIDR:    "10.250.0.0/16",
			ExpectedShootWorkers: fixMultipleWorkers([]workerConfig{
				{"main-worker", "openstack.small", "gardenlinux", "1310.4.0", 1, 3, []string{"eu-de-1a"}},
				{"additional", "openstack.big", "gardenlinux", "1311.2.0", 2, 4, []string{"eu-de-1b", "eu-de-1c"}},
				{"another", "openstack.large", "gardenlinux", "1312.2.0", 3, 5, []string{"eu-de-1c"}}}),
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
			assertProviderSpecificConfigOpenstack(t, shoot, tc.ExpectedInfraConfigCIDR)
		})
	}
}

func TestProviderExtenderForPatchWorkersUpdateOpenstack(t *testing.T) {
	// tests of NewProviderExtenderPatch for workers update operation for Openstack provider
	for tname, tc := range map[string]struct {
		Runtime                    imv1.Runtime
		DefaultMachineImageVersion string
		DefaultMachineImageName    string
		CurrentShootWorkers        []gardener.Worker
		ExistingInfraConfig        *runtime.RawExtension
		ExistingControlPlaneConfig *runtime.RawExtension
		ExpectedShootWorkers       []gardener.Worker
		ExpectedInfraConfigCIDR    string
	}{
		"Add additional worker": {
			Runtime: imv1.Runtime{
				Spec: imv1.RuntimeSpec{
					Shoot: imv1.RuntimeShoot{
						Provider: fixProviderWithMultipleWorkers(hyperscaler.TypeOpenStack, fixMultipleWorkers([]workerConfig{
							{"main-worker", "openstack.small", "gardenlinux", "1312.4.0", 1, 3, []string{"eu-de-1a", "eu-de-1b", "eu-de-1c"}},
							{"next-worker", "openstack.big", "gardenlinux", "1312.2.0", 1, 3, []string{"eu-de-1a", "eu-de-1b", "eu-de-1c"}},
						})),
						Networking: imv1.Networking{
							Nodes: "10.250.0.0/22",
						},
					},
				},
			},
			DefaultMachineImageName:    "gardenlinux",
			DefaultMachineImageVersion: "1312.3.0",
			CurrentShootWorkers:        fixWorkers("main-worker", "openstack.small", "gardenlinux", "1312.4.0", 1, 3, []string{"eu-de-1a", "eu-de-1b", "eu-de-1c"}),
			ExpectedShootWorkers: fixMultipleWorkers([]workerConfig{
				{"main-worker", "openstack.small", "gardenlinux", "1312.4.0", 1, 3, []string{"eu-de-1a", "eu-de-1b", "eu-de-1c"}},
				{"next-worker", "openstack.big", "gardenlinux", "1312.2.0", 1, 3, []string{"eu-de-1a", "eu-de-1b", "eu-de-1c"}}}),
			ExistingInfraConfig:        fixOpenstackInfrastructureConfig("10.250.0.0/22"),
			ExistingControlPlaneConfig: fixOpenstackControlPlaneConfig(),
			ExpectedInfraConfigCIDR:    "10.250.0.0/22",
		},
		"Add additional worker with new zone": {
			Runtime: imv1.Runtime{
				Spec: imv1.RuntimeSpec{
					Shoot: imv1.RuntimeShoot{
						Provider: fixProviderWithMultipleWorkers(hyperscaler.TypeOpenStack, fixMultipleWorkers([]workerConfig{
							{"main-worker", "openstack.small", "gardenlinux", "1312.4.0", 1, 3, []string{"eu-de-1a", "eu-de-1b", "eu-de-1c"}},
							{"next-worker", "openstack.big", "gardenlinux", "1312.2.0", 1, 3, []string{"eu-de-1a", "eu-de-1b", "eu-de-1d"}},
						})),
						Networking: imv1.Networking{
							Nodes: "10.250.0.0/22",
						},
					},
				},
			},
			DefaultMachineImageName:    "gardenlinux",
			DefaultMachineImageVersion: "1312.3.0",
			CurrentShootWorkers:        fixWorkers("main-worker", "openstack.small", "gardenlinux", "1312.4.0", 1, 3, []string{"eu-de-1a", "eu-de-1b", "eu-de-1c"}),
			ExpectedShootWorkers: fixMultipleWorkers([]workerConfig{
				{"main-worker", "openstack.small", "gardenlinux", "1312.4.0", 1, 3, []string{"eu-de-1a", "eu-de-1b", "eu-de-1c"}},
				{"next-worker", "openstack.big", "gardenlinux", "1312.2.0", 1, 3, []string{"eu-de-1a", "eu-de-1b", "eu-de-1d"}}}),
			ExistingInfraConfig:        fixOpenstackInfrastructureConfig("10.250.0.0/22"),
			ExistingControlPlaneConfig: fixOpenstackControlPlaneConfig(),
			ExpectedInfraConfigCIDR:    "10.250.0.0/22",
		},
		"Add additional worker - extend existing additional worker from non HA setup to HA setup by adding more zones (use zone already specified in the existing shoot)": {
			Runtime: imv1.Runtime{
				Spec: imv1.RuntimeSpec{
					Shoot: imv1.RuntimeShoot{
						Provider: fixProviderWithMultipleWorkers(hyperscaler.TypeOpenStack, fixMultipleWorkers([]workerConfig{
							{"main-worker", "openstack.small", "gardenlinux", "1312.4.0", 1, 3, []string{"eu-de-1a", "eu-de-1b", "eu-de-1c"}},
							{"additional", "openstack.big", "gardenlinux", "1312.2.0", 1, 3, []string{"eu-de-1a", "eu-de-1b", "eu-de-1c"}},
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
				{"main-worker", "openstack.small", "gardenlinux", "1312.4.0", 1, 3, []string{"eu-de-1a", "eu-de-1b", "eu-de-1c"}},
				{"additional", "openstack.big", "gardenlinux", "1312.2.0", 1, 3, []string{"eu-de-1c"}}}),
			ExpectedShootWorkers: fixMultipleWorkers([]workerConfig{
				{"main-worker", "openstack.small", "gardenlinux", "1312.4.0", 1, 3, []string{"eu-de-1a", "eu-de-1b", "eu-de-1c"}},
				{"additional", "openstack.big", "gardenlinux", "1312.2.0", 1, 3, []string{"eu-de-1c", "eu-de-1a", "eu-de-1b"}}}),
			ExistingInfraConfig:        fixOpenstackInfrastructureConfig("10.250.0.0/22"),
			ExistingControlPlaneConfig: fixOpenstackControlPlaneConfig(),
			ExpectedInfraConfigCIDR:    "10.250.0.0/22",
		},
		"Add additional worker - extend existing additional worker from non HA setup to HA setup by adding more zones (use zone not specified in the existing shoot)": {
			Runtime: imv1.Runtime{
				Spec: imv1.RuntimeSpec{
					Shoot: imv1.RuntimeShoot{
						Provider: fixProviderWithMultipleWorkers(hyperscaler.TypeOpenStack, fixMultipleWorkers([]workerConfig{
							{"main-worker", "openstack.small", "gardenlinux", "1312.4.0", 1, 3, []string{"eu-de-1a", "eu-de-1b", "eu-de-1c"}},
							{"additional", "openstack.big", "gardenlinux", "1312.2.0", 1, 3, []string{"eu-de-1a", "eu-de-1b", "eu-de-1d"}},
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
				{"main-worker", "openstack.small", "gardenlinux", "1312.4.0", 1, 3, []string{"eu-de-1a", "eu-de-1b", "eu-de-1c"}},
				{"additional", "openstack.big", "gardenlinux", "1312.2.0", 1, 3, []string{"eu-de-1c"}}}),
			ExpectedShootWorkers: fixMultipleWorkers([]workerConfig{
				{"main-worker", "openstack.small", "gardenlinux", "1312.4.0", 1, 3, []string{"eu-de-1a", "eu-de-1b", "eu-de-1c"}},
				{"additional", "openstack.big", "gardenlinux", "1312.2.0", 1, 3, []string{"eu-de-1c", "eu-de-1a", "eu-de-1b", "eu-de-1d"}}}),
			ExistingInfraConfig:        fixOpenstackInfrastructureConfig("10.250.0.0/22"),
			ExistingControlPlaneConfig: fixOpenstackControlPlaneConfig(),
			ExpectedInfraConfigCIDR:    "10.250.0.0/22",
		},
		"Add additional worker - extend existing additional worker from non HA setup to HA setup by adding more zones (use zone not specified in the existing shoot, and infrastructure config provided in the Runtime CR)": {
			Runtime: imv1.Runtime{
				Spec: imv1.RuntimeSpec{
					Shoot: imv1.RuntimeShoot{
						Provider: fixProviderWithMultipleWorkersAndConfig(hyperscaler.TypeOpenStack, fixMultipleWorkers([]workerConfig{
							{"main-worker", "openstack.small", "gardenlinux", "1312.4.0", 1, 3, []string{"eu-de-1a", "eu-de-1b", "eu-de-1c"}},
							{"additional", "openstack.big", "gardenlinux", "1312.2.0", 1, 3, []string{"eu-de-1a", "eu-de-1b", "eu-de-1d"}},
						}), fixOpenstackInfrastructureConfig("10.250.0.0/22"), fixOpenstackControlPlaneConfig()),
						Networking: imv1.Networking{
							Nodes: "10.250.0.0/22",
						},
					},
				},
			},
			DefaultMachineImageName:    "gardenlinux",
			DefaultMachineImageVersion: "1312.3.0",
			CurrentShootWorkers: fixMultipleWorkers([]workerConfig{
				{"main-worker", "openstack.small", "gardenlinux", "1312.4.0", 1, 3, []string{"eu-de-1a", "eu-de-1b", "eu-de-1c"}},
				{"additional", "openstack.big", "gardenlinux", "1312.2.0", 1, 3, []string{"eu-de-1c"}}}),
			ExpectedShootWorkers: fixMultipleWorkers([]workerConfig{
				{"main-worker", "openstack.small", "gardenlinux", "1312.4.0", 1, 3, []string{"eu-de-1a", "eu-de-1b", "eu-de-1c"}},
				{"additional", "openstack.big", "gardenlinux", "1312.2.0", 1, 3, []string{"eu-de-1c", "eu-de-1a", "eu-de-1b", "eu-de-1d"}}}),
			ExistingInfraConfig:        fixOpenstackInfrastructureConfig("10.250.0.0/22"),
			ExistingControlPlaneConfig: fixOpenstackControlPlaneConfig(),
			ExpectedInfraConfigCIDR:    "10.250.0.0/22",
		},
		"Remove additional worker from existing set of workers": {
			Runtime: imv1.Runtime{
				Spec: imv1.RuntimeSpec{
					Shoot: imv1.RuntimeShoot{
						Provider: fixProviderWithMultipleWorkers(hyperscaler.TypeOpenStack, fixMultipleWorkers([]workerConfig{
							{"main-worker", "openstack.small", "gardenlinux", "1312.4.0", 1, 3, []string{"eu-de-1a", "eu-de-1b", "eu-de-1c"}}})),
						Networking: imv1.Networking{
							Nodes: "10.250.0.0/22",
						},
					},
				},
			},
			DefaultMachineImageName:    "gardenlinux",
			DefaultMachineImageVersion: "1312.3.0",
			CurrentShootWorkers: fixMultipleWorkers([]workerConfig{
				{"main-worker", "openstack.small", "gardenlinux", "1312.4.0", 1, 3, []string{"eu-de-1a", "eu-de-1b", "eu-de-1c"}},
				{"next-worker", "openstack.large", "gardenlinux", "1312.2.0", 2, 4, []string{"eu-de-1a", "eu-de-1b", "eu-de-1c"}}}),
			ExpectedShootWorkers: fixMultipleWorkers([]workerConfig{
				{"main-worker", "openstack.small", "gardenlinux", "1312.4.0", 1, 3, []string{"eu-de-1a", "eu-de-1b", "eu-de-1c"}}}),
			ExistingInfraConfig:        fixOpenstackInfrastructureConfig("10.250.0.0/22"),
			ExistingControlPlaneConfig: fixOpenstackControlPlaneConfig(),
			ExpectedInfraConfigCIDR:    "10.250.0.0/22",
		},
		"Update machine type and image name and version in multiple workers separately": {
			Runtime: imv1.Runtime{
				Spec: imv1.RuntimeSpec{
					Shoot: imv1.RuntimeShoot{
						Provider: fixProviderWithMultipleWorkers(hyperscaler.TypeOpenStack, fixMultipleWorkers([]workerConfig{
							{"main-worker", "openstack.small", "gardenlinux", "1313.4.0", 1, 3, []string{"eu-de-1a", "eu-de-1b", "eu-de-1c"}},
							{"next-worker", "openstack.large", "gardenlinux", "1313.2.0", 1, 3, []string{"eu-de-1a", "eu-de-1b", "eu-de-1c"}},
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
				{"main-worker", "openstack.small", "gardenlinux", "1312.4.0", 1, 3, []string{"eu-de-1a", "eu-de-1b", "eu-de-1c"}},
				{"next-worker", "openstack.large", "gardenlinux", "1312.2.0", 1, 3, []string{"eu-de-1a", "eu-de-1b", "eu-de-1c"}}}),
			ExpectedShootWorkers: fixMultipleWorkers([]workerConfig{
				{"main-worker", "openstack.small", "gardenlinux", "1313.4.0", 1, 3, []string{"eu-de-1a", "eu-de-1b", "eu-de-1c"}},
				{"next-worker", "openstack.large", "gardenlinux", "1313.2.0", 1, 3, []string{"eu-de-1a", "eu-de-1b", "eu-de-1c"}}}),
			ExistingInfraConfig:        fixOpenstackInfrastructureConfig("10.250.0.0/22"),
			ExistingControlPlaneConfig: fixOpenstackControlPlaneConfig(),
			ExpectedInfraConfigCIDR:    "10.250.0.0/22",
		},
		"Remove worker from existing set of workers networking zones set in infrastructureConfig should not change": {
			Runtime: imv1.Runtime{
				Spec: imv1.RuntimeSpec{
					Shoot: imv1.RuntimeShoot{
						Provider: fixProviderWithMultipleWorkers(hyperscaler.TypeOpenStack, fixMultipleWorkers([]workerConfig{
							{"main-worker", "openstack.small", "gardenlinux", "1313.4.0", 1, 3, []string{"eu-de-1a"}},
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
				{"main-worker", "openstack.small", "gardenlinux", "1312.4.0", 1, 3, []string{"eu-de-1a"}},
				{"additional", "openstack.small", "gardenlinux", "1312.2.0", 1, 3, []string{"eu-de-1a", "eu-de-1b", "eu-de-1c"}}}),
			ExpectedShootWorkers: fixMultipleWorkers([]workerConfig{
				{"main-worker", "openstack.small", "gardenlinux", "1313.4.0", 1, 3, []string{"eu-de-1a"}}}),
			ExistingInfraConfig:        fixOpenstackInfrastructureConfig("10.250.0.0/22"),
			ExistingControlPlaneConfig: fixOpenstackControlPlaneConfig(),
			ExpectedInfraConfigCIDR:    "10.250.0.0/22",
		},
		"Modify infrastructure config with value provided externally with own CIDR": {
			Runtime: imv1.Runtime{
				Spec: imv1.RuntimeSpec{
					Shoot: imv1.RuntimeShoot{
						Provider: fixProviderWithMultipleWorkersAndConfig(hyperscaler.TypeOpenStack, fixMultipleWorkers([]workerConfig{
							{"main-worker", "openstack.small", "gardenlinux", "1313.4.0", 1, 3, []string{"eu-de-1a"}},
							{"next-worker", "openstack.big", "gardenlinux", "1313.2.0", 1, 3, []string{"eu-de-1a", "eu-de-1b"}},
						}), fixOpenstackInfrastructureConfig("10.250.0.0/16"), fixOpenstackControlPlaneConfig()),
						Networking: imv1.Networking{
							Nodes: "10.250.0.0/22",
						},
					},
				},
			},
			DefaultMachineImageName:    "gardenlinux",
			DefaultMachineImageVersion: "1312.3.0",
			CurrentShootWorkers: fixMultipleWorkers([]workerConfig{
				{"main-worker", "openstack.small", "gardenlinux", "1312.4.0", 1, 3, []string{"eu-de-1a"}},
				{"next-worker", "openstack.big", "gardenlinux", "1312.2.0", 1, 3, []string{"eu-de-1a", "eu-de-1b"}}}),
			ExpectedShootWorkers: fixMultipleWorkers([]workerConfig{
				{"main-worker", "openstack.small", "gardenlinux", "1313.4.0", 1, 3, []string{"eu-de-1a"}},
				{"next-worker", "openstack.big", "gardenlinux", "1313.2.0", 1, 3, []string{"eu-de-1a", "eu-de-1b"}}}),
			ExistingInfraConfig:        fixOpenstackInfrastructureConfig("10.250.0.0/22"),
			ExistingControlPlaneConfig: fixOpenstackControlPlaneConfig(),
			ExpectedInfraConfigCIDR:    "10.250.0.0/16",
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
			assertProviderSpecificConfigOpenstack(t, shoot, tc.ExpectedInfraConfigCIDR)
		})
	}
}

func fixOpenstackInfrastructureConfig(workersCIDR string) *runtime.RawExtension {
	infraConfig, _ := ops.GetInfrastructureConfig(workersCIDR, []string{})
	return &runtime.RawExtension{Raw: infraConfig}
}

func fixOpenstackControlPlaneConfig() *runtime.RawExtension {
	controlPlaneConfig, _ := ops.GetControlPlaneConfig([]string{})
	return &runtime.RawExtension{Raw: controlPlaneConfig}
}

func assertProviderSpecificConfigOpenstack(t *testing.T, shoot gardener.Shoot, expectedWorkersCIDR string) {
	var ctrlPlaneConfig ostext.ControlPlaneConfig
	var infraConfig ostext.InfrastructureConfig
	//
	err := json.Unmarshal(shoot.Spec.Provider.ControlPlaneConfig.Raw, &ctrlPlaneConfig)
	require.NoError(t, err)
	assert.Equal(t, ctrlPlaneConfig.LoadBalancerProvider, "f5")

	err = json.Unmarshal(shoot.Spec.Provider.InfrastructureConfig.Raw, &infraConfig)
	require.NoError(t, err)
	assert.Equal(t, infraConfig.Networks.Workers, expectedWorkersCIDR)
}
