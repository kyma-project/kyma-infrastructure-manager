package fsm

import (
	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/extender/maintenance"
)

func getMaintenanceTimeWindow(s *systemState, m *fsm) *gardener.MaintenanceTimeWindow {
	var maintenanceWindowData *gardener.MaintenanceTimeWindow
	if s.instance.Spec.Shoot.Purpose == "production" && m.ConverterConfig.MaintenanceWindow.WindowMapPath != "" {
		var err error
		maintenanceWindowData, err = maintenance.GetMaintenanceWindow(m.ConverterConfig.MaintenanceWindow.WindowMapPath, s.instance.Spec.Shoot.Region)
		if err != nil {
			m.log.Error(err, "Failed to get Maintenance Window data for region", "Region", s.instance.Spec.Shoot.Region)
		}
	}
	return maintenanceWindowData
}
