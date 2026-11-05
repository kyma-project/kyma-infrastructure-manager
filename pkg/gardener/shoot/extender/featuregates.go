package extender

import (
	"maps"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
)

const (
	kubeletClusterTrustBundleFeatureGate = "ClusterTrustBundle"
	// See: https://v1-35.docs.kubernetes.io/docs/concepts/storage/projected-volumes/#clustertrustbundle
	kubeletClusterTrustBundleMinVersion = "1.35"
)

func NewFeatureGatesExtender(apiServerFeatureGates map[string]bool, kubeletFeatureGates map[string]bool, defaultKubernetesVersion string) func(runtime imv1.Runtime, shoot *gardener.Shoot) error {
	return func(runtime imv1.Runtime, shoot *gardener.Shoot) error {
		if shoot.Spec.Kubernetes.KubeAPIServer == nil {
			shoot.Spec.Kubernetes.KubeAPIServer = &gardener.KubeAPIServerConfig{}
		}
		shoot.Spec.Kubernetes.KubeAPIServer.FeatureGates = apiServerFeatureGates

		if shoot.Spec.Kubernetes.Kubelet == nil {
			shoot.Spec.Kubernetes.Kubelet = &gardener.KubeletConfig{}
		}
		shoot.Spec.Kubernetes.Kubelet.FeatureGates = filterKubeletFeatureGates(kubeletFeatureGates, runtime, defaultKubernetesVersion)

		return nil
	}
}

// filterKubeletFeatureGates removes the kubelet ClusterTrustBundle feature gate for Kubernetes versions lower than 1.35.
// The Runtime CR does not contain the actual (possibly auto-updated) Shoot version, so the requested version
// (or the default one) is used. This never enables the gate on a cluster that is older than 1.35.
// See: https://v1-35.docs.kubernetes.io/docs/concepts/storage/projected-volumes/#clustertrustbundle
// This code can be removed once the minimum supported Kubernetes version is 1.35 or higher.
func filterKubeletFeatureGates(kubeletFeatureGates map[string]bool, runtime imv1.Runtime, defaultKubernetesVersion string) map[string]bool {
	if _, found := kubeletFeatureGates[kubeletClusterTrustBundleFeatureGate]; !found {
		return kubeletFeatureGates
	}

	kubernetesVersion := defaultKubernetesVersion
	if runtime.Spec.Shoot.Kubernetes.Version != nil && *runtime.Spec.Shoot.Kubernetes.Version != "" {
		kubernetesVersion = *runtime.Spec.Shoot.Kubernetes.Version
	}

	result, err := CompareVersions(kubernetesVersion, kubeletClusterTrustBundleMinVersion)
	if err == nil && result >= 0 {
		return kubeletFeatureGates
	}

	filtered := maps.Clone(kubeletFeatureGates)
	delete(filtered, kubeletClusterTrustBundleFeatureGate)

	return filtered
}
