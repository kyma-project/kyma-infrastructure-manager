package provider

import (
	"encoding/json"
	"testing"

	alicloudext "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud/v1alpha1"
	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/extender/testutils"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/hyperscaler"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/hyperscaler/alicloud"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
)

func TestProviderExtenderForCreateAlicloud(t *testing.T) {
	// tests of ProviderExtenderForCreateOperation for Alicloud provider and single worker
	for tname, tc := range map[string]struct {
		Runtime                     imv1.Runtime
		DefaultMachineImageVersion  string
		ExpectedMachineImageVersion string
		DefaultMachineImageName     string
		ExpectedMachineImageName    string
		CurrentZonesConfig          []string
		ExpectedZonesCount          int
	}{
		"Create provider specific config for Alicloud without worker config and one zone": {
			Runtime: imv1.Runtime{
				Spec: imv1.RuntimeSpec{
					Shoot: imv1.RuntimeShoot{
						Provider: fixProvider(hyperscaler.TypeAlicloud, "gardenlinux", "1312.2.0", []string{"eu-central-1a"}),
						Networking: imv1.Networking{
							Nodes: "10.250.0.0/22",
						},
					},
				},
			},
			DefaultMachineImageVersion:  "1312.3.0",
			ExpectedMachineImageVersion: "1312.2.0",
			ExpectedMachineImageName:    "gardenlinux",
			ExpectedZonesCount:          1,
		},
		"Create provider specific config for Alicloud without worker config and two zones": {
			Runtime: imv1.Runtime{
				Spec: imv1.RuntimeSpec{
					Shoot: imv1.RuntimeShoot{
						Provider: fixProvider(hyperscaler.TypeAlicloud, "gardenlinux", "1312.2.0", []string{"eu-central-1a", "eu-central-1b"}),
						Networking: imv1.Networking{
							Nodes: "10.250.0.0/22",
						},
					},
				},
			},
			DefaultMachineImageVersion:  "1312.3.0",
			ExpectedMachineImageVersion: "1312.2.0",
			ExpectedMachineImageName:    "gardenlinux",
			ExpectedZonesCount:          2,
		},
		"Create provider specific config for Alicloud without worker config and three zones": {
			Runtime: imv1.Runtime{
				Spec: imv1.RuntimeSpec{
					Shoot: imv1.RuntimeShoot{
						Provider: fixProvider(hyperscaler.TypeAlicloud, "gardenlinux", "1312.2.0", []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}),
						Networking: imv1.Networking{
							Nodes: "10.250.0.0/22",
						},
					},
				},
			},
			DefaultMachineImageVersion:  "1312.3.0",
			ExpectedMachineImageVersion: "1312.2.0",
			ExpectedMachineImageName:    "gardenlinux",
			ExpectedZonesCount:          3,
		},
		"Create provider config for Alicloud with worker config and three zones": {
			Runtime: imv1.Runtime{
				Spec: imv1.RuntimeSpec{
					Shoot: imv1.RuntimeShoot{
						Provider: fixProvider(hyperscaler.TypeAlicloud, "", "", []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}),
						Networking: imv1.Networking{
							Nodes: "10.250.0.0/22",
						},
					},
				},
			},
			DefaultMachineImageVersion:  "1312.3.0",
			ExpectedMachineImageVersion: "1312.3.0",
			ExpectedZonesCount:          3,
		},
	} {
		t.Run(tname, func(t *testing.T) {
			// given
			shoot := testutils.FixEmptyGardenerShoot("cluster", "kcp-system")

			// when

			extender := NewProviderExtenderForCreateOperation(false, false, tc.DefaultMachineImageName, tc.DefaultMachineImageVersion)
			err := extender(tc.Runtime, &shoot)

			// then
			require.NoError(t, err)

			assertProvider(t, tc.Runtime.Spec.Shoot, shoot, false, tc.ExpectedMachineImageName, tc.ExpectedMachineImageVersion)
			assertProviderSpecificConfigAlicloud(t, shoot, tc.ExpectedZonesCount)
		})
	}
}

