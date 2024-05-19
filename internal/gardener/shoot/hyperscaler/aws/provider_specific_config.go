package aws

import (
	"encoding/json"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const infrastructureConfigKind = "InfrastructureConfig"
const controlPlaneConfigKind = "ControlPlaneConfig"
const apiVersion = "aws.provider.extensions.gardener.cloud/v1alpha1"

func GetInfrastructureConfig(workersCidr string, zones []string) ([]byte, error) {
	return json.Marshal(NewInfrastructureConfig(workersCidr, zones))
}

func GetControlPlaneConfig() ([]byte, error) {
	return json.Marshal(NewControlPlaneConfig())
}

func NewInfrastructureConfig(workersCidr string, zones []string) InfrastructureConfig {
	return InfrastructureConfig{
		TypeMeta: metav1.TypeMeta{
			Kind:       infrastructureConfigKind,
			APIVersion: apiVersion,
		},
		Networks: Networks{
			Zones: generateAWSZones(workersCidr, zones),
			VPC: VPC{
				CIDR: &workersCidr,
			},
		},
	}
}

func NewControlPlaneConfig() *ControlPlaneConfig {
	return &ControlPlaneConfig{
		TypeMeta: v1.TypeMeta{
			Kind:       controlPlaneConfigKind,
			APIVersion: apiVersion,
		},
	}
}
