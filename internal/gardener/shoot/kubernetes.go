package shoot

import (
	gardenerv1beta "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
)

func newKubernetesExtender(defaultKubernetesVersion string) extender {
	return func(runtime imv1.RuntimeShoot, shoot *gardenerv1beta.Shoot) error {
		kubernetesVersion := runtime.Kubernetes.Version
		if kubernetesVersion != nil && *kubernetesVersion != "" {
			kubernetesVersion = &defaultKubernetesVersion
		}

		shoot.Spec.Kubernetes.Version = *kubernetesVersion

		oidcConfig := runtime.Kubernetes.KubeAPIServer.OidcConfig

		shoot.Spec.Kubernetes.KubeAPIServer.OIDCConfig = &gardenerv1beta.OIDCConfig{
			CABundle:       oidcConfig.CABundle,
			ClientID:       oidcConfig.ClientID,
			GroupsClaim:    oidcConfig.GroupsClaim,
			GroupsPrefix:   oidcConfig.GroupsPrefix,
			IssuerURL:      oidcConfig.IssuerURL,
			RequiredClaims: oidcConfig.RequiredClaims,
			SigningAlgs:    oidcConfig.SigningAlgs,
			UsernameClaim:  oidcConfig.UsernameClaim,
			UsernamePrefix: oidcConfig.UsernamePrefix,
		}

		return nil
	}
}
