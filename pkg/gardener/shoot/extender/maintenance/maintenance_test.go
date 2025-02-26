package maintenance

import (
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/extender/testutils"
	"testing"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/stretchr/testify/assert"
)

func TestMaintenanceExtender(t *testing.T) {

	maintenanceTimeWindow := &gardener.MaintenanceTimeWindow{
		Begin: "200000+0000",
		End:   "230000+0000",
	}
	for _, testCase := range []struct {
		name                                string
		enableKubernetesVersionAutoUpdate   bool
		enableMachineImageVersionAutoUpdate bool
		maintenanceWindow                   *gardener.MaintenanceTimeWindow
	}{
		{
			name:                                "Enable auto-update for only KubernetesVersion",
			enableKubernetesVersionAutoUpdate:   true,
			enableMachineImageVersionAutoUpdate: false,
			maintenanceWindow:                   maintenanceTimeWindow,
		},
		{
			name:                                "Enable auto-update for only MachineImageVersion",
			enableKubernetesVersionAutoUpdate:   false,
			enableMachineImageVersionAutoUpdate: true,
			maintenanceWindow:                   maintenanceTimeWindow,
		},
		{
			name:                                "Should set maintenance time window if it is provided",
			enableKubernetesVersionAutoUpdate:   true,
			enableMachineImageVersionAutoUpdate: true,
			maintenanceWindow:                   maintenanceTimeWindow,
		},
		{
			name:                                "Should not set maintenance time window if it is not provided",
			enableKubernetesVersionAutoUpdate:   true,
			enableMachineImageVersionAutoUpdate: true,
			maintenanceWindow:                   nil,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			// given
			shoot := testutils.FixEmptyGardenerShoot("test", "dev")
			shoot.Spec.Maintenance = &gardener.Maintenance{
				AutoUpdate: &gardener.MaintenanceAutoUpdate{},
			}
			runtimeShoot := imv1.Runtime{
				Spec: imv1.RuntimeSpec{
					Shoot: imv1.RuntimeShoot{
						Name: "test",
					},
				},
			}

			// when
			extender := NewMaintenanceExtender(testCase.enableKubernetesVersionAutoUpdate, testCase.enableMachineImageVersionAutoUpdate, testCase.maintenanceWindow)
			err := extender(runtimeShoot, &shoot)

			// then
			assert.NoError(t, err)
			assert.Equal(t, testCase.enableKubernetesVersionAutoUpdate, shoot.Spec.Maintenance.AutoUpdate.KubernetesVersion)
			assert.Equal(t, testCase.enableMachineImageVersionAutoUpdate, *shoot.Spec.Maintenance.AutoUpdate.MachineImageVersion)
			assert.Equal(t, testCase.maintenanceWindow, shoot.Spec.Maintenance.TimeWindow)
		})
	}
}
