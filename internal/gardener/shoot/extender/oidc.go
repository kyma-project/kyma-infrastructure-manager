package extender

import (
	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"k8s.io/utils/ptr"
)

const (
	OidcExtensionType = "shoot-oidc-service"
)

func ExtendWithOIDC(runtime imv1.Runtime, shoot *gardener.Shoot) error {
	oidcConfig := runtime.Spec.Shoot.Kubernetes.KubeAPIServer.OidcConfig

	setOIDCExtension(shoot)
	setKubeAPIServerOIDCConfig(shoot, oidcConfig)

	return nil
}

func setOIDCExtension(shoot *gardener.Shoot) {
	oidcService := gardener.Extension{
		Type:     OidcExtensionType,
		Disabled: ptr.To(false),
	}

	shoot.Spec.Extensions = append(shoot.Spec.Extensions, oidcService)
}

func setKubeAPIServerOIDCConfig(shoot *gardener.Shoot, oidcConfig gardener.OIDCConfig) {
	shoot.Spec.Kubernetes.KubeAPIServer = &gardener.KubeAPIServerConfig{
		OIDCConfig: &gardener.OIDCConfig{
			CABundle:       oidcConfig.CABundle,
			ClientID:       oidcConfig.ClientID,
			GroupsClaim:    oidcConfig.GroupsClaim,
			GroupsPrefix:   oidcConfig.GroupsPrefix,
			IssuerURL:      oidcConfig.IssuerURL,
			RequiredClaims: oidcConfig.RequiredClaims,
			SigningAlgs:    oidcConfig.SigningAlgs,
			UsernameClaim:  oidcConfig.UsernameClaim,
			UsernamePrefix: oidcConfig.UsernamePrefix,
		},
	}
}
