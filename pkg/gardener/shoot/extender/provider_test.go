package extender

import (
	"encoding/json"
	"testing"

	"github.com/gardener/gardener-extension-provider-aws/pkg/apis/aws/v1alpha1"
	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/hyperscaler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProviderExtender(t *testing.T) {
	/*for tname, tc := range map[string]struct {
		Runtime                     imv1.Runtime
		EnableIMDSv2                bool
		DefaultMachineImageVersion  string
		CurrentMachineImageVersion  string
		ExpectedMachineImageVersion string
		DefaultMachineImageName     string
		CurrentMachineImageName     string
		ExpectedMachineImageName    string
		CurrentZonesConfig          []string
		ExpectedZonesCount          int
		TestForPatch                bool
	}{
		"Create provider specific config for AWS without worker config - create option": {
			Runtime: imv1.Runtime{
				Spec: imv1.RuntimeSpec{
					Shoot: imv1.RuntimeShoot{
						Provider: fixAWSProvider("gardenlinux", "1312.2.0"),
					},
				},
			},
			EnableIMDSv2:                false,
			DefaultMachineImageVersion:  "1312.3.0",
			ExpectedMachineImageVersion: "1312.2.0",
			ExpectedMachineImageName:    "gardenlinux",
			ExpectedZonesCount:          3,
			TestForPatch:                false,
		},
		"Create provider specific config for AWS with worker config - create option": {
			Runtime: imv1.Runtime{
				Spec: imv1.RuntimeSpec{
					Shoot: imv1.RuntimeShoot{
						Provider: fixAWSProvider("", ""),
					},
				},
			},
			EnableIMDSv2:                true,
			DefaultMachineImageVersion:  "1312.3.0",
			ExpectedMachineImageVersion: "1312.3.0",
			ExpectedZonesCount:          3,
			TestForPatch:                false,
		},
		"Create provider specific config for AWS with multiple workers - create option": {
			Runtime: imv1.Runtime{
				Spec: imv1.RuntimeSpec{
					Shoot: imv1.RuntimeShoot{
						Provider: fixAWSProviderWithMultipleWorkers(),
					},
				},
			},
			EnableIMDSv2:                false,
			DefaultMachineImageVersion:  "1312.3.0",
			ExpectedMachineImageVersion: "1312.3.0",
			ExpectedZonesCount:          3,
			TestForPatch:                false,
		},
		"Patch option same image name - use bigger current shoot machine image version as image version": {
			Runtime: imv1.Runtime{
				Spec: imv1.RuntimeSpec{
					Shoot: imv1.RuntimeShoot{
						Provider: fixAWSProvider("gardenlinux", "1312.2.0"),
					},
				},
			},
			EnableIMDSv2:                false,
			DefaultMachineImageName:     "gardenlinux",
			DefaultMachineImageVersion:  "1312.3.0",
			CurrentMachineImageName:     "gardenlinux",
			CurrentMachineImageVersion:  "1312.4.0",
			ExpectedMachineImageVersion: "1312.4.0",
			ExpectedZonesCount:          3,
			ExpectedMachineImageName:    "gardenlinux",
			CurrentZonesConfig:          []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"},
			TestForPatch:                true,
		},
		"Patch option same image name - override current shoot machine image version with new bigger version from RuntimeCR": {
			Runtime: imv1.Runtime{
				Spec: imv1.RuntimeSpec{
					Shoot: imv1.RuntimeShoot{
						Provider: fixAWSProvider("gardenlinux", "1312.2.0"),
					},
				},
			},
			EnableIMDSv2:                false,
			DefaultMachineImageName:     "gardenlinux",
			DefaultMachineImageVersion:  "1312.3.0",
			CurrentMachineImageName:     "gardenlinux",
			CurrentMachineImageVersion:  "1312.1.0",
			ExpectedMachineImageVersion: "1312.2.0",
			ExpectedZonesCount:          3,
			ExpectedMachineImageName:    "gardenlinux",
			CurrentZonesConfig:          []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"},
			TestForPatch:                true,
		},
		"Patch option same image name - override current shoot machine image version with default version": {
			Runtime: imv1.Runtime{
				Spec: imv1.RuntimeSpec{
					Shoot: imv1.RuntimeShoot{
						Provider: fixAWSProvider("", ""),
					},
				},
			},
			EnableIMDSv2:                false,
			DefaultMachineImageName:     "gardenlinux",
			DefaultMachineImageVersion:  "1312.3.0",
			CurrentMachineImageName:     "gardenlinux",
			CurrentMachineImageVersion:  "1312.1.0",
			ExpectedMachineImageVersion: "1312.3.0",
			ExpectedZonesCount:          3,
			ExpectedMachineImageName:    "gardenlinux",
			CurrentZonesConfig:          []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"},
			TestForPatch:                true,
		},
		"Patch option different image name - override current shoot machine image and version with new data from RuntimeCR": {
			Runtime: imv1.Runtime{
				Spec: imv1.RuntimeSpec{
					Shoot: imv1.RuntimeShoot{
						Provider: fixAWSProvider("gardenlinux", "1312.2.0"),
					},
				},
			},
			EnableIMDSv2:                false,
			DefaultMachineImageName:     "gardenlinux",
			DefaultMachineImageVersion:  "1312.3.0",
			CurrentMachineImageName:     "ubuntu",
			CurrentMachineImageVersion:  "1312.4.0",
			ExpectedMachineImageVersion: "1312.2.0",
			ExpectedZonesCount:          3,
			ExpectedMachineImageName:    "gardenlinux",
			CurrentZonesConfig:          []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"},
			TestForPatch:                true,
		},
		"Patch option different image name - override current shoot machine image and version with default data": {
			Runtime: imv1.Runtime{
				Spec: imv1.RuntimeSpec{
					Shoot: imv1.RuntimeShoot{
						Provider: fixAWSProvider("", ""),
					},
				},
			},
			EnableIMDSv2:                false,
			DefaultMachineImageName:     "gardenlinux",
			DefaultMachineImageVersion:  "1312.3.0",
			CurrentMachineImageName:     "ubuntu",
			CurrentMachineImageVersion:  "1312.4.0",
			ExpectedMachineImageVersion: "1312.3.0",
			ExpectedZonesCount:          3,
			ExpectedMachineImageName:    "gardenlinux",
			CurrentZonesConfig:          []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"},
			TestForPatch:                true,
		},
		"Patch option wrong current image name - use data from RuntimeCR": {
			Runtime: imv1.Runtime{
				Spec: imv1.RuntimeSpec{
					Shoot: imv1.RuntimeShoot{
						Provider: fixAWSProvider("gardenlinux", "1312.2.0"),
					},
				},
			},
			EnableIMDSv2:                false,
			DefaultMachineImageName:     "gardenlinux",
			DefaultMachineImageVersion:  "1312.3.0",
			CurrentMachineImageName:     "",
			CurrentMachineImageVersion:  "1312.4.0",
			ExpectedMachineImageVersion: "1312.2.0",
			ExpectedZonesCount:          3,
			ExpectedMachineImageName:    "gardenlinux",
			CurrentZonesConfig:          []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"},
			TestForPatch:                true,
		},
		"Patch option wrong current image version - use data from RuntimeCR": {
			Runtime: imv1.Runtime{
				Spec: imv1.RuntimeSpec{
					Shoot: imv1.RuntimeShoot{
						Provider: fixAWSProvider("gardenlinux", "1312.2.0"),
					},
				},
			},
			EnableIMDSv2:                false,
			DefaultMachineImageName:     "gardenlinux",
			DefaultMachineImageVersion:  "1312.3.0",
			CurrentMachineImageName:     "gardenlinux",
			CurrentMachineImageVersion:  "",
			ExpectedMachineImageVersion: "1312.2.0",
			ExpectedZonesCount:          3,
			ExpectedMachineImageName:    "gardenlinux",
			CurrentZonesConfig:          []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"},
			TestForPatch:                true,
		},
		// "Patch option different image name - override image name and version with current image name and version": {},
	} {
		t.Run(tname, func(t *testing.T) {
			// given
			shoot := fixEmptyGardenerShoot("cluster", "kcp-system")

			// when

			var err error
			if tc.TestForPatch {
				extender := NewProviderExtenderPatchOperation(tc.EnableIMDSv2, tc.DefaultMachineImageName, tc.DefaultMachineImageVersion)
				err = extender(tc.Runtime, &shoot)
			} else {
				extender := NewProviderExtenderForCreateOperation(tc.EnableIMDSv2, tc.DefaultMachineImageName, tc.DefaultMachineImageVersion)
				err = extender(tc.Runtime, &shoot)
			}

			// then
			require.NoError(t, err)

			assertProvider(t, tc.Runtime.Spec.Shoot, shoot, tc.EnableIMDSv2, tc.ExpectedMachineImageName, tc.ExpectedMachineImageVersion)
			assertProviderSpecificConfig(t, shoot, tc.ExpectedZonesCount)
		})
	}*/

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

func fixAWSProvider(machineImageName, machineImageVersion string) imv1.Provider {
	return imv1.Provider{
		Type: hyperscaler.TypeAWS,
		Workers: []gardener.Worker{
			{
				Name: "worker",
				Machine: gardener.Machine{
					Type:  "m6i.large",
					Image: fixMachineImage(machineImageName, machineImageVersion),
				},
				Minimum: 1,
				Maximum: 3,
				Zones: []string{
					"eu-central-1a",
					"eu-central-1b",
					"eu-central-1c",
				},
			},
		},
	}
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

func fixAWSProviderWithMultipleWorkers() imv1.Provider {
	return imv1.Provider{
		Type: hyperscaler.TypeAWS,
		Workers: []gardener.Worker{
			{
				Name: "worker",
				Machine: gardener.Machine{
					Type: "m6i.large",
				},
				Minimum: 1,
				Maximum: 3,
				Zones: []string{
					"eu-central-1a",
					"eu-central-1b",
					"eu-central-1c",
				},
			},
			{
				Name: "worker",
				Machine: gardener.Machine{
					Type:  "m6i.large",
					Image: &gardener.ShootMachineImage{},
				},
				Minimum: 1,
				Maximum: 3,
				Zones: []string{
					"eu-central-1a",
					"eu-central-1b",
					"eu-central-1c",
				},
			},
			{
				Name: "worker",
				Machine: gardener.Machine{
					Type: "m6i.large",
				},
				Minimum: 1,
				Maximum: 3,
				Zones: []string{
					"eu-central-1a",
					"eu-central-1b",
					"eu-central-1c",
				},
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

func assertProviderSpecificConfig(t *testing.T, shoot gardener.Shoot, expectedZonesCount int) {
	var infrastructureConfig v1alpha1.InfrastructureConfig

	err := json.Unmarshal(shoot.Spec.Provider.InfrastructureConfig.Raw, &infrastructureConfig)
	require.NoError(t, err)

	assert.Equal(t, expectedZonesCount, len(infrastructureConfig.Networks.Zones))
}
