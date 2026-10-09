package structuredauth

import (
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
)

func GetOIDCConfigOrDefault(runtime imv1.Runtime, defaultOIDC imv1.GardenerOIDCConfig) imv1.GardenerOIDCConfig {
	oidcConfig := runtime.Spec.Shoot.Kubernetes.KubeAPIServer.OidcConfig

	if oidcConfig.IssuerURL == nil || oidcConfig.ClientID == nil {
		return defaultOIDC
	}

	return oidcConfig
}
