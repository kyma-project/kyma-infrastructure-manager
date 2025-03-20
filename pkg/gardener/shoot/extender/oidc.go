package extender

import (
	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
)

const (
	OidcExtensionType = "shoot-oidc-service"
)

func NewOidcExtender(authenticationConfigurationConfigMap string) func(runtime imv1.Runtime, shoot *gardener.Shoot) error {
	return func(runtime imv1.Runtime, shoot *gardener.Shoot) error {
		shoot.Spec.Kubernetes.KubeAPIServer = &gardener.KubeAPIServerConfig{
			StructuredAuthentication: &gardener.StructuredAuthentication{
				ConfigMapName: authenticationConfigurationConfigMap,
			},
		}

		return nil
	}
}
