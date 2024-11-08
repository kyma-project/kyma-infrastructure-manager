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
	for tname, testCase := range map[string]struct {
		Runtime                     imv1.Runtime
		EnableIMDSv2                bool
		DefaultMachineImageVersion  string
		ExpectedMachineImageVersion string
		DefaultMachineImageName     string
		ExpectedMachineImageName    string
		ExpectedZonesCount          int
	}{
		"Create provider specific config for AWS without worker config": {
			Runtime: imv1.Runtime{
				Spec: imv1.RuntimeSpec{
					Shoot: imv1.RuntimeShoot{
						Provider: fixAWSProvider("1312.2.0"),
					},
				},
			},
			EnableIMDSv2:                false,
			DefaultMachineImageVersion:  "1312.3.0",
			ExpectedMachineImageVersion: "1312.2.0",
			ExpectedZonesCount:          3,
		},
		"Create provider specific config for AWS with worker config": {
			Runtime: imv1.Runtime{
				Spec: imv1.RuntimeSpec{
					Shoot: imv1.RuntimeShoot{
						Provider: fixAWSProvider(""),
					},
				},
			},
			EnableIMDSv2:                true,
			DefaultMachineImageVersion:  "1312.3.0",
			ExpectedMachineImageVersion: "1312.3.0",
			ExpectedZonesCount:          3,
		},
		"Create provider specific config for AWS with multiple workers": {
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
		},
	} {
		t.Run(tname, func(t *testing.T) {
			// given
			shoot := fixEmptyGardenerShoot("cluster", "kcp-system")

			// when
			extender := NewProviderExtenderForCreateOperation(testCase.EnableIMDSv2, testCase.DefaultMachineImageName, testCase.DefaultMachineImageVersion)
			err := extender(testCase.Runtime, &shoot)

			// then
			require.NoError(t, err)

			assertProvider(t, testCase.Runtime.Spec.Shoot, shoot, testCase.EnableIMDSv2, testCase.ExpectedMachineImageName, testCase.ExpectedMachineImageVersion)
			assertProviderSpecificConfig(t, shoot, testCase.ExpectedZonesCount)
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

func fixAWSProvider(machineImageVersion string) imv1.Provider {
	return imv1.Provider{
		Type: hyperscaler.TypeAWS,
		Workers: []gardener.Worker{
			{
				Name: "worker",
				Machine: gardener.Machine{
					Type:  "m6i.large",
					Image: fixMachineImage(machineImageVersion),
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

func fixMachineImage(machineImageVersion string) *gardener.ShootMachineImage {
	if machineImageVersion != "" {
		return &gardener.ShootMachineImage{
			Version: &machineImageVersion,
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
					"eu-central-1b",
					"eu-central-1c",
				},
			},
		},
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
