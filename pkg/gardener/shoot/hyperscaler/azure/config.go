package azure

import (
	"encoding/json"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const infrastructureConfigKind = "InfrastructureConfig"
const controlPlaneConfigKind = "ControlPlaneConfig"
const apiVersion = "azure.provider.extensions.gardener.cloud/v1alpha1"

func GetInfrastructureConfig(workerCIDR string, zones []string) ([]byte, error) {
	config, err := NewInfrastructureConfig(workerCIDR, zones)

	if err != nil {
		return nil, err
	}

	return json.Marshal(config)
}

func GetInfrastructureConfigForPatch(workersCidr string, zones []string, existingInfrastructureConfigBytes []byte) ([]byte, error) {
	newConfig, err := NewInfrastructureConfigForPatch(workersCidr, zones, existingInfrastructureConfigBytes)
	if err != nil {
		return nil, err
	}

	return json.Marshal(newConfig)
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

func NewInfrastructureConfig(workerCIDR string, zones []string) (InfrastructureConfig, error) {
	// All standard Azure shoots are zoned.
	// No zones - old Azure lite shoots where config should be preserved.

	azureZones, err := generateAzureZones(workerCIDR, zones)
	if err != nil {
		return InfrastructureConfig{}, err
	}

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
			Zones: azureZones,
		},
		Zoned: isZoned,
	}

	return azureConfig, nil
}

func NewInfrastructureConfigForPatch(workersCidr string, zones []string, existingInfrastructureConfigBytes []byte) (InfrastructureConfig, error) {
	newConfig, err := NewInfrastructureConfig(workersCidr, zones)
	if err != nil {
		return InfrastructureConfig{}, err
	}

	existingInfrastructureConfig, err := DecodeInfrastructureConfig(existingInfrastructureConfigBytes)

	if err != nil {
		return InfrastructureConfig{}, err
	}
	for _, zone := range existingInfrastructureConfig.Networks.Zones {
		for i := 0; i < len(newConfig.Networks.Zones); i++ {
			newZone := &newConfig.Networks.Zones[i]
			if newZone.Name == zone.Name {
				newZone.NatGateway = zone.NatGateway
				newZone.ServiceEndpoints = zone.ServiceEndpoints
			}
		}
	}

	newConfig.ResourceGroup = existingInfrastructureConfig.ResourceGroup
	newConfig.Networks.VNet = existingInfrastructureConfig.Networks.VNet
	newConfig.Networks.ServiceEndpoints = existingInfrastructureConfig.Networks.ServiceEndpoints
	newConfig.Networks.NatGateway = existingInfrastructureConfig.Networks.NatGateway
	newConfig.Networks.Workers = existingInfrastructureConfig.Networks.Workers

	return newConfig, nil
}
