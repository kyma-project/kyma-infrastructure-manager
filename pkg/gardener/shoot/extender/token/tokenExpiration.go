package token

import (
	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewExpirationTimeExtender(maxTokenExpiration metav1.Duration, extendTokenExpiration *bool) func(_ imv1.Runtime, shoot *gardener.Shoot) error {
	return func(_ imv1.Runtime, shoot *gardener.Shoot) error {
		if shoot.Spec.Kubernetes.KubeAPIServer == nil {
			shoot.Spec.Kubernetes.KubeAPIServer = &gardener.KubeAPIServerConfig{}
		}
		if shoot.Spec.Kubernetes.KubeAPIServer.ServiceAccountConfig == nil {
			shoot.Spec.Kubernetes.KubeAPIServer.ServiceAccountConfig = &gardener.ServiceAccountConfig{}
		}
		shoot.Spec.Kubernetes.KubeAPIServer.ServiceAccountConfig.MaxTokenExpiration = &maxTokenExpiration
		shoot.Spec.Kubernetes.KubeAPIServer.ServiceAccountConfig.ExtendTokenExpiration = extendTokenExpiration
		return nil
	}
}
