package extender

import (
	"encoding/json"
	"github.com/gardener/gardener-extension-provider-gcp/pkg/apis/gcp"
	aws "github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/hyperscaler/aws"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/hyperscaler/azure"
	"k8s.io/apimachinery/pkg/runtime"

	"testing"

	"github.com/gardener/gardener-extension-provider-aws/pkg/apis/aws/v1alpha1"
	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/hyperscaler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProviderExtenderForCreateAWS(t *testing.T) {
	// tests of ProviderExtenderForCreateOperation for AWS provider
	for tname, tc := range map[string]struct {
		Runtime                     imv1.Runtime
		EnableIMDSv2                bool
		DefaultMachineImageVersion  string
		ExpectedMachineImageVersion string
		DefaultMachineImageName     string
		ExpectedMachineImageName    string
		CurrentZonesConfig          []string
		ExpectedZonesCount          int
	}{
		"Create provider specific config for AWS without worker config and one zone": {
			Runtime: imv1.Runtime{
				Spec: imv1.RuntimeSpec{
					Shoot: imv1.RuntimeShoot{
						Provider: fixProvider(hyperscaler.TypeAWS, "gardenlinux", "1312.2.0", []string{"eu-central-1a"}),
					},
				},
			},
			EnableIMDSv2:                false,
			DefaultMachineImageVersion:  "1312.3.0",
			ExpectedMachineImageVersion: "1312.2.0",
			ExpectedMachineImageName:    "gardenlinux",
			ExpectedZonesCount:          1,
		},
		"Create provider specific config for AWS without worker config and two zones": {
			Runtime: imv1.Runtime{
				Spec: imv1.RuntimeSpec{
					Shoot: imv1.RuntimeShoot{
						Provider: fixProvider(hyperscaler.TypeAWS, "gardenlinux", "1312.2.0", []string{"eu-central-1a", "eu-central-1b"}),
					},
				},
			},
			EnableIMDSv2:                false,
			DefaultMachineImageVersion:  "1312.3.0",
			ExpectedMachineImageVersion: "1312.2.0",
			ExpectedMachineImageName:    "gardenlinux",
			ExpectedZonesCount:          2,
		},
		"Create provider specific config for AWS without worker config and three zones": {
			Runtime: imv1.Runtime{
				Spec: imv1.RuntimeSpec{
					Shoot: imv1.RuntimeShoot{
						Provider: fixProvider(hyperscaler.TypeAWS, "gardenlinux", "1312.2.0", []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}),
					},
				},
			},
			EnableIMDSv2:                false,
			DefaultMachineImageVersion:  "1312.3.0",
			ExpectedMachineImageVersion: "1312.2.0",
			ExpectedMachineImageName:    "gardenlinux",
			ExpectedZonesCount:          3,
		},
		"Create provider specific config for AWS with worker config and three zones": {
			Runtime: imv1.Runtime{
				Spec: imv1.RuntimeSpec{
					Shoot: imv1.RuntimeShoot{
						Provider: fixProvider(hyperscaler.TypeAWS, "", "", []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}),
					},
				},
			},
			EnableIMDSv2:                true,
			DefaultMachineImageVersion:  "1312.3.0",
			ExpectedMachineImageVersion: "1312.3.0",
			ExpectedZonesCount:          3,
		},
		"Create provider specific config for AWS with multiple workers - create option": {
			Runtime: imv1.Runtime{
				Spec: imv1.RuntimeSpec{
					Shoot: imv1.RuntimeShoot{
						Provider: fixProviderWithMultipleWorkers(hyperscaler.TypeAWS, []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}),
					},
				},
			},
			EnableIMDSv2:                false,
			DefaultMachineImageVersion:  "1312.3.0",
			ExpectedMachineImageVersion: "1312.3.0",
			ExpectedZonesCount:          3,
		},
	} {
		t.Run(tname, func(t *testing.T) {
			// given
			shoot := fixEmptyGardenerShoot("cluster", "kcp-system")

			// when

			extender := NewProviderExtenderForCreateOperation(tc.EnableIMDSv2, tc.DefaultMachineImageName, tc.DefaultMachineImageVersion)
			err := extender(tc.Runtime, &shoot)

			// then
			require.NoError(t, err)

			assertProvider(t, tc.Runtime.Spec.Shoot, shoot, tc.EnableIMDSv2, tc.ExpectedMachineImageName, tc.ExpectedMachineImageVersion)
			assertProviderSpecificConfigAWS(t, shoot, tc.ExpectedZonesCount)
		})
	}

	t.Run("Return error for unknown provider", func(t *testing.T) {
		// given
		shoot := fixEmptyGardenerShoot("cluster", "kcp-system")
		runtime := imv1.Runtime{
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
		err := extender(runtime, &shoot)

		// then
		require.Error(t, err)
	})
}

