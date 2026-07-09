package gdch

import (
	"encoding/json"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	infrastructureConfigKind = "InfrastructureConfig"
	controlPlaneConfigKind   = "ControlPlaneConfig"
	apiVersion               = "gdch.provider.extensions.gardener.gdc.goog/v1alpha1"
)

func GetInfrastructureConfig(workerCIDR string, zonesName []string) ([]byte, error) {
	config, err := NewInfrastructureConfig(workerCIDR, zonesName)

	if err != nil {
		return nil, err
	}
	return json.Marshal(config)
}

func NewInfrastructureConfig(workerCIDR string, zonesName []string) (*InfrastructureConfig, error) {
	gdchZones, err := generateGDCHZones(workerCIDR, zonesName)
	if err != nil {
		return &InfrastructureConfig{}, err
	}

	gdchConfig := &InfrastructureConfig{
		TypeMeta: metav1.TypeMeta{
			Kind:       infrastructureConfigKind,
			APIVersion: apiVersion,
		},
		Networks: NetworkConfig{
			NodeCIDR: workerCIDR,
			Zones:    gdchZones,
		},
	}

	return gdchConfig, nil
}

func GetControlPlaneConfig(_ []string) ([]byte, error) {
	return json.Marshal(NewControlPlaneConfig())
}

func NewControlPlaneConfig() *ControlPlaneConfig {
	return &ControlPlaneConfig{
		TypeMeta: metav1.TypeMeta{
			Kind:       controlPlaneConfigKind,
			APIVersion: apiVersion,
		},
	}
}
