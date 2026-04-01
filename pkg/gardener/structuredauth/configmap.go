package structuredauth

import (
	"context"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

type JWTAuthenticator struct {
	Issuer        Issuer        `json:"issuer"`
	ClaimMappings ClaimMappings `json:"claimMappings"`
}

type Issuer struct {
	URL       string   `json:"url"`
	Audiences []string `json:"audiences"`
}

type ClaimMappings struct {
	Username PrefixedClaim `json:"username"`
	Groups   PrefixedClaim `json:"groups"`
}

type PrefixedClaim struct {
	Claim  string  `json:"claim"`
	Prefix *string `json:"prefix,omitempty"`
}

type AuthenticationConfiguration struct {
	metav1.TypeMeta

	JWT []JWTAuthenticator `json:"jwt"`
}

func toAuthenticationConfiguration(oidcConfig gardener.OIDCConfig) AuthenticationConfiguration {

	toJWTAuthenticator := func(oidcConfig gardener.OIDCConfig) JWTAuthenticator {
		// If Groups prefix is not set by the KEB, default is set as Gardener requires non-empty value
		groupsPrefix := ptr.To("")

		if oidcConfig.GroupsPrefix != nil {
			groupsPrefix = oidcConfig.GroupsPrefix
		}

		return JWTAuthenticator{
			Issuer: Issuer{
				URL:       ptr.Deref(oidcConfig.IssuerURL, ""),
				Audiences: []string{ptr.Deref(oidcConfig.ClientID, "")},
			},
			ClaimMappings: ClaimMappings{
				Username: PrefixedClaim{
					Claim:  ptr.Deref(oidcConfig.UsernameClaim, ""),
					Prefix: oidcConfig.UsernamePrefix,
				},
				Groups: PrefixedClaim{
					Claim:  ptr.Deref(oidcConfig.GroupsClaim, ""),
					Prefix: groupsPrefix,
				},
			},
		}
	}

	jwtAuthenticators := make([]JWTAuthenticator, 0)
	jwtAuthenticators = append(jwtAuthenticators, toJWTAuthenticator(oidcConfig))

	return AuthenticationConfiguration{
		TypeMeta: metav1.TypeMeta{
			Kind:       "AuthenticationConfiguration",
			APIVersion: "apiserver.config.k8s.io/v1beta1",
		},
		JWT: jwtAuthenticators,
	}
}

func CreateOrUpdateStructuredAuthConfigMap(ctx context.Context, gardenClient client.Client, cmKey types.NamespacedName, oidcConfig gardener.OIDCConfig) error {
	creteConfigMapObject := func() (v1.ConfigMap, error) {
		authenticationConfig := toAuthenticationConfiguration(oidcConfig)
		authConfigBytes, err := yaml.Marshal(authenticationConfig)
		if err != nil {
			return v1.ConfigMap{}, err
		}

		return v1.ConfigMap{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ConfigMap",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      cmKey.Name,
				Namespace: cmKey.Namespace,
			},
			Data: map[string]string{
				"config.yaml": string(authConfigBytes),
			},
		}, err
	}

	var existingCM v1.ConfigMap
	err := gardenClient.Get(ctx, cmKey, &existingCM)

	if err != nil && !k8serrors.IsNotFound(err) {
		return err
	}

	configMapAlreadyExists := err == nil

	newConfigMap, err := creteConfigMapObject()

	if err != nil {
		return err
	}

	if configMapAlreadyExists {
		existingCM.Data = newConfigMap.Data
		return gardenClient.Update(ctx, &existingCM)
	}

	return gardenClient.Create(ctx, &newConfigMap)
}

func DeleteStructuredConfigMap(ctx context.Context, gardenClient client.Client, shoot gardener.Shoot) error {
	if shoot.Spec.Kubernetes.KubeAPIServer != nil && shoot.Spec.Kubernetes.KubeAPIServer.StructuredAuthentication != nil {
		cmName := shoot.Spec.Kubernetes.KubeAPIServer.StructuredAuthentication.ConfigMapName

		if cmName == "" {
			return nil
		}

		err := gardenClient.Delete(ctx, &v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      shoot.Spec.Kubernetes.KubeAPIServer.StructuredAuthentication.ConfigMapName,
				Namespace: shoot.Namespace,
			},
		})

		return client.IgnoreNotFound(err)
	}

	return nil
}
