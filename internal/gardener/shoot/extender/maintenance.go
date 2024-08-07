package extender

import (
	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"k8s.io/utils/ptr"
)

func NewMaintenanceExtender(enableKubernetesVersionAutoUpdate, enableMachineImageVersionAutoUpdate bool) func(runtime imv1.Runtime, shoot *gardener.Shoot) error { //nolint:revive
	return func(runtime imv1.Runtime, shoot *gardener.Shoot) error { //nolint:revive
		shoot.Spec.Maintenance = &gardener.Maintenance{
			AutoUpdate: &gardener.MaintenanceAutoUpdate{
				KubernetesVersion:   enableKubernetesVersionAutoUpdate,
				MachineImageVersion: ptr.To(enableMachineImageVersionAutoUpdate),
			},
		}

		return nil
	}
}
