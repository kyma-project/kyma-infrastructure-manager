package provider

import (
	"encoding/json"
	"testing"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/pkg/config"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/extender/testutils"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/hyperscaler"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/hyperscaler/gdch"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestProviderExtenderForCreateGDCH(t *testing.T) {
	for tname, tc := range map[string]struct {
		Runtime                     imv1.Runtime
		DefaultMachineImageVersion  string
		ExpectedMachineImageVersion string
		DefaultMachineImageName     string
		ExpectedMachineImageName    string
		ExpectedNodeCIDR            string
		ExpectedZones               []gdch.Zones
	}{
		"Create provider specific config for GDCH with three zones (doc example /24 -> three /26)": {
			Runtime: imv1.Runtime{
				Spec: imv1.RuntimeSpec{
					Shoot: imv1.RuntimeShoot{
						Provider: fixProvider(hyperscaler.TypeGDCH, "gardenlinux", "1312.2.0", []string{"us-west16-b", "us-west16-c", "us-west16-d"}),
						Networking: imv1.Networking{
							Pods:     "100.64.0.0/22",
							Nodes:    "10.72.0.0/24",
							Services: "100.104.0.0/13",
						},
					},
				},
			},
			DefaultMachineImageVersion:  "1312.3.0",
			ExpectedMachineImageVersion: "1312.2.0",
			ExpectedMachineImageName:    "gardenlinux",
			ExpectedNodeCIDR:            "10.72.0.0/24",
			ExpectedZones: []gdch.Zones{
				{Name: "us-west16-b", CIDR: "10.72.0.0/26"},
				{Name: "us-west16-c", CIDR: "10.72.0.64/26"},
				{Name: "us-west16-d", CIDR: "10.72.0.128/26"},
			},
		},
		"Create provider specific config for GDCH with three zones (reference input /16 -> three /18)": {
			Runtime: imv1.Runtime{
				Spec: imv1.RuntimeSpec{
					Shoot: imv1.RuntimeShoot{
						Provider: fixProvider(hyperscaler.TypeGDCH, "gardenlinux", "1312.2.0", []string{"us-west16-a", "us-west16-b", "us-west16-c"}),
						Networking: imv1.Networking{
							Pods:     "100.64.0.0/22",
							Nodes:    "10.180.0.0/16",
							Services: "100.104.0.0/13",
						},
					},
				},
			},
			DefaultMachineImageVersion:  "1312.3.0",
			ExpectedMachineImageVersion: "1312.2.0",
			ExpectedMachineImageName:    "gardenlinux",
			ExpectedNodeCIDR:            "10.180.0.0/16",
			ExpectedZones: []gdch.Zones{
				{Name: "us-west16-a", CIDR: "10.180.0.0/18"},
				{Name: "us-west16-b", CIDR: "10.180.64.0/18"},
				{Name: "us-west16-c", CIDR: "10.180.128.0/18"},
			},
		},
	} {
		t.Run(tname, func(t *testing.T) {
			// given
			shoot := testutils.FixEmptyGardenerShoot("cluster", "kcp-system")

			// when
			extender := NewProviderExtenderForCreateOperation(false, false, config.MachineImageConfig{DefaultName: tc.DefaultMachineImageName, DefaultVersion: tc.DefaultMachineImageVersion}, config.WorkerConfig{DefaultMaxEvictRetries: "2", DefaultMachineDrainTimeout: "15m"})
			err := extender(tc.Runtime, &shoot)

			// then
			require.NoError(t, err)
			assertProvider(t, tc.Runtime.Spec.Shoot, shoot, false, tc.ExpectedMachineImageName, tc.ExpectedMachineImageVersion)
			assertProviderSpecificConfigGDCH(t, shoot, tc.ExpectedNodeCIDR, tc.ExpectedZones)
		})
	}
}

func TestProviderExtenderForCreateGDCHErrors(t *testing.T) {
	for tname, tc := range map[string]struct {
		Runtime imv1.Runtime
	}{
		"Fail when fewer than three zones are provided": {
			Runtime: imv1.Runtime{
				Spec: imv1.RuntimeSpec{
					Shoot: imv1.RuntimeShoot{
						Provider: fixProvider(hyperscaler.TypeGDCH, "gardenlinux", "1312.2.0", []string{"us-west16-b", "us-west16-c"}),
						Networking: imv1.Networking{
							Pods:     "100.64.0.0/22",
							Nodes:    "10.72.0.0/24",
							Services: "100.104.0.0/13",
						},
					},
				},
			},
		},
		"Fail when more than three zones are provided": {
			Runtime: imv1.Runtime{
				Spec: imv1.RuntimeSpec{
					Shoot: imv1.RuntimeShoot{
						Provider: fixProvider(hyperscaler.TypeGDCH, "gardenlinux", "1312.2.0", []string{"us-west16-a", "us-west16-b", "us-west16-c", "us-west16-d"}),
						Networking: imv1.Networking{
							Pods:     "100.64.0.0/22",
							Nodes:    "10.72.0.0/24",
							Services: "100.104.0.0/13",
						},
					},
				},
			},
		},
		"Fail when node CIDR is too small to split into three zones": {
			Runtime: imv1.Runtime{
				Spec: imv1.RuntimeSpec{
					Shoot: imv1.RuntimeShoot{
						Provider: fixProvider(hyperscaler.TypeGDCH, "gardenlinux", "1312.2.0", []string{"us-west16-b", "us-west16-c", "us-west16-d"}),
						Networking: imv1.Networking{
							Pods:     "100.64.0.0/22",
							Nodes:    "10.72.0.0/31",
							Services: "100.104.0.0/13",
						},
					},
				},
			},
		},
	} {
		t.Run(tname, func(t *testing.T) {
			// given
			shoot := testutils.FixEmptyGardenerShoot("cluster", "kcp-system")

			// when
			extender := NewProviderExtenderForCreateOperation(false, false, config.MachineImageConfig{DefaultName: "gardenlinux", DefaultVersion: "1312.3.0"}, config.WorkerConfig{DefaultMaxEvictRetries: "2", DefaultMachineDrainTimeout: "15m"})
			err := extender(tc.Runtime, &shoot)

			// then
			require.Error(t, err)
		})
	}
}

func TestProviderExtenderForPatchWorkersUpdateGDCH(t *testing.T) {
	// tests of NewProviderExtenderPatchOperation for GDCH provider.
	// The set of zones stays at exactly three, so no zones are added and the
	// existing infrastructure/control plane config is reused unchanged.
	gdchZones := []string{"us-west16-b", "us-west16-c", "us-west16-d"}

	for tname, tc := range map[string]struct {
		Runtime                    imv1.Runtime
		DefaultMachineImageVersion string
		DefaultMachineImageName    string
		CurrentShootWorkers        []gardener.Worker
		ExistingInfraConfig        *runtime.RawExtension
		ExistingControlPlaneConfig *runtime.RawExtension
		ExpectedShootWorkers       []gardener.Worker
		ExpectedNodeCIDR           string
		ExpectedZones              []gdch.Zones
	}{
		"Update machine type and image name and version in multiple workers separately": {
			Runtime: imv1.Runtime{
				Spec: imv1.RuntimeSpec{
					Shoot: imv1.RuntimeShoot{
						Provider: fixProviderWithMultipleWorkers(hyperscaler.TypeGDCH, fixMultipleWorkers([]workerConfig{
							{"main-worker", "n2-standard-4", "gardenlinux", "1313.4.0", 1, 3, gdchZones},
							{"additional", "n2-standard-4", "gardenlinux", "1313.2.0", 1, 3, gdchZones},
						})),
						Networking: imv1.Networking{
							Pods:     "100.64.0.0/22",
							Nodes:    "10.72.0.0/24",
							Services: "100.104.0.0/13",
						},
					},
				},
			},
			DefaultMachineImageName:    "gardenlinux",
			DefaultMachineImageVersion: "1312.3.0",
			CurrentShootWorkers: fixMultipleWorkers([]workerConfig{
				{"main-worker", "n2-standard-2", "gardenlinux", "1312.4.0", 1, 3, gdchZones},
				{"additional", "n2-standard-2", "gardenlinux", "1312.2.0", 1, 3, gdchZones}}),
			ExpectedShootWorkers: fixMultipleWorkers([]workerConfig{
				{"main-worker", "n2-standard-4", "gardenlinux", "1313.4.0", 1, 3, gdchZones},
				{"additional", "n2-standard-4", "gardenlinux", "1313.2.0", 1, 3, gdchZones}}),
			ExistingInfraConfig:        fixGDCHInfrastructureConfig(t, "10.72.0.0/24", gdchZones),
			ExistingControlPlaneConfig: fixGDCHControlPlaneConfig(t),
			ExpectedNodeCIDR:           "10.72.0.0/24",
			ExpectedZones: []gdch.Zones{
				{Name: "us-west16-b", CIDR: "10.72.0.0/26"},
				{Name: "us-west16-c", CIDR: "10.72.0.64/26"},
				{Name: "us-west16-d", CIDR: "10.72.0.128/26"},
			},
		},
	} {
		t.Run(tname, func(t *testing.T) {
			// given
			shoot := testutils.FixEmptyGardenerShoot("cluster", "kcp-system")

			// when
			extender := NewProviderExtenderPatchOperation(false, tc.CurrentShootWorkers, config.MachineImageConfig{DefaultName: tc.DefaultMachineImageName, DefaultVersion: tc.DefaultMachineImageVersion}, config.WorkerConfig{DefaultMaxEvictRetries: "2", DefaultMachineDrainTimeout: "15m"}, tc.ExistingInfraConfig, tc.ExistingControlPlaneConfig)
			err := extender(tc.Runtime, &shoot)

			// then
			require.NoError(t, err)

			assertProviderMultipleWorkers(t, tc.Runtime.Spec.Shoot, shoot, false, tc.ExpectedShootWorkers)
			assertProviderSpecificConfigGDCH(t, shoot, tc.ExpectedNodeCIDR, tc.ExpectedZones)
		})
	}
}

func fixGDCHInfrastructureConfig(t *testing.T, nodeCIDR string, zones []string) *runtime.RawExtension {
	infraConfig, err := gdch.GetInfrastructureConfig(nodeCIDR, zones)
	require.NoError(t, err)
	return &runtime.RawExtension{Raw: infraConfig}
}

func fixGDCHControlPlaneConfig(t *testing.T) *runtime.RawExtension {
	controlPlaneConfig, err := gdch.GetControlPlaneConfig(nil)
	require.NoError(t, err)
	return &runtime.RawExtension{Raw: controlPlaneConfig}
}

func assertProviderSpecificConfigGDCH(t *testing.T, shoot gardener.Shoot, expectedNodeCIDR string, expectedZones []gdch.Zones) {
	var infraConfig gdch.InfrastructureConfig
	err := json.Unmarshal(shoot.Spec.Provider.InfrastructureConfig.Raw, &infraConfig)
	require.NoError(t, err)

	assert.Equal(t, expectedNodeCIDR, infraConfig.Networks.NodeCIDR)
	assert.Equal(t, expectedZones, infraConfig.Networks.Zones)

	var ctrlPlaneConfig gdch.ControlPlaneConfig
	err = json.Unmarshal(shoot.Spec.Provider.ControlPlaneConfig.Raw, &ctrlPlaneConfig)
	require.NoError(t, err)
	assert.Equal(t, "ControlPlaneConfig", ctrlPlaneConfig.Kind)
}
