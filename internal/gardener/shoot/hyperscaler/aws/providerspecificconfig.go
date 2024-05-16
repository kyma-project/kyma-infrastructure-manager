package aws

import (
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const infrastructureConfigKind = "InfrastructureConfig"
const controlPlaneConfigKind = "ControlPlaneConfig"
const apiVersion = "aws.provider.extensions.gardener.cloud/v1alpha1"

func GetInfrastructureConfig(runtimeShoot imv1.RuntimeShoot) ([]byte, error) {
	return nil, nil
}

func GetControlPlaneConfig(runtimeShoot imv1.RuntimeShoot) ([]byte, error) {
	return nil, nil
}

func ToInfrastructure(shoot imv1.RuntimeShoot) InfrastructureConfig {
	return InfrastructureConfig{
		TypeMeta: metav1.TypeMeta{
			Kind:       infrastructureConfigKind,
			APIVersion: apiVersion,
		},
		Networks: Networks{
			Zones: generateAWSZones(shoot.Networking.Nodes, shoot.Provider.Zones),
			VPC: VPC{
				CIDR: &shoot.Networking.Nodes,
			},
		},
	}
	return InfrastructureConfig{}
}