func TestProviderExtenderForPatchSingleWorkerAlicloud(t *testing.T) {
	// tests of NewProviderExtenderPatch for provider image version patching Alicloud only operation is provider-agnostic
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
						Provider: fixProvider(hyperscaler.TypeAlicloud, "gardenlinux", "1312.2.0", []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}),
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
			CurrentShootWorkers:         fixWorkers("worker", "m6i.large", "gardenlinux", "1312.4.0", 1, 3, []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}),
			ExistingInfraConfig:         fixAlicloudInfrastructureConfig(t, "10.250.0.0/22", []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}),
			ExistingControlPlaneConfig:  fixAlicloudControlPlaneConfig(),
		},
		"Same image name - override current shoot machine image version with new bigger version from RuntimeCR": {
			Runtime: imv1.Runtime{
				Spec: imv1.RuntimeSpec{
					Shoot: imv1.RuntimeShoot{
						Provider: fixProvider(hyperscaler.TypeAlicloud, "gardenlinux", "1312.2.0", []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}),
					},
				},
			},
			EnableIMDSv2:                false,
			DefaultMachineImageName:     "gardenlinux",
			DefaultMachineImageVersion:  "1312.3.0",
			CurrentShootWorkers:         fixWorkers("worker", "m6i.large", "gardenlinux", "1312.1.0", 1, 3, []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}),
			ExpectedMachineImageVersion: "1312.2.0",
			ExpectedZonesCount:          3,
			ExpectedMachineImageName:    "gardenlinux",
			ExistingInfraConfig:         fixAlicloudInfrastructureConfig(t, "10.250.0.0/22", []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}),
			ExistingControlPlaneConfig:  fixAlicloudControlPlaneConfig(),
		},
		"Same image name - no version is provided override current shoot machine image version with default version": {
			Runtime: imv1.Runtime{
				Spec: imv1.RuntimeSpec{
					Shoot: imv1.RuntimeShoot{
						Provider: fixProvider(hyperscaler.TypeAlicloud, "", "", []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}),
					},
				},
			},
			EnableIMDSv2:                false,
			DefaultMachineImageName:     "gardenlinux",
			DefaultMachineImageVersion:  "1312.3.0",
			CurrentShootWorkers:         fixWorkers("worker", "m6i.large", "gardenlinux", "1312.1.0", 1, 3, []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}),
			ExpectedMachineImageVersion: "1312.3.0",
			ExpectedZonesCount:          3,
			ExpectedMachineImageName:    "gardenlinux",
			ExistingInfraConfig:         fixAlicloudInfrastructureConfig(t, "10.250.0.0/22", []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}),
			ExistingControlPlaneConfig:  fixAlicloudControlPlaneConfig(),
		},
		"Different image name - override current shoot machine image and version with new data from RuntimeCR": {
			Runtime: imv1.Runtime{
				Spec: imv1.RuntimeSpec{
					Shoot: imv1.RuntimeShoot{
						Provider: fixProvider(hyperscaler.TypeAlicloud, "gardenlinux", "1312.2.0", []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}),
					},
				},
			},
			EnableIMDSv2:                false,
			DefaultMachineImageName:     "gardenlinux",
			DefaultMachineImageVersion:  "1312.3.0",
			CurrentShootWorkers:         fixWorkers("worker", "m6i.large", "ubuntu", "1312.4.0", 1, 3, []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}),
			ExpectedZonesCount:          3,
			ExpectedMachineImageName:    "gardenlinux",
			ExpectedMachineImageVersion: "1312.2.0",
			ExistingInfraConfig:         fixAlicloudInfrastructureConfig(t, "10.250.0.0/22", []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}),
			ExistingControlPlaneConfig:  fixAlicloudControlPlaneConfig(),
		},
		"Different image name - no data is provided override current shoot machine image and version with default data": {
			Runtime: imv1.Runtime{
				Spec: imv1.RuntimeSpec{
					Shoot: imv1.RuntimeShoot{
						Provider: fixProvider(hyperscaler.TypeAlicloud, "", "", []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}),
					},
				},
			},
			EnableIMDSv2:                false,
			DefaultMachineImageName:     "gardenlinux",
			DefaultMachineImageVersion:  "1312.3.0",
			CurrentShootWorkers:         fixWorkers("worker", "m6i.large", "ubuntu", "1312.4.0", 1, 3, []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}),
			ExpectedMachineImageVersion: "1312.3.0",
			ExpectedZonesCount:          3,
			ExpectedMachineImageName:    "gardenlinux",
			ExistingInfraConfig:         fixAlicloudInfrastructureConfig(t, "10.250.0.0/22", []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}),
			ExistingControlPlaneConfig:  fixAlicloudControlPlaneConfig(),
		},
		"Wrong current image name - use data from RuntimeCR": {
			Runtime: imv1.Runtime{
				Spec: imv1.RuntimeSpec{
					Shoot: imv1.RuntimeShoot{
						Provider: fixProvider(hyperscaler.TypeAlicloud, "gardenlinux", "1312.2.0", []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}),
					},
				},
			},
			EnableIMDSv2:                false,
			DefaultMachineImageName:     "gardenlinux",
			DefaultMachineImageVersion:  "1312.3.0",
			CurrentShootWorkers:         fixWorkers("worker", "m6i.large", "", "1312.4.0", 1, 3, []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}),
			ExpectedMachineImageVersion: "1312.2.0",
			ExpectedZonesCount:          3,
			ExpectedMachineImageName:    "gardenlinux",
			ExistingInfraConfig:         fixAlicloudInfrastructureConfig(t, "10.250.0.0/22", []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}),
			ExistingControlPlaneConfig:  fixAlicloudControlPlaneConfig(),
		},
		"Wrong current image version - use data from RuntimeCR": {
			Runtime: imv1.Runtime{
				Spec: imv1.RuntimeSpec{
					Shoot: imv1.RuntimeShoot{
						Provider: fixProvider(hyperscaler.TypeAlicloud, "gardenlinux", "1312.2.0", []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}),
					},
				},
			},
			EnableIMDSv2:                false,
			DefaultMachineImageName:     "gardenlinux",
			DefaultMachineImageVersion:  "1312.3.0",
			CurrentShootWorkers:         fixWorkers("worker", "m6i.large", "gardenlinux", "", 1, 3, []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}),
			ExpectedMachineImageVersion: "1312.2.0",
			ExpectedZonesCount:          3,
			ExpectedMachineImageName:    "gardenlinux",
			ExistingInfraConfig:         fixAlicloudInfrastructureConfig(t, "10.250.0.0/22", []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}),
			ExistingControlPlaneConfig:  fixAlicloudControlPlaneConfig(),
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

			assertProvider(t, tc.Runtime.Spec.Shoot, shoot, false, tc.ExpectedMachineImageName, tc.ExpectedMachineImageVersion)
			assertProviderSpecificConfigAlicloud(t, shoot, tc.ExpectedZonesCount)
		})
	}
}

