package aws

import (
	"encoding/json"
	"github.com/gardener/gardener-extension-provider-aws/pkg/apis/aws/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const apiVersion = "aws.provider.extensions.gardener.cloud/v1alpha1"
const infrastructureConfigKind = "InfrastructureConfig"
const controlPlaneConfigKind = "ControlPlaneConfig"
const workerConfigKind = "WorkerConfig"

const awsIMDSv2HTTPPutResponseHopLimit int64 = 2

func GetInfrastructureConfig(workersCidr string, zones []string) ([]byte, error) {
	return json.Marshal(NewInfrastructureConfig(workersCidr, zones))
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

func GetWorkerConfig() ([]byte, error) {
	return json.Marshal(NewWorkerConfig())
}

func DecodeInfrastructureConfig(data []byte) (*v1alpha1.InfrastructureConfig, error) {
	infrastructureConfig := &v1alpha1.InfrastructureConfig{}
	err := json.Unmarshal(data, infrastructureConfig)
	if err != nil {
		return nil, err
	}
	return infrastructureConfig, nil
}

func NewInfrastructureConfig(workersCidr string, zones []string) v1alpha1.InfrastructureConfig {
	return v1alpha1.InfrastructureConfig{
		TypeMeta: metav1.TypeMeta{
			Kind:       infrastructureConfigKind,
			APIVersion: apiVersion,
		},
		Networks: v1alpha1.Networks{
			Zones: generateAWSZones(workersCidr, zones),
			VPC: v1alpha1.VPC{
				CIDR: &workersCidr,
			},
		},
	}
}

func NewInfrastructureConfigForPatch(workersCidr string, zones []string, existingInfrastructureConfigBytes []byte) (v1alpha1.InfrastructureConfig, error) {
	newConfig := NewInfrastructureConfig(workersCidr, zones)
	existingInfrastructureConfig, err := DecodeInfrastructureConfig(existingInfrastructureConfigBytes)

	if err != nil {
		return v1alpha1.InfrastructureConfig{}, err
	}
	existingInfrastructureConfig.Networks.Zones = newConfig.Networks.Zones
	existingInfrastructureConfig.Networks.VPC.CIDR = newConfig.Networks.VPC.CIDR

	return *existingInfrastructureConfig, nil
}

func NewControlPlaneConfig() *v1alpha1.ControlPlaneConfig {
	return &v1alpha1.ControlPlaneConfig{
		TypeMeta: metav1.TypeMeta{
			Kind:       controlPlaneConfigKind,
			APIVersion: apiVersion,
		},
	}
}

func NewWorkerConfig() *v1alpha1.WorkerConfig {
	httpTokens := v1alpha1.HTTPTokensRequired
	hopLimit := awsIMDSv2HTTPPutResponseHopLimit

	return &v1alpha1.WorkerConfig{
		TypeMeta: metav1.TypeMeta{
			APIVersion: apiVersion,
			Kind:       workerConfigKind,
		},
		InstanceMetadataOptions: &v1alpha1.InstanceMetadataOptions{
			HTTPTokens:              &httpTokens,
			HTTPPutResponseHopLimit: &hopLimit,
		},
	}
}
