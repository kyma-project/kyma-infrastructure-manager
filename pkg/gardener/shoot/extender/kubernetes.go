package extender

import (
	"github.com/Masterminds/semver/v3"
	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
)

// NewKubernetesExtender creates a new Kubernetes extender function.
// It sets the Kubernetes version of the Shoot to the version specified in the Runtime.
// If the version is not specified in the Runtime, it sets the version to the `defaultKubernetesVersion`, set in `converter_config.json`.
// If the current Kubernetes version on Shoot is greater than the version determined above, it sets the version to the current Kubernetes version.
func NewKubernetesExtender(defaultKubernetesVersion, currentKubernetesVersion string) func(runtime imv1.Runtime, shoot *gardener.Shoot) error {
	return func(runtime imv1.Runtime, shoot *gardener.Shoot) error {
		shoot.Spec.Kubernetes.Version = EffectiveKubernetesVersion(runtime, defaultKubernetesVersion, currentKubernetesVersion)

		return nil
	}
}

// EffectiveKubernetesVersion returns the Kubernetes version from the Runtime, or `defaultKubernetesVersion` if not set.
// If `currentKubernetesVersion` (the version on the existing Shoot) is greater, it is returned instead - autoupdate case.
func EffectiveKubernetesVersion(runtime imv1.Runtime, defaultKubernetesVersion, currentKubernetesVersion string) string {
	kubernetesVersion := defaultKubernetesVersion
	if runtime.Spec.Shoot.Kubernetes.Version != nil && *runtime.Spec.Shoot.Kubernetes.Version != "" {
		kubernetesVersion = *runtime.Spec.Shoot.Kubernetes.Version
	}

	if currentKubernetesVersion != "" && currentKubernetesVersion != kubernetesVersion {
		result, err := CompareVersions(kubernetesVersion, currentKubernetesVersion)
		if err == nil && result < 0 {
			return currentKubernetesVersion
		}
	}

	return kubernetesVersion
}

func CompareVersions(prevVersion, currVersion string) (int, error) {
	v1, err := semver.NewVersion(prevVersion)
	if err != nil {
		return 0, err
	}

	v2, err := semver.NewVersion(currVersion)
	if err != nil {
		return 0, err
	}

	return v1.Compare(v2), nil
}
