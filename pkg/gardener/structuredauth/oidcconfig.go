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
