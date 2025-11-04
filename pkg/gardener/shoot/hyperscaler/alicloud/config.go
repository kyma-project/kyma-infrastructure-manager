package alicloud

import (
	"encoding/json"

	"github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	infrastructureConfigKind = "InfrastructureConfig"
	controlPlaneConfigKind   = "ControlPlaneConfig"
	apiVersion               = "alicloud.provider.extensions.gardener.cloud/v1alpha1"
)

func GetInfrastructureConfig(workerCIDR string, zones []string) ([]byte, error) {
	return json.Marshal(NewInfrastructureConfig(workerCIDR, zones))
}

func GetControlPlaneConfig(_ []string) ([]byte, error) {
	return json.Marshal(NewControlPlaneConfig())
}

func NewInfrastructureConfig(workerCIDR string, zones []string) v1alpha1.InfrastructureConfig {
	var networkZones = make([]v1alpha1.Zone, len(zones))
	for i, zone := range zones {
		networkZones[i] = v1alpha1.Zone{
			Name: zone,
			Workers: workerCIDR,
		}
	}

	return v1alpha1.InfrastructureConfig {

		TypeMeta: v1.TypeMeta{
			Kind:       infrastructureConfigKind,
			APIVersion: apiVersion,
		},
		Networks: v1alpha1.Networks{
			VPC:   v1alpha1.VPC{CIDR: &workerCIDR}, //TODO: verify if this is correct
			Zones: networkZones,
		},
	}
}

func NewControlPlaneConfig() *v1alpha1.ControlPlaneConfig {
	return &v1alpha1.ControlPlaneConfig{
		TypeMeta: v1.TypeMeta{
			Kind:       controlPlaneConfigKind,
			APIVersion: apiVersion,
		},
	}
}