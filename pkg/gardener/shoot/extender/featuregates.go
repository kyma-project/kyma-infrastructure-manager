package extender

import (
	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
)

func NewFeatureGatesExtender(apiServerFeatureGates map[string]bool, kubeletFeatureGates map[string]bool) func(runtime imv1.Runtime, shoot *gardener.Shoot) error {
	return func(runtime imv1.Runtime, shoot *gardener.Shoot) error {
		if shoot.Spec.Kubernetes.KubeAPIServer == nil {
			shoot.Spec.Kubernetes.KubeAPIServer = &gardener.KubeAPIServerConfig{}
		}
		shoot.Spec.Kubernetes.KubeAPIServer.FeatureGates = apiServerFeatureGates

		if shoot.Spec.Kubernetes.Kubelet == nil {
			shoot.Spec.Kubernetes.Kubelet = &gardener.KubeletConfig{}
		}
		shoot.Spec.Kubernetes.Kubelet.FeatureGates = kubeletFeatureGates

		return nil
	}
}
