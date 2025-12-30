package extender

import (
	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
)

func NewFeatureGatesExtender(featureGates map[string]bool) func(runtime imv1.Runtime, shoot *gardener.Shoot) error {
	return func(runtime imv1.Runtime, shoot *gardener.Shoot) error {
		if shoot.Spec.Kubernetes.Kubelet == nil {
			shoot.Spec.Kubernetes.Kubelet = &gardener.KubeletConfig{}
		}
		shoot.Spec.Kubernetes.Kubelet.FeatureGates = featureGates

		return nil
	}
}
