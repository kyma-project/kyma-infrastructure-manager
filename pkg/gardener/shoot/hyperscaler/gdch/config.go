package gdch

import (
	"encoding/json"

	"github.com/kyma-project/infrastructure-manager/pkg/config"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	infrastructureConfigKind = "InfrastructureConfig"
	controlPlaneConfigKind   = "ControlPlaneConfig"
	apiVersion               = "gdch.provider.extensions.gardener.gdc.goog/v1alpha1"
)

func GetInfrastructureConfig(workerCIDR string, zonesName []string, gdhcConfig config.GDCHConfig) ([]byte, error) {
	config, err := NewInfrastructureConfig(workerCIDR, zonesName, gdhcConfig)

	if err != nil {
		return nil, err
	}
	return json.Marshal(config)
}

func NewInfrastructureConfig(workerCIDR string, zonesName []string, gdhcConfig config.GDCHConfig) (*InfrastructureConfig, error) {
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
			ParentReference: ParentReference{
				Name:      gdhcConfig.ParentReferenceName,
				Namespace: gdhcConfig.ParentReferenceNamespace,
				Type:      gdhcConfig.ParentReferenceType,
			},
		},
		EnableEgress: true,
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
