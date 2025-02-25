package extender

import (
	"encoding/json"
	gcpext "github.com/gardener/gardener-extension-provider-gcp/pkg/apis/gcp/v1alpha1"
	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/extender/testutils"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/hyperscaler"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/hyperscaler/gcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	"testing"
)

func TestProviderExtenderForCreateGCP(t *testing.T) {
	// tests of ProviderExtenderForCreateOperation for GCP provider
	for tname, tc := range map[string]struct {
		Runtime                     imv1.Runtime
		DefaultMachineImageVersion  string
		ExpectedMachineImageVersion string
		DefaultMachineImageName     string
		ExpectedMachineImageName    string
		CurrentZonesConfig          []string
		ExpectedZonesCount          int
	}{
		"Create provider specific config for GCP without worker config and one zone": {
			Runtime: imv1.Runtime{
				Spec: imv1.RuntimeSpec{
					Shoot: imv1.RuntimeShoot{
						Provider: fixProvider(hyperscaler.TypeGCP, "ubuntu", "18.04", []string{"us-central1-a"}),
						Networking: imv1.Networking{
							Nodes: "10.250.0.0/22",
						},
					},
				},
			},
			DefaultMachineImageVersion:  "18.04-LTS",
			ExpectedMachineImageVersion: "18.04",
			ExpectedMachineImageName:    "ubuntu",
		},
		"Create provider specific config for GCP without worker config and two zones": {
			Runtime: imv1.Runtime{
				Spec: imv1.RuntimeSpec{
					Shoot: imv1.RuntimeShoot{
						Provider: fixProvider(hyperscaler.TypeGCP, "ubuntu", "18.04", []string{"us-central1-a", "us-central1-b"}),
						Networking: imv1.Networking{
							Nodes: "10.250.0.0/22",
						},
					},
				},
			},
			DefaultMachineImageVersion:  "18.04-LTS",
			ExpectedMachineImageVersion: "18.04",
			ExpectedMachineImageName:    "ubuntu",
		},
		"Create provider specific config for GCP without worker config and three zones": {
			Runtime: imv1.Runtime{
				Spec: imv1.RuntimeSpec{
					Shoot: imv1.RuntimeShoot{
						Provider: fixProvider(hyperscaler.TypeGCP, "ubuntu", "18.04", []string{"us-central1-a", "us-central1-b", "us-central1-c"}),
						Networking: imv1.Networking{
							Nodes: "10.250.0.0/22",
						},
					},
				},
			},
			DefaultMachineImageVersion:  "18.04-LTS",
			ExpectedMachineImageVersion: "18.04",
			ExpectedMachineImageName:    "ubuntu",
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

			assertProviderSpecificConfigGCP(t, shoot)
		})
	}
}

