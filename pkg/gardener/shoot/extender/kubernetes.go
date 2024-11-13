package extender

import (
	"github.com/Masterminds/semver/v3"
	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"k8s.io/utils/ptr"
)

// NewKubernetesExtender creates a new Kubernetes extender function.
// It sets the Kubernetes version of the Shoot to the version specified in the Runtime.
// If the version is not specified in the Runtime, it sets the version to the `defaultKubernetesVersion`, set in `converter_config.json`.
// If the current Kubernetes version on Shoot is greater than the version determined above, it sets the version to the current Kubernetes version.
// It sets the EnableStaticTokenKubeconfig field of the Shoot to false.
func NewKubernetesExtender(defaultKubernetesVersion, currentKubernetesVersion string) func(runtime imv1.Runtime, shoot *gardener.Shoot) error {
	return func(runtime imv1.Runtime, shoot *gardener.Shoot) error {
		kubernetesVersion := runtime.Spec.Shoot.Kubernetes.Version
		if kubernetesVersion == nil || *kubernetesVersion == "" {
			kubernetesVersion = &defaultKubernetesVersion
		}

		shoot.Spec.Kubernetes.Version = *kubernetesVersion

		// use current Kubernetes version from shoot when it is greater than determined above - autoupdate case
		if currentKubernetesVersion != "" && currentKubernetesVersion != shoot.Spec.Kubernetes.Version {
			result, err := compareVersions(shoot.Spec.Kubernetes.Version, currentKubernetesVersion)
			if err == nil && result < 0 {
				shoot.Spec.Kubernetes.Version = currentKubernetesVersion
			}
		}

		shoot.Spec.Kubernetes.EnableStaticTokenKubeconfig = ptr.To(false)

		return nil
	}
}

func compareVersions(version1, version2 string) (int, error) {
	v1, err := semver.NewVersion(version1)
	if err != nil {
		return 0, err
	}

	v2, err := semver.NewVersion(version2)
	if err != nil {
		return 0, err
	}

	return v1.Compare(v2), nil
}
