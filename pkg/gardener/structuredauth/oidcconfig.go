package structuredauth

import (
	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
)

func GetOIDCConfigOrDefault(runtime imv1.Runtime, defaultOIDC gardener.OIDCConfig) gardener.OIDCConfig {
	oidcConfig := runtime.Spec.Shoot.Kubernetes.KubeAPIServer.OidcConfig

	if oidcConfig.IssuerURL == nil || oidcConfig.ClientID == nil {
		return defaultOIDC
	}

	return oidcConfig
}

func OIDCConfigured(shoot gardener.Shoot) bool {
	if shoot.Spec.Kubernetes.KubeAPIServer == nil {
		return false
	}

	oidcConfig := shoot.Spec.Kubernetes.KubeAPIServer.OIDCConfig

	return oidcConfig != nil && oidcConfig.IssuerURL != nil && oidcConfig.ClientID != nil
}
