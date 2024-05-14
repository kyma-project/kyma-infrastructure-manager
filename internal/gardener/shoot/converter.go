package shoot

import (
	gardenerv1beta "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Converter struct {
	runtime imv1.Runtime
}

type Extender func(imv1.Runtime, *gardenerv1beta.Shoot) error

func dns(imv1.Runtime, *gardenerv1beta.Shoot) error {
	return nil
}

func oidc(imv1.Runtime, *gardenerv1beta.Shoot) error {
	return nil
}

func networking(imv1.Runtime, *gardenerv1beta.Shoot) error {
	return nil
}

func (c Converter) ToShoot() gardenerv1beta.Shoot {
	s := gardenerv1beta.Shoot{
		ObjectMeta: v1.ObjectMeta{
			Name:        c.runtime.Spec.Shoot.Name,
			Namespace:   c.runtime.Namespace,
			Labels:      getLabels(c.runtime),
			Annotations: getAnnotations(c.runtime),
		},
		Spec: gardenerv1beta.ShootSpec{},
		//	getShootSpec(c.runtime.Spec.Shoot),
	}

	extenders := []Extender{
		dns,
		oidc,
		networking,
	}

	for _, e := range extenders {

	}

}

func getLabels(_ imv1.Runtime) map[string]string {
	return map[string]string{}
}

func getAnnotations(_ imv1.Runtime) map[string]string {
	return map[string]string{}
}

func getShootSpec(runtimeShoot imv1.RuntimeShoot) gardenerv1beta.ShootSpec {
	return gardenerv1beta.ShootSpec{
		Purpose:           &runtimeShoot.Purpose,
		Region:            runtimeShoot.Region,
		SecretBindingName: &runtimeShoot.SecretBindingName,
		Kubernetes:        getKubernetes(runtimeShoot.Kubernetes),
		Networking:        getNetworking(runtimeShoot.Networking),
		Provider:          getProvider(runtimeShoot.Provider),
		ControlPlane:      &runtimeShoot.ControlPlane,
	}
}

func getKubernetes(kubernetes imv1.Kubernetes) gardenerv1beta.Kubernetes {
	return gardenerv1beta.Kubernetes{
		Version: getKubernetesVersion(kubernetes),
		KubeAPIServer: &gardenerv1beta.KubeAPIServerConfig{
			OIDCConfig: getOIDCConfig(kubernetes.KubeAPIServer.OidcConfig),
		},
	}
}

func getKubernetesVersion(kubernetes imv1.Kubernetes) string {
	if kubernetes.Version != nil {
		return *kubernetes.Version
	}

	// Determine the default Kubernetes version
	// it must be read from the configuration (please refer to KEB)
	return ""
}

func getOIDCConfig(oidcConfig gardenerv1beta.OIDCConfig) *gardenerv1beta.OIDCConfig {
	return &gardenerv1beta.OIDCConfig{
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
}

func getProvider(runtimeProvider imv1.Provider) gardenerv1beta.Provider {
	return gardenerv1beta.Provider{
		Type:                 runtimeProvider.Type,
		ControlPlaneConfig:   &runtimeProvider.ControlPlaneConfig,
		InfrastructureConfig: &runtimeProvider.InfrastructureConfig,
		Workers:              runtimeProvider.Workers,
	}
}

func getNetworking(runtimeNetworking imv1.Networking) *gardenerv1beta.Networking {
	return &gardenerv1beta.Networking{
		Nodes:    &runtimeNetworking.Nodes,
		Pods:     &runtimeNetworking.Pods,
		Services: &runtimeNetworking.Services,
	}
}
