package gcp

import (
	"encoding/json"

	"github.com/gardener/gardener-extension-provider-gcp/pkg/apis/gcp/v1alpha1"
	"github.com/pkg/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	infrastructureConfigKind = "InfrastructureConfig"
	controlPlaneConfigKind   = "ControlPlaneConfig"
	apiVersion               = "gcp.provider.extensions.gardener.cloud/v1alpha1"
)

func GetInfrastructureConfig(workerCIDR string, _ []string) ([]byte, error) {
	return json.Marshal(NewInfrastructureConfig(workerCIDR))
}

func GetControlPlaneConfig(zones []string) ([]byte, error) {
	if len(zones) == 0 {
		return nil, errors.New("zones list is empty")
	}

	return json.Marshal(NewControlPlaneConfig(zones))
}

func NewInfrastructureConfig(workerCIDR string) v1alpha1.InfrastructureConfig {
	return v1alpha1.InfrastructureConfig{
		TypeMeta: v1.TypeMeta{
			Kind:       infrastructureConfigKind,
			APIVersion: apiVersion,
		},
		Networks: v1alpha1.NetworkConfig{
			// Provisioner sets also deprecated Worker field.
			// Logic for comparing shoots in integration tests must be adjusted accordingly
			Workers: workerCIDR,
			Worker:  workerCIDR,
		},
	}
}

func NewControlPlaneConfig(zones []string) *v1alpha1.ControlPlaneConfig {
	return &v1alpha1.ControlPlaneConfig{
		TypeMeta: v1.TypeMeta{
			Kind:       controlPlaneConfigKind,
			APIVersion: apiVersion,
		},
		Zone: zones[0],
	}
}
