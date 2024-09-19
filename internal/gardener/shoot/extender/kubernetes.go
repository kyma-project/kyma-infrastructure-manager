package extender

import (
	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"k8s.io/utils/ptr"
)

// NewKubernetesExtender creates a new Kubernetes extender function.
// It sets the Kubernetes version of the Shoot to the version specified in the Runtime.
// If the version is not specified in the Runtime, it sets the version to the `defaultKubernetesVersion`, set in `converter_config.json`.
// It sets the EnableStaticTokenKubeconfig field of the Shoot to false.
func NewKubernetesExtender(defaultKubernetesVersion string) func(runtime imv1.Runtime, shoot *gardener.Shoot) error {
	return func(runtime imv1.Runtime, shoot *gardener.Shoot) error {
		kubernetesVersion := runtime.Spec.Shoot.Kubernetes.Version
		if kubernetesVersion == nil || *kubernetesVersion == "" {
			kubernetesVersion = &defaultKubernetesVersion
		}

		shoot.Spec.Kubernetes.Version = *kubernetesVersion
		shoot.Spec.Kubernetes.EnableStaticTokenKubeconfig = ptr.To(false)

		return nil
	}
}
