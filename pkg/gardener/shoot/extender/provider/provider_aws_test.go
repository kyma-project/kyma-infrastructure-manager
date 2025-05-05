package provider

import (
	"encoding/json"
	awsext "github.com/gardener/gardener-extension-provider-aws/pkg/apis/aws/v1alpha1"
	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/extender/testutils"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/hyperscaler"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/hyperscaler/aws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	"testing"
)

func TestProviderExtenderForCreateAWS(t *testing.T) {
	// tests of ProviderExtenderForCreateOperation for AWS provider and single worker
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
		"Create provider config for AWS with worker config and three zones": {
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
	} {
		t.Run(tname, func(t *testing.T) {
			// given
			shoot := testutils.FixEmptyGardenerShoot("cluster", "kcp-system")

			// when

			extender := NewProviderExtenderForCreateOperation(tc.EnableIMDSv2, tc.DefaultMachineImageName, tc.DefaultMachineImageVersion)
			err := extender(tc.Runtime, &shoot)

			// then
			require.NoError(t, err)

			assertProvider(t, tc.Runtime.Spec.Shoot, shoot, tc.EnableIMDSv2, tc.ExpectedMachineImageName, tc.ExpectedMachineImageVersion)
			assertProviderSpecificConfigAWS(t, shoot, tc.ExpectedZonesCount)
		})
	}
}

func TestProviderExtenderForPatchSingleWorkerAWS(t *testing.T) {
	// tests of NewProviderExtenderPatch for provider image version patching AWS only operation is provider-agnostic
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
			CurrentShootWorkers:         fixWorkers("worker", "m6i.large", "gardenlinux", "1312.4.0", 1, 3, []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}),
			ExistingInfraConfig:         fixAWSInfrastructureConfig(t, "10.250.0.0/22", []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}),
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
			CurrentShootWorkers:         fixWorkers("worker", "m6i.large", "gardenlinux", "1312.1.0", 1, 3, []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}),
			ExpectedMachineImageVersion: "1312.2.0",
			ExpectedZonesCount:          3,
			ExpectedMachineImageName:    "gardenlinux",
			ExistingInfraConfig:         fixAWSInfrastructureConfig(t, "10.250.0.0/22", []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}),
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
			CurrentShootWorkers:         fixWorkers("worker", "m6i.large", "gardenlinux", "1312.1.0", 1, 3, []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}),
			ExpectedMachineImageVersion: "1312.3.0",
			ExpectedZonesCount:          3,
			ExpectedMachineImageName:    "gardenlinux",
			ExistingInfraConfig:         fixAWSInfrastructureConfig(t, "10.250.0.0/22", []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}),
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
			CurrentShootWorkers:         fixWorkers("worker", "m6i.large", "ubuntu", "1312.4.0", 1, 3, []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}),
			ExpectedZonesCount:          3,
			ExpectedMachineImageName:    "gardenlinux",
			ExpectedMachineImageVersion: "1312.2.0",
			ExistingInfraConfig:         fixAWSInfrastructureConfig(t, "10.250.0.0/22", []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}),
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
			CurrentShootWorkers:         fixWorkers("worker", "m6i.large", "ubuntu", "1312.4.0", 1, 3, []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}),
			ExpectedMachineImageVersion: "1312.3.0",
			ExpectedZonesCount:          3,
			ExpectedMachineImageName:    "gardenlinux",
			ExistingInfraConfig:         fixAWSInfrastructureConfig(t, "10.250.0.0/22", []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}),
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
			CurrentShootWorkers:         fixWorkers("worker", "m6i.large", "", "1312.4.0", 1, 3, []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}),
			ExpectedMachineImageVersion: "1312.2.0",
			ExpectedZonesCount:          3,
			ExpectedMachineImageName:    "gardenlinux",
			ExistingInfraConfig:         fixAWSInfrastructureConfig(t, "10.250.0.0/22", []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}),
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
			CurrentShootWorkers:         fixWorkers("worker", "m6i.large", "gardenlinux", "", 1, 3, []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}),
			ExpectedMachineImageVersion: "1312.2.0",
			ExpectedZonesCount:          3,
			ExpectedMachineImageName:    "gardenlinux",
			ExistingInfraConfig:         fixAWSInfrastructureConfig(t, "10.250.0.0/22", []string{"eu-central-1a", "eu-central-1b", "eu-central-1c"}),
			ExistingControlPlaneConfig:  fixAWSControlPlaneConfig(),
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

			assertProvider(t, tc.Runtime.Spec.Shoot, shoot, tc.EnableIMDSv2, tc.ExpectedMachineImageName, tc.ExpectedMachineImageVersion)
			assertProviderSpecificConfigAWS(t, shoot, tc.ExpectedZonesCount)
		})
	}
}

func fixAWSInfrastructureConfig(t *testing.T, workersCIDR string, zones []string) *runtime.RawExtension {
	infraConfig := aws.NewInfrastructureConfig(workersCIDR, zones)

	for i := 0; i < len(zones); i++ {
		infraConfig.Networks.Zones[i].ElasticIPAllocationID = ptr.To("eipalloc-123456")
	}

	infraConfig.DualStack = &awsext.DualStack{
		Enabled: true,
	}
	infraConfig.IgnoreTags = &awsext.IgnoreTags{Keys: []string{"key1"}, KeyPrefixes: []string{"key-prefix-1"}}
	infraConfig.EnableECRAccess = ptr.To(true)
	infraConfig.Networks.VPC.ID = ptr.To("vpc-123456")
	infraConfig.Networks.VPC.GatewayEndpoints = []string{"service-1", "service-2"}

	infraConfigBytes, err := json.Marshal(infraConfig)

	assert.NoError(t, err)

	return &runtime.RawExtension{Raw: infraConfigBytes}
}

func fixAWSControlPlaneConfig() *runtime.RawExtension {
	controlPlaneConfig, _ := aws.GetControlPlaneConfig([]string{})
	return &runtime.RawExtension{Raw: controlPlaneConfig}
}

func assertProviderSpecificConfigAWS(t *testing.T, shoot gardener.Shoot, expectedZonesCount int) {
	var infrastructureConfig awsext.InfrastructureConfig

	err := json.Unmarshal(shoot.Spec.Provider.InfrastructureConfig.Raw, &infrastructureConfig)
	require.NoError(t, err)

	assert.Equal(t, expectedZonesCount, len(infrastructureConfig.Networks.Zones))
}
