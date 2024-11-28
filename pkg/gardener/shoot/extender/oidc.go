package extender

import (
	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/pkg/config"
	"k8s.io/utils/ptr"
)

const (
	OidcExtensionType = "shoot-oidc-service"
)

func shouldDefaultOidcConfig(config gardener.OIDCConfig) bool {
	return config.ClientID == nil && config.IssuerURL == nil
}

func NewOidcExtenderFromShoot(oidcProvider config.OidcProvider, extensions []gardener.Extension) func(runtime imv1.Runtime, shoot *gardener.Shoot) error {
	return func(runtime imv1.Runtime, shoot *gardener.Shoot) error {
		oidcExtension := func() *gardener.Extension {
			for _, extension := range extensions {
				if extension.Type == "shoot-oidc-service" {
					return &extension
				}
			}
			return nil
		}()

		if oidcExtension != nil {
			shoot.Spec.Extensions = append(shoot.Spec.Extensions, *oidcExtension)
		}

		oidcConfig := runtime.Spec.Shoot.Kubernetes.KubeAPIServer.OidcConfig
		if shouldDefaultOidcConfig(oidcConfig) {
			oidcConfig = gardener.OIDCConfig{
				ClientID:       &oidcProvider.ClientID,
				GroupsClaim:    &oidcProvider.GroupsClaim,
				IssuerURL:      &oidcProvider.IssuerURL,
				SigningAlgs:    oidcProvider.SigningAlgs,
				UsernameClaim:  &oidcProvider.UsernameClaim,
				UsernamePrefix: &oidcProvider.UsernamePrefix,
			}
		}
		setKubeAPIServerOIDCConfig(shoot, oidcConfig)

		return nil
	}
}

func NewOidcExtender(oidcProvider config.OidcProvider) func(runtime imv1.Runtime, shoot *gardener.Shoot) error {
	return func(runtime imv1.Runtime, shoot *gardener.Shoot) error {
		if CanEnableExtension(runtime) {
			setOIDCExtension(shoot)
		}

		oidcConfig := runtime.Spec.Shoot.Kubernetes.KubeAPIServer.OidcConfig
		if shouldDefaultOidcConfig(oidcConfig) {
			oidcConfig = gardener.OIDCConfig{
				ClientID:       &oidcProvider.ClientID,
				GroupsClaim:    &oidcProvider.GroupsClaim,
				IssuerURL:      &oidcProvider.IssuerURL,
				SigningAlgs:    oidcProvider.SigningAlgs,
				UsernameClaim:  &oidcProvider.UsernameClaim,
				UsernamePrefix: &oidcProvider.UsernamePrefix,
			}
		}
		setKubeAPIServerOIDCConfig(shoot, oidcConfig)

		return nil
	}
}

func CanEnableExtension(runtime imv1.Runtime) bool {
	return runtime.Labels["operator.kyma-project.io/created-by-migrator"] != "true"
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