func TestProviderExtenderForPatchAWS(t *testing.T) {
	// tests of NewProviderExtenderPatchOperation for AWS only for provider image version patching
	for tname, tc := range map[string]struct {
		Runtime                     imv1.Runtime
		EnableIMDSv2                bool
		DefaultMachineImageVersion  string
		ExpectedMachineImageVersion string
		DefaultMachineImageName     string
		ExpectedMachineImageName    string
		CurrentShootWorkers         []gardener.Worker
		ExistingInfraConfig         *runtime.RawExtension
		ExistingControlPlaneConfig  *runtime.RawExtension
		ExpectedShootWorkersCount   int
		ExpectedZonesCount          int
	}{
		"Same image name - use bigger current shoot machine image version as image version": {
			Runtime: imv1.Runtime{
				Spec: imv1.RuntimeSpec{
					Shoot: imv1.RuntimeShoot{
						Provider: fixProvider(hyperscaler.TypeAWS, "gardenlinux", "1312.2.0", []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}),
						Networking: imv1.Networking{
							Nodes: "10.250.0.0/22",
						},
					},
				},
			},
			EnableIMDSv2:                false,
			DefaultMachineImageName:     "gardenlinux",
			DefaultMachineImageVersion:  "1312.3.0",
			ExpectedMachineImageVersion: "1312.4.0",
			ExpectedZonesCount:          3,
			ExpectedMachineImageName:    "gardenlinux",
			CurrentShootWorkers:         fixWorkers("m6i.large", "gardenlinux", "1312.4.0", 1, 3, []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}),
			ExistingInfraConfig:         fixAWSInfrastructureConfig("10.250.0.0/22", []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}),
			ExistingControlPlaneConfig:  fixAWSControlPlaneConfig(),
		},
		"Same image name - override current shoot machine image version with new bigger version from RuntimeCR": {
			Runtime: imv1.Runtime{
				Spec: imv1.RuntimeSpec{
					Shoot: imv1.RuntimeShoot{
						Provider: fixProvider(hyperscaler.TypeAWS, "gardenlinux", "1312.2.0", []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}),
					},
				},
			},
			EnableIMDSv2:                false,
			DefaultMachineImageName:     "gardenlinux",
			DefaultMachineImageVersion:  "1312.3.0",
			CurrentShootWorkers:         fixWorkers("m6i.large", "gardenlinux", "1312.1.0", 1, 3, []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}),
			ExpectedMachineImageVersion: "1312.2.0",
			ExpectedZonesCount:          3,
			ExpectedMachineImageName:    "gardenlinux",
			ExistingInfraConfig:         fixAWSInfrastructureConfig("10.250.0.0/22", []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}),
			ExistingControlPlaneConfig:  fixAWSControlPlaneConfig(),
		},
		"Same image name - no version is provided override current shoot machine image version with default version": {
			Runtime: imv1.Runtime{
				Spec: imv1.RuntimeSpec{
					Shoot: imv1.RuntimeShoot{
						Provider: fixProvider(hyperscaler.TypeAWS, "", "", []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}),
					},
				},
			},
			EnableIMDSv2:                false,
			DefaultMachineImageName:     "gardenlinux",
			DefaultMachineImageVersion:  "1312.3.0",
			CurrentShootWorkers:         fixWorkers("m6i.large", "gardenlinux", "1312.1.0", 1, 3, []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}),
			ExpectedMachineImageVersion: "1312.3.0",
			ExpectedZonesCount:          3,
			ExpectedMachineImageName:    "gardenlinux",
			ExistingInfraConfig:         fixAWSInfrastructureConfig("10.250.0.0/22", []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}),
			ExistingControlPlaneConfig:  fixAWSControlPlaneConfig(),
		},
		"Different image name - override current shoot machine image and version with new data from RuntimeCR": {
			Runtime: imv1.Runtime{
				Spec: imv1.RuntimeSpec{
					Shoot: imv1.RuntimeShoot{
						Provider: fixProvider(hyperscaler.TypeAWS, "gardenlinux", "1312.2.0", []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}),
					},
				},
			},
			EnableIMDSv2:                false,
			DefaultMachineImageName:     "gardenlinux",
			DefaultMachineImageVersion:  "1312.3.0",
			CurrentShootWorkers:         fixWorkers("m6i.large", "ubuntu", "1312.4.0", 1, 3, []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}),
			ExpectedZonesCount:          3,
			ExpectedMachineImageName:    "gardenlinux",
			ExpectedMachineImageVersion: "1312.2.0",
			ExistingInfraConfig:         fixAWSInfrastructureConfig("10.250.0.0/22", []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}),
			ExistingControlPlaneConfig:  fixAWSControlPlaneConfig(),
		},
		"Different image name - no data is provided override current shoot machine image and version with default data": {
			Runtime: imv1.Runtime{
				Spec: imv1.RuntimeSpec{
					Shoot: imv1.RuntimeShoot{
						Provider: fixProvider(hyperscaler.TypeAWS, "", "", []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}),
					},
				},
			},
			EnableIMDSv2:                false,
			DefaultMachineImageName:     "gardenlinux",
			DefaultMachineImageVersion:  "1312.3.0",
			CurrentShootWorkers:         fixWorkers("m6i.large", "ubuntu", "1312.4.0", 1, 3, []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}),
			ExpectedMachineImageVersion: "1312.3.0",
			ExpectedZonesCount:          3,
			ExpectedMachineImageName:    "gardenlinux",
			ExistingInfraConfig:         fixAWSInfrastructureConfig("10.250.0.0/22", []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}),
			ExistingControlPlaneConfig:  fixAWSControlPlaneConfig(),
		},
		"Wrong current image name - use data from RuntimeCR": {
			Runtime: imv1.Runtime{
				Spec: imv1.RuntimeSpec{
					Shoot: imv1.RuntimeShoot{
						Provider: fixProvider(hyperscaler.TypeAWS, "gardenlinux", "1312.2.0", []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}),
					},
				},
			},
			EnableIMDSv2:                false,
			DefaultMachineImageName:     "gardenlinux",
			DefaultMachineImageVersion:  "1312.3.0",
			CurrentShootWorkers:         fixWorkers("m6i.large", "", "1312.4.0", 1, 3, []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}),
			ExpectedMachineImageVersion: "1312.2.0",
			ExpectedZonesCount:          3,
			ExpectedMachineImageName:    "gardenlinux",
			ExistingInfraConfig:         fixAWSInfrastructureConfig("10.250.0.0/22", []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}),
			ExistingControlPlaneConfig:  fixAWSControlPlaneConfig(),
		},
		"Wrong current image version - use data from RuntimeCR": {
			Runtime: imv1.Runtime{
				Spec: imv1.RuntimeSpec{
					Shoot: imv1.RuntimeShoot{
						Provider: fixProvider(hyperscaler.TypeAWS, "gardenlinux", "1312.2.0", []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}),
					},
				},
			},
			EnableIMDSv2:                false,
			DefaultMachineImageName:     "gardenlinux",
			DefaultMachineImageVersion:  "1312.3.0",
			CurrentShootWorkers:         fixWorkers("m6i.large", "gardenlinux", "", 1, 3, []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}),
			ExpectedMachineImageVersion: "1312.2.0",
			ExpectedZonesCount:          3,
			ExpectedMachineImageName:    "gardenlinux",
			ExistingInfraConfig:         fixAWSInfrastructureConfig("10.250.0.0/22", []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}),
			ExistingControlPlaneConfig:  fixAWSControlPlaneConfig(),
		},
	} {
		t.Run(tname, func(t *testing.T) {
			// given
			shoot := fixEmptyGardenerShoot("cluster", "kcp-system")

			// when
			extender := NewProviderExtenderPatchOperation(tc.EnableIMDSv2, tc.DefaultMachineImageName, tc.DefaultMachineImageVersion, tc.CurrentShootWorkers, tc.ExistingControlPlaneConfig, tc.ExistingInfraConfig)
			err := extender(tc.Runtime, &shoot)

			// then
			require.NoError(t, err)

			assertProvider(t, tc.Runtime.Spec.Shoot, shoot, tc.EnableIMDSv2, tc.ExpectedMachineImageName, tc.ExpectedMachineImageVersion)
			assertProviderSpecificConfigAWS(t, shoot, tc.ExpectedZonesCount)
		})
	}
}

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
		"Create provider specific config for Azure with worker config and three zones": {
			Runtime: imv1.Runtime{
				Spec: imv1.RuntimeSpec{
					Shoot: imv1.RuntimeShoot{
						Provider: fixProvider(hyperscaler.TypeAzure, "", "", []string{"1", "2", "3"}),
						Networking: imv1.Networking{
							Nodes: "10.250.0.0/22",
						},
					},
				},
			},
			DefaultMachineImageVersion:  "18.04-LTS",
			ExpectedMachineImageVersion: "18.04-LTS",
			ExpectedZonesCount:          3,
		},
		"Create provider specific config for Azure with multiple workers - create option": {
			Runtime: imv1.Runtime{
				Spec: imv1.RuntimeSpec{
					Shoot: imv1.RuntimeShoot{
						Provider: fixProviderWithMultipleWorkers(hyperscaler.TypeAzure, []string{"1", "2", "3"}),
						Networking: imv1.Networking{
							Nodes: "10.250.0.0/22",
						},
					},
				},
			},
			DefaultMachineImageVersion:  "18.04-LTS",
			ExpectedMachineImageVersion: "18.04-LTS",
			ExpectedZonesCount:          3,
		},
	} {
		t.Run(tname, func(t *testing.T) {
			// given
			shoot := fixEmptyGardenerShoot("cluster", "kcp-system")

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

/*func TestProviderExtenderForCreateGCP(t *testing.T) {

}

func TestProviderExtenderForCreateOpenStack(t *testing.T) {

}*/

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

func fixWorkers(machineType, machineImageName, machineImageVersion string, min, max int32, zones []string) []gardener.Worker {
	return []gardener.Worker{
		{
			Name: "worker",
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

func fixProviderWithInfrastructureConfig(providerType string, machineImageName, machineImageVersion string, zones []string) imv1.Provider {
	var infraConfigBytes []byte
	switch providerType {
	case hyperscaler.TypeAWS:
		infraConfigBytes, _ = aws.GetInfrastructureConfig("10.250.0.0/22", zones)
	case hyperscaler.TypeAzure:
		infraConfigBytes, _ = azure.GetInfrastructureConfig("10.250.0.0/22", zones)
	default:
		panic("unknown provider type")
	}

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
		InfrastructureConfig: &runtime.RawExtension{Raw: infraConfigBytes},
	}
}

func fixAWSInfrastructureConfig(workersCIDR string, zones []string) *runtime.RawExtension {
	infraConfig, _ := aws.GetInfrastructureConfig(workersCIDR, zones)
	return &runtime.RawExtension{Raw: infraConfig}
}

func fixAWSControlPlaneConfig() *runtime.RawExtension {
	controlPlaneConfig, _ := aws.GetControlPlaneConfig([]string{})
	return &runtime.RawExtension{Raw: controlPlaneConfig}
}

func fixAzureInfrastructureConfig(workersCIDR string, zones []string) *runtime.RawExtension {
	infraConfig, _ := azure.GetInfrastructureConfig(workersCIDR, zones)
	return &runtime.RawExtension{Raw: infraConfig}
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

func fixProviderWithMultipleWorkers(providerType string, zones []string) imv1.Provider {
	return imv1.Provider{
		Type: providerType,
		Workers: []gardener.Worker{
			{
				Name: "worker",
				Machine: gardener.Machine{
					Type: "m6i.large",
				},
				Minimum: 1,
				Maximum: 3,
				Zones:   zones,
			},
			{
				Name: "worker",
				Machine: gardener.Machine{
					Type:  "m6i.large",
					Image: &gardener.ShootMachineImage{},
				},
				Minimum: 1,
				Maximum: 3,
				Zones:   zones,
			},
			{
				Name: "worker",
				Machine: gardener.Machine{
					Type: "m6i.large",
				},
				Minimum: 1,
				Maximum: 3,
				Zones:   zones,
			},
		},
	}
}

func TestGetAllWorkersZones(t *testing.T) {
	tests := []struct {
		name     string
		workers  []gardener.Worker
		expected []string
		wantErr  bool
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
			wantErr:  false,
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
			wantErr:  false,
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
					Zones: []string{"zone1", "zone3"},
				},
			},
			expected: nil,
			wantErr:  true,
		},
		{
			name:     "No workers provided",
			workers:  []gardener.Worker{},
			expected: nil,
			wantErr:  true,
		},
		{
			name: "Duplicate zones in a single worker",
			workers: []gardener.Worker{
				{
					Name:  "worker1",
					Zones: []string{"zone1", "zone1"},
				},
			},
			expected: nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			zones, err := getNetworkingZonesFromWorkers(tt.workers)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, zones)
			}
		})
	}
}

