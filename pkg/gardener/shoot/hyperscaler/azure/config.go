package azure

import (
	"encoding/json"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const infrastructureConfigKind = "InfrastructureConfig"
const controlPlaneConfigKind = "ControlPlaneConfig"
const apiVersion = "azure.provider.extensions.gardener.cloud/v1alpha1"

func GetInfrastructureConfig(workerCIDR string, zones []string) ([]byte, error) {
	return json.Marshal(NewInfrastructureConfig(workerCIDR, zones))
}

func GetControlPlaneConfig(_ []string) ([]byte, error) {
	return json.Marshal(NewControlPlaneConfig())
}

func NewControlPlaneConfig() *ControlPlaneConfig {
	return &ControlPlaneConfig{
		TypeMeta: v1.TypeMeta{
			Kind:       controlPlaneConfigKind,
			APIVersion: apiVersion,
		},
	}
}

func DecodeInfrastructureConfig(data []byte) (*InfrastructureConfig, error) {
	infrastructureConfig := &InfrastructureConfig{}
	err := json.Unmarshal(data, infrastructureConfig)
	if err != nil {
		return nil, err
	}
	return infrastructureConfig, nil
}

func NewInfrastructureConfig(workerCIDR string, zones []string) InfrastructureConfig {
	// All Azure shoots are zoned.
	// No zones - the shoot configuration is invalid.
	// We should validate the config before calling this function.
	isZoned := len(zones) > 0

	azureConfig := InfrastructureConfig{
		TypeMeta: v1.TypeMeta{
			Kind:       infrastructureConfigKind,
			APIVersion: apiVersion,
		},
		Networks: NetworkConfig{
			VNet: VNet{
				CIDR: &workerCIDR,
			},
		},
		Zoned: isZoned,
	}

	azureConfig.Networks.Zones = generateAzureZones(workerCIDR, zones)

	return azureConfig
}
