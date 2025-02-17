package maintenance

import (
	"encoding/json"
	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/pkg/errors"
	"os"
	"sigs.k8s.io/yaml"
)

const (
	BeginMaintenanceWindowKey = "begin"
	EndMaintenanceWindowKey   = "end"
)

func GetMaintenanceWindow(maintenanceWindowConfigPath, region string) (*gardener.MaintenanceTimeWindow, error) {
	timeWindow, err := getWindowForRegion(maintenanceWindowConfigPath, region)

	if err != nil {
		return nil, errors.Errorf("error during getting maintanence window data: %s", err.Error())
	}

	if timeWindow == nil {
		return nil, errors.Errorf("maintenance window is not defined for region: %s", region)
	}

	return &gardener.MaintenanceTimeWindow{Begin: timeWindow.Begin, End: timeWindow.End}, nil
}

func getWindowForRegion(maintenanceWindowConfigPath, region string) (*gardener.MaintenanceTimeWindow, error) {
	windowData, err := getDataFromFile(maintenanceWindowConfigPath, region)

	if err != nil {
		return nil, err
	}

	return &gardener.MaintenanceTimeWindow{Begin: windowData[BeginMaintenanceWindowKey], End: windowData[EndMaintenanceWindowKey]}, nil

}

func getDataFromFile(filepath, region string) (map[string]string, error) {
	fileData, err := os.ReadFile(filepath)
	if err != nil {
		return nil, errors.Errorf("failed to read file: %s", err.Error())
	}

	var config map[string]interface{}
	if err = yaml.Unmarshal(fileData, &config); err != nil {
		return nil, errors.Errorf("failed to unmarshal yaml: %s", err.Error())
	}

	dataField, ok := config["data"].(map[string]interface{})
	if !ok {
		return nil, errors.New("failed to get data field from config map")
	}

	configJSON, ok := dataField["config"].(string)
	if !ok {
		return nil, errors.New("failed to get config field from data")
	}

	var maintenanceWindow map[string]map[string]string
	if err := json.Unmarshal([]byte(configJSON), &maintenanceWindow); err != nil {
		return nil, errors.Errorf("failed to decode json: %s", err.Error())
	}
	return maintenanceWindow[region], nil
}
