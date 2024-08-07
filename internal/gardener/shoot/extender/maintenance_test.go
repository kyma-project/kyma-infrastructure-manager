package extender

import (
	"testing"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/stretchr/testify/assert"
)

func TestMaintenanceExtender(t *testing.T) {
	for _, testCase := range []struct {
		name                                string
		enableKubernetesVersionAutoUpdate   bool
		enableMachineImageVersionAutoUpdate bool
	}{
		{
			name:                                "Enable auto-update for only KubernetesVersion",
			enableKubernetesVersionAutoUpdate:   true,
			enableMachineImageVersionAutoUpdate: false,
		},
		{
			name:                                "Enable auto-update for only MachineImageVersion",
			enableKubernetesVersionAutoUpdate:   false,
			enableMachineImageVersionAutoUpdate: true,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			// given
			shoot := fixEmptyGardenerShoot("test", "dev")
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
			extender := NewMaintenanceExtender(testCase.enableKubernetesVersionAutoUpdate, testCase.enableMachineImageVersionAutoUpdate)
			err := extender(runtimeShoot, &shoot)

			// then
			assert.NoError(t, err)
			assert.Equal(t, testCase.enableKubernetesVersionAutoUpdate, shoot.Spec.Maintenance.AutoUpdate.KubernetesVersion)
			assert.Equal(t, testCase.enableMachineImageVersionAutoUpdate, *shoot.Spec.Maintenance.AutoUpdate.MachineImageVersion)
		})
	}
}