func fixAlicloudInfrastructureConfig(t *testing.T, workersCIDR string, zones []string) *runtime.RawExtension {
	infraConfig, err := alicloud.NewInfrastructureConfig(workersCIDR, zones)

	assert.NoError(t, err)

	infraConfig.Networks.VPC.ID = ptr.To("vpc-123456")
	infraConfig.Networks.VPC.CIDR = ptr.To("192.168.0.1/24")

	infraConfigBytes, err := json.Marshal(infraConfig)

	assert.NoError(t, err)

	return &runtime.RawExtension{Raw: infraConfigBytes}
}

func fixAlicloudControlPlaneConfig() *runtime.RawExtension {
	controlPlaneConfig, _ := alicloud.GetControlPlaneConfig([]string{})
	return &runtime.RawExtension{Raw: controlPlaneConfig}
}

func assertProviderSpecificConfigAlicloud(t *testing.T, shoot gardener.Shoot, expectedZonesCount int) {
	var infrastructureConfig alicloudext.InfrastructureConfig
	var controlPlaneConfig alicloudext.ControlPlaneConfig

	err := json.Unmarshal(shoot.Spec.Provider.InfrastructureConfig.Raw, &infrastructureConfig)
	require.NoError(t, err)

	err = json.Unmarshal(shoot.Spec.Provider.ControlPlaneConfig.Raw, &controlPlaneConfig)

	assert.Equal(t, expectedZonesCount, len(infrastructureConfig.Networks.Zones))
}
