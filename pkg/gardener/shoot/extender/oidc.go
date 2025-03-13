package extender

import (
	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
)

func NewOidcExtender() func(runtime imv1.Runtime, shoot *gardener.Shoot) error {
	return func(runtime imv1.Runtime, shoot *gardener.Shoot) error {

		// shoot.Spec.Kubernetes.KubeAPIServer.OIDCConfig will be no longer supported, so we need to remove it
		shoot.Spec.Kubernetes.KubeAPIServer = &gardener.KubeAPIServerConfig{
			OIDCConfig: nil,
		}

		return nil
	}
}