func TestProviderExtenderForPatchWorkersUpdateGCP(t *testing.T) {
	// tests of NewProviderExtenderPatch for workers update operation for GCP provider
	for tname, tc := range map[string]struct {
		Runtime                    imv1.Runtime
		DefaultMachineImageVersion string
		DefaultMachineImageName    string
		CurrentShootWorkers        []gardener.Worker
		ExistingInfraConfig        *runtime.RawExtension
		ExistingControlPlaneConfig *runtime.RawExtension
		ExpectedShootWorkers       []gardener.Worker
	}{
		"Add additional worker": {
			Runtime: imv1.Runtime{
				Spec: imv1.RuntimeSpec{
					Shoot: imv1.RuntimeShoot{
						Provider: fixProviderWithMultipleWorkers(hyperscaler.TypeGCP, fixMultipleWorkers([]workerConfig{
							{"main-worker", "n2-standard-2", "gardenlinux", "1312.4.0", 1, 3, []string{"us-central1-a", "us-central1-b", "us-central1-c"}},
							{"next-worker", "n2-standard-2", "gardenlinux", "1312.2.0", 1, 3, []string{"us-central1-a", "us-central1-b", "us-central1-c"}},
						})),
						Networking: imv1.Networking{
							Nodes: "10.250.0.0/22",
						},
					},
				},
			},
			DefaultMachineImageName:    "gardenlinux",
			DefaultMachineImageVersion: "1312.3.0",
			CurrentShootWorkers:        fixWorkers("main-worker", "m6i.large", "gardenlinux", "1312.4.0", 1, 3, []string{"us-central1-a", "us-central1-b", "us-central1-c"}),
			ExpectedShootWorkers: fixMultipleWorkers([]workerConfig{
				{"main-worker", "n2-standard-2", "gardenlinux", "1312.4.0", 1, 3, []string{"us-central1-a", "us-central1-b", "us-central1-c"}},
				{"next-worker", "n2-standard-2", "gardenlinux", "1312.2.0", 1, 3, []string{"us-central1-a", "us-central1-b", "us-central1-c"}}}),
			ExistingInfraConfig:        fixGCPInfrastructureConfig("10.250.0.0/22"),
			ExistingControlPlaneConfig: fixGCPControlPlaneConfig([]string{"us-central1-a", "us-central1-b", "us-central1-c"}),
		},
		// NOTE: uncomment this unit test after the issue of extending the existing infrastructure config with additional zones is implemented
		"Add additional worker - extend existing additional worker from non HA setup to HA setup by adding more zones, infrastructureConfig already has three zones, keep existing order": {
			Runtime: imv1.Runtime{
				Spec: imv1.RuntimeSpec{
					Shoot: imv1.RuntimeShoot{
						Provider: fixProviderWithMultipleWorkers(hyperscaler.TypeGCP, fixMultipleWorkers([]workerConfig{
							{"main-worker", "n2-standard-2", "gardenlinux", "1312.4.0", 1, 3, []string{"us-central1-a", "us-central1-b", "us-central1-c"}},
							{"additional", "n2-standard-2", "gardenlinux", "1312.2.0", 1, 3, []string{"us-central1-a", "us-central1-b", "us-central1-c"}},
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
				{"main-worker", "n2-standard-2", "gardenlinux", "1312.4.0", 1, 3, []string{"us-central1-a", "us-central1-b", "us-central1-c"}},
				{"additional", "n2-standard-2", "gardenlinux", "1312.2.0", 1, 3, []string{"us-central1-b", "us-central1-a"}}}),
			ExpectedShootWorkers: fixMultipleWorkers([]workerConfig{
				{"main-worker", "n2-standard-2", "gardenlinux", "1312.4.0", 1, 3, []string{"us-central1-a", "us-central1-b", "us-central1-c"}},
				{"additional", "n2-standard-2", "gardenlinux", "1312.2.0", 1, 3, []string{"us-central1-b", "us-central1-a", "us-central1-c"}}}),
			ExistingInfraConfig:        fixGCPInfrastructureConfig("10.250.0.0/22"),
			ExistingControlPlaneConfig: fixGCPControlPlaneConfig([]string{"us-central1-a", "us-central1-b", "us-central1-c"}),
		},
		"Remove additional worker from existing set of workers": {
			Runtime: imv1.Runtime{
				Spec: imv1.RuntimeSpec{
					Shoot: imv1.RuntimeShoot{
						Provider: fixProviderWithMultipleWorkers(hyperscaler.TypeGCP, fixMultipleWorkers([]workerConfig{
							{"main-worker", "n2-standard-2", "gardenlinux", "1312.4.0", 1, 3, []string{"us-central1-a", "us-central1-b", "us-central1-c"}}})),
						Networking: imv1.Networking{
							Nodes: "10.250.0.0/22",
						},
					},
				},
			},
			DefaultMachineImageName:    "gardenlinux",
			DefaultMachineImageVersion: "1312.3.0",
			CurrentShootWorkers: fixMultipleWorkers([]workerConfig{
				{"main-worker", "n2-standard-2", "gardenlinux", "1312.4.0", 1, 3, []string{"us-central1-a", "us-central1-b", "us-central1-c"}},
				{"next-worker", "n2-standard-2", "gardenlinux", "1312.2.0", 2, 4, []string{"us-central1-a", "us-central1-b", "us-central1-c"}}}),
			ExpectedShootWorkers: fixMultipleWorkers([]workerConfig{
				{"main-worker", "n2-standard-2", "gardenlinux", "1312.4.0", 1, 3, []string{"us-central1-a", "us-central1-b", "us-central1-c"}}}),
			ExistingInfraConfig:        fixGCPInfrastructureConfig("10.250.0.0/22"),
			ExistingControlPlaneConfig: fixGCPControlPlaneConfig([]string{"us-central1-a", "us-central1-b", "us-central1-c"}),
		},
		"Update machine type and image name and version in multiple workers separately": {
			Runtime: imv1.Runtime{
				Spec: imv1.RuntimeSpec{
					Shoot: imv1.RuntimeShoot{
						Provider: fixProviderWithMultipleWorkers(hyperscaler.TypeGCP, fixMultipleWorkers([]workerConfig{
							{"main-worker", "n2-standard-4", "gardenlinux", "1313.4.0", 1, 3, []string{"us-central1-a", "us-central1-b", "us-central1-c"}},
							{"additional", "n2-standard-4", "gardenlinux", "1313.2.0", 1, 3, []string{"us-central1-a", "us-central1-b", "us-central1-c"}},
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
				{"main-worker", "n2-standard-2", "gardenlinux", "1312.4.0", 1, 3, []string{"us-central1-a", "us-central1-b", "us-central1-c"}},
				{"additional", "n2-standard-2", "gardenlinux", "1312.2.0", 1, 3, []string{"us-central1-a", "us-central1-b", "us-central1-c"}}}),
			ExpectedShootWorkers: fixMultipleWorkers([]workerConfig{
				{"main-worker", "n2-standard-4", "gardenlinux", "1313.4.0", 1, 3, []string{"us-central1-a", "us-central1-b", "us-central1-c"}},
				{"additional", "n2-standard-4", "gardenlinux", "1313.2.0", 1, 3, []string{"us-central1-a", "us-central1-b", "us-central1-c"}}}),
			ExistingInfraConfig:        fixGCPInfrastructureConfig("10.250.0.0/22"),
			ExistingControlPlaneConfig: fixGCPControlPlaneConfig([]string{"us-central1-a", "us-central1-b", "us-central1-c"}),
		},
		"Remove worker from existing set of workers networking zones set in infrastructureConfig should not change": {
			Runtime: imv1.Runtime{
				Spec: imv1.RuntimeSpec{
					Shoot: imv1.RuntimeShoot{
						Provider: fixProviderWithMultipleWorkers(hyperscaler.TypeGCP, fixMultipleWorkers([]workerConfig{
							{"main-worker", "n2-standard-2", "gardenlinux", "1313.4.0", 1, 3, []string{"us-central1-a"}},
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
				{"main-worker", "n2-standard-2", "gardenlinux", "1312.4.0", 1, 3, []string{"us-central1-a"}},
				{"additional", "n2-standard-2", "gardenlinux", "1312.2.0", 1, 3, []string{"us-central1-a", "us-central1-b", "us-central1-c"}}}),
			ExpectedShootWorkers: fixMultipleWorkers([]workerConfig{
				{"main-worker", "n2-standard-2", "gardenlinux", "1313.4.0", 1, 3, []string{"us-central1-a"}}}),
			ExistingInfraConfig:        fixGCPInfrastructureConfig("10.250.0.0/22"),
			ExistingControlPlaneConfig: fixGCPControlPlaneConfig([]string{"us-central1-a", "us-central1-b", "us-central1-c"}),
		},
		"Modify infrastructure config with value provided externally with 3 zones": {
			Runtime: imv1.Runtime{
				Spec: imv1.RuntimeSpec{
					Shoot: imv1.RuntimeShoot{
						Provider: fixProviderWithMultipleWorkersAndConfig(hyperscaler.TypeGCP, fixMultipleWorkers([]workerConfig{
							{"main-worker", "n2-standard-2", "gardenlinux", "1313.4.0", 1, 3, []string{"us-central1-a"}},
							{"next-worker", "n2-standard-3", "gardenlinux", "1313.2.0", 1, 3, []string{"us-central1-a", "us-central1-b"}},
						}), fixGCPInfrastructureConfig("10.250.0.0/22"), fixGCPControlPlaneConfig([]string{"us-central1-a", "us-central1-b", "us-central1-c"})),
						Networking: imv1.Networking{
							Nodes: "10.250.0.0/22",
						},
					},
				},
			},
			DefaultMachineImageName:    "gardenlinux",
			DefaultMachineImageVersion: "1312.3.0",
			CurrentShootWorkers: fixMultipleWorkers([]workerConfig{
				{"main-worker", "n2-standard-2", "gardenlinux", "1312.4.0", 1, 3, []string{"us-central1-a"}},
				{"next-worker", "n2-standard-3", "gardenlinux", "1312.2.0", 1, 3, []string{"us-central1-a", "us-central1-b"}}}),
			ExpectedShootWorkers: fixMultipleWorkers([]workerConfig{
				{"main-worker", "n2-standard-2", "gardenlinux", "1313.4.0", 1, 3, []string{"us-central1-a"}},
				{"next-worker", "n2-standard-3", "gardenlinux", "1313.2.0", 1, 3, []string{"us-central1-a", "us-central1-b"}}}),
			ExistingInfraConfig:        fixGCPInfrastructureConfig("10.250.0.0/22"),
			ExistingControlPlaneConfig: fixGCPControlPlaneConfig([]string{"us-central1-a", "us-central1-b", "us-central1-c"}),
		},
	} {
		t.Run(tname, func(t *testing.T) {
			// given
			shoot := testutils.FixEmptyGardenerShoot("cluster", "kcp-system")

			// when
			extender := NewProviderExtenderPatchOperation(false, tc.DefaultMachineImageName, tc.DefaultMachineImageVersion, tc.CurrentShootWorkers, tc.ExistingInfraConfig, tc.ExistingControlPlaneConfig, nil)
			err := extender(tc.Runtime, &shoot)

			// then
			require.NoError(t, err)

			assertProviderMultipleWorkers(t, tc.Runtime.Spec.Shoot, shoot, false, tc.ExpectedShootWorkers)
			assertProviderSpecificConfigGCP(t, shoot)
		})
	}
}

func TestProviderExtenderForCreateMultipleWorkersGCP(t *testing.T) {
	// tests of NewProviderExtenderForCreateOperation for workers create operation
	for tname, tc := range map[string]struct {
		Runtime                    imv1.Runtime
		DefaultMachineImageVersion string
		DefaultMachineImageName    string
		CurrentShootWorkers        []gardener.Worker
		ExistingInfraConfig        *runtime.RawExtension
		ExistingControlPlaneConfig *runtime.RawExtension
		ExpectedShootWorkers       []gardener.Worker
	}{
		"Create multiple GCP workers without worker config": {
			Runtime: imv1.Runtime{
				Spec: imv1.RuntimeSpec{
					Shoot: imv1.RuntimeShoot{
						Provider: fixProviderWithMultipleWorkers(hyperscaler.TypeGCP, fixMultipleWorkers([]workerConfig{
							{"main-worker", "gcp.large", "gardenlinux", "1310.4.0", 1, 3, []string{"us-central1-a"}},
							{"additional", "gcp.large", "gardenlinux", "1311.2.0", 2, 4, []string{"us-central1-b", "us-central1-c"}},
							{"another", "gcp.large", "gardenlinux", "1312.2.0", 3, 5, []string{"us-central1-c"}},
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
				{"main-worker", "gcp.large", "gardenlinux", "1310.4.0", 1, 3, []string{"us-central1-a"}},
				{"additional", "gcp.large", "gardenlinux", "1311.2.0", 2, 4, []string{"us-central1-b", "us-central1-c"}},
				{"another", "gcp.large", "gardenlinux", "1312.2.0", 3, 5, []string{"us-central1-c"}}}),
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
			assertProviderSpecificConfigGCP(t, shoot)
		})
	}
}

func fixGCPInfrastructureConfig(workersCIDR string) *runtime.RawExtension {
	infraConfig, _ := gcp.GetInfrastructureConfig(workersCIDR, []string{})
	return &runtime.RawExtension{Raw: infraConfig}
}

func fixGCPControlPlaneConfig(zones []string) *runtime.RawExtension {
	controlPlaneConfig, _ := gcp.GetControlPlaneConfig(zones)
	return &runtime.RawExtension{Raw: controlPlaneConfig}
}

func assertProviderSpecificConfigGCP(t *testing.T, shoot gardener.Shoot) {
	var ctrlPlaneConfig gcpext.ControlPlaneConfig

	err := json.Unmarshal(shoot.Spec.Provider.ControlPlaneConfig.Raw, &ctrlPlaneConfig)
	require.NoError(t, err)
	assert.NotEmpty(t, ctrlPlaneConfig.Zone)
}