func assertProvider(t *testing.T, runtimeShoot imv1.RuntimeShoot, shoot gardener.Shoot, expectWorkerConfig bool, expectedMachineImageName, expectedMachineImageVersion string) {
	assert.Equal(t, runtimeShoot.Provider.Type, shoot.Spec.Provider.Type)
	assert.Equal(t, runtimeShoot.Provider.Workers, shoot.Spec.Provider.Workers)
	assert.Equal(t, false, shoot.Spec.Provider.WorkersSettings.SSHAccess.Enabled)
	assert.NotEmpty(t, shoot.Spec.Provider.InfrastructureConfig)
	assert.NotEmpty(t, shoot.Spec.Provider.InfrastructureConfig.Raw)
	assert.NotEmpty(t, shoot.Spec.Provider.ControlPlaneConfig)
	assert.NotEmpty(t, shoot.Spec.Provider.ControlPlaneConfig.Raw)
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

func assertProviderSpecificConfigAWS(t *testing.T, shoot gardener.Shoot, expectedZonesCount int) {
	var infrastructureConfig v1alpha1.InfrastructureConfig

	err := json.Unmarshal(shoot.Spec.Provider.InfrastructureConfig.Raw, &infrastructureConfig)
	require.NoError(t, err)

	assert.Equal(t, expectedZonesCount, len(infrastructureConfig.Networks.Zones))
}

func assertProviderSpecificConfigAzure(t *testing.T, shoot gardener.Shoot, expectedZonesCount int) {
	var infrastructureConfig azure.InfrastructureConfig

	err := json.Unmarshal(shoot.Spec.Provider.InfrastructureConfig.Raw, &infrastructureConfig)
	require.NoError(t, err)

	assert.Equal(t, expectedZonesCount, len(infrastructureConfig.Networks.Zones))
}

func assertProviderSpecificConfigGCP(t *testing.T, shoot gardener.Shoot, expectedZonesCount int) {
	var infrastructureConfig gcp.InfrastructureConfig

	err := json.Unmarshal(shoot.Spec.Provider.InfrastructureConfig.Raw, &infrastructureConfig)
	require.NoError(t, err)

	// validate the networking cidr here
}
