package openstack

import (
	"encoding/json"

	"github.com/gardener/gardener-extension-provider-openstack/pkg/apis/openstack/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	infrastructureConfigKind    = "InfrastructureConfig"
	controlPlaneConfigKind      = "ControlPlaneConfig"
	apiVersion                  = "openstack.provider.extensions.gardener.cloud/v1alpha1"
	DefaultFloatingPoolName     = "FloatingIP-external-kyma-01"
	defaultLoadBalancerProvider = "f5"
)

func GetInfrastructureConfig(workerCIDR string, floatingPoolName string) ([]byte, error) {
	return json.Marshal(NewInfrastructureConfig(workerCIDR, floatingPoolName))
}

func GetControlPlaneConfig(_ []string) ([]byte, error) {
	return json.Marshal(NewControlPlaneConfig())
}

func NewInfrastructureConfig(workerCIDR string, floatingPoolName string) v1alpha1.InfrastructureConfig {
	if floatingPoolName == "" {
		floatingPoolName = DefaultFloatingPoolName
	}

	return v1alpha1.InfrastructureConfig{
		TypeMeta: v1.TypeMeta{
			Kind:       infrastructureConfigKind,
			APIVersion: apiVersion,
		},
		FloatingPoolName: floatingPoolName,
		Networks: v1alpha1.Networks{
			Workers: workerCIDR,
		},
	}
}

func NewControlPlaneConfig() *v1alpha1.ControlPlaneConfig {
	return &v1alpha1.ControlPlaneConfig{
		TypeMeta: v1.TypeMeta{
			Kind:       controlPlaneConfigKind,
			APIVersion: apiVersion,
		},
		LoadBalancerProvider: defaultLoadBalancerProvider,
	}
}

// FloatingPoolNameFromInfrastructureConfig extracts the floating pool name from an existing
// (JSON-encoded) InfrastructureConfig. Used on patch to keep the pool fixed at its creation
// value, since it is immutable landscape configuration.
func FloatingPoolNameFromInfrastructureConfig(raw []byte) (string, error) {
	var cfg v1alpha1.InfrastructureConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return "", err
	}
	return cfg.FloatingPoolName, nil
}
