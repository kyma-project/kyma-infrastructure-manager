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

func NewFeatureGatesExtender(apiServerFeatureGates map[string]bool, kubeletFeatureGates map[string]bool, defaultKubernetesVersion, currentKubernetesVersion string) func(runtime imv1.Runtime, shoot *gardener.Shoot) error {
	return func(runtime imv1.Runtime, shoot *gardener.Shoot) error {
		if shoot.Spec.Kubernetes.KubeAPIServer == nil {
			shoot.Spec.Kubernetes.KubeAPIServer = &gardener.KubeAPIServerConfig{}
		}
		shoot.Spec.Kubernetes.KubeAPIServer.FeatureGates = apiServerFeatureGates

		if shoot.Spec.Kubernetes.Kubelet == nil {
			shoot.Spec.Kubernetes.Kubelet = &gardener.KubeletConfig{}
		}
		shoot.Spec.Kubernetes.Kubelet.FeatureGates = filterKubeletFeatureGates(kubeletFeatureGates, EffectiveKubernetesVersion(runtime, defaultKubernetesVersion, currentKubernetesVersion))

		return nil
	}
}

// filterKubeletFeatureGates removes the kubelet ClusterTrustBundle feature gate for Kubernetes versions lower than 1.35.
func filterKubeletFeatureGates(kubeletFeatureGates map[string]bool, kubernetesVersion string) map[string]bool {
	if _, found := kubeletFeatureGates[kubeletClusterTrustBundleFeatureGate]; !found {
		return kubeletFeatureGates
	}

	result, err := CompareVersions(kubernetesVersion, kubeletClusterTrustBundleMinVersion)
	if err == nil && result >= 0 {
		return kubeletFeatureGates
	}

	filtered := maps.Clone(kubeletFeatureGates)
	delete(filtered, kubeletClusterTrustBundleFeatureGate)

	return filtered
}
