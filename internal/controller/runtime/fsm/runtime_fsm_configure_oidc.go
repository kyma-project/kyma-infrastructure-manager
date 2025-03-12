package fsm

import (
	"context"
	"fmt"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/extender/extensions"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	authenticationv1alpha1 "github.com/gardener/oidc-webhook-authenticator/apis/authentication/v1alpha1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/pkg/config"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apiserver/pkg/apis/apiserver"
	ctrl "sigs.k8s.io/controller-runtime"
	k8s_client "sigs.k8s.io/controller-runtime/pkg/client"
)

func sFnConfigureOidc(ctx context.Context, m *fsm, s *systemState) (stateFn, *ctrl.Result, error) {
	m.log.Info("Configure OIDC state")

	if !isOidcExtensionEnabled(*s.shoot) {
		m.log.Info("OIDC extension is disabled")
		s.instance.UpdateStatePending(
			imv1.ConditionTypeOidcConfigured,
			imv1.ConditionReasonOidcConfigured,
			"True",
			"OIDC extension disabled",
		)

		return switchState(sFnApplyClusterRoleBindings)
	}

	if !multiOidcSupported(s.instance) {
		// New OIDC functionality is supported only for new clusters
		m.log.Info("Multi OIDC is not supported for migrated runtimes")
		s.instance.UpdateStatePending(
			imv1.ConditionTypeOidcConfigured,
			imv1.ConditionReasonOidcConfigured,
			"True",
			"Multi OIDC not supported for migrated runtimes",
		)
		return switchState(sFnApplyClusterRoleBindings)
	}

	defaultAdditionalOidcIfNotPresent(&s.instance, m.RCCfg)
	err := recreateOpenIDConnectResources(ctx, m, s)

	if err != nil {
		updateConditionFailed(&s.instance)
		m.log.Error(err, "Failed to create OpenIDConnect resource. Scheduling for retry")
		return requeue()
	}

	err = createOIDCConfigMap(ctx, &s.instance, m.RCCfg, m, s)

	if err != nil {
		updateConditionFailed(&s.instance)
		m.log.Error(err, "Failed to create structured authentication config map. Scheduling for retry")
		return requeue()
	}

	m.log.Info("OIDC has been configured", "Name", s.shoot.Name)
	s.instance.UpdateStatePending(
		imv1.ConditionTypeOidcConfigured,
		imv1.ConditionReasonOidcConfigured,
		"True",
		"OIDC configuration completed",
	)

	return switchState(sFnApplyClusterRoleBindings)
}

func defaultAdditionalOidcIfNotPresent(runtime *imv1.Runtime, cfg RCCfg) {
	additionalOidcConfig := runtime.Spec.Shoot.Kubernetes.KubeAPIServer.AdditionalOidcConfig

	if additionalOidcConfig == nil {
		additionalOidcConfig = &[]gardener.OIDCConfig{}
		defaultOIDCConfig := createDefaultOIDCConfig(cfg.ClusterConfig.DefaultSharedIASTenant)
		*additionalOidcConfig = append(*additionalOidcConfig, defaultOIDCConfig)
		runtime.Spec.Shoot.Kubernetes.KubeAPIServer.AdditionalOidcConfig = additionalOidcConfig
	}
}

func createDefaultOIDCConfig(defaultSharedIASTenant config.OidcProvider) gardener.OIDCConfig {
	return gardener.OIDCConfig{
		ClientID:       &defaultSharedIASTenant.ClientID,
		GroupsClaim:    &defaultSharedIASTenant.GroupsClaim,
		IssuerURL:      &defaultSharedIASTenant.IssuerURL,
		SigningAlgs:    defaultSharedIASTenant.SigningAlgs,
		UsernameClaim:  &defaultSharedIASTenant.UsernameClaim,
		UsernamePrefix: &defaultSharedIASTenant.UsernamePrefix,
	}
}

func recreateOpenIDConnectResources(ctx context.Context, m *fsm, s *systemState) error {
	shootAdminClient, shootClientError := GetShootClient(ctx, m.Client, s.instance)
	if shootClientError != nil {
		return shootClientError
	}

	err := deleteExistingKymaOpenIDConnectResources(ctx, shootAdminClient)
	if err != nil {
		return err
	}

	additionalOidcConfigs := *s.instance.Spec.Shoot.Kubernetes.KubeAPIServer.AdditionalOidcConfig
	var errResourceCreation error
	for id, additionalOidcConfig := range additionalOidcConfigs {
		openIDConnectResource := createOpenIDConnectResource(additionalOidcConfig, id)
		errResourceCreation = shootAdminClient.Create(ctx, openIDConnectResource)
	}
	return errResourceCreation
}

func deleteExistingKymaOpenIDConnectResources(ctx context.Context, client k8s_client.Client) (err error) {
	err = client.DeleteAllOf(ctx, &authenticationv1alpha1.OpenIDConnect{}, k8s_client.MatchingLabels(map[string]string{
		imv1.LabelKymaManagedBy: "infrastructure-manager",
	}))

	return err
}

func isOidcExtensionEnabled(shoot gardener.Shoot) bool {
	for _, extension := range shoot.Spec.Extensions {
		if extension.Type == extensions.OidcExtensionType {
			if extension.Disabled == nil {
				return true
			}
			return !(*extension.Disabled)
		}
	}
	return false
}

func multiOidcSupported(runtime imv1.Runtime) bool {
	return runtime.Labels["operator.kyma-project.io/created-by-migrator"] != "true" //nolint:all
}

func createOpenIDConnectResource(additionalOidcConfig gardener.OIDCConfig, oidcID int) *authenticationv1alpha1.OpenIDConnect {
	toSupportedSigningAlgs := func(signingAlgs []string) []authenticationv1alpha1.SigningAlgorithm {
		var supportedSigningAlgs []authenticationv1alpha1.SigningAlgorithm
		for _, alg := range signingAlgs {
			supportedSigningAlgs = append(supportedSigningAlgs, authenticationv1alpha1.SigningAlgorithm(alg))
		}
		return supportedSigningAlgs
	}

	cr := &authenticationv1alpha1.OpenIDConnect{
		TypeMeta: metav1.TypeMeta{
			Kind:       "OpenIDConnect",
			APIVersion: "authentication.gardener.cloud/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("kyma-oidc-%v", oidcID),
			Labels: map[string]string{
				imv1.LabelKymaManagedBy: "infrastructure-manager",
			},
		},
		Spec: authenticationv1alpha1.OIDCAuthenticationSpec{
			IssuerURL:            *additionalOidcConfig.IssuerURL,
			ClientID:             *additionalOidcConfig.ClientID,
			UsernameClaim:        additionalOidcConfig.UsernameClaim,
			UsernamePrefix:       additionalOidcConfig.UsernamePrefix,
			GroupsClaim:          additionalOidcConfig.GroupsClaim,
			GroupsPrefix:         additionalOidcConfig.GroupsPrefix,
			RequiredClaims:       additionalOidcConfig.RequiredClaims,
			SupportedSigningAlgs: toSupportedSigningAlgs(additionalOidcConfig.SigningAlgs),
		},
	}

	return cr
}

func updateConditionFailed(rt *imv1.Runtime) {
	rt.UpdateStatePending(
		imv1.ConditionTypeOidcConfigured,
		imv1.ConditionReasonOidcError,
		string(metav1.ConditionFalse),
		"failed to configure OIDC",
	)
}

func createOIDCConfigMap(ctx context.Context, runtime *imv1.Runtime, cfg RCCfg, m *fsm, s *systemState) error {
	cmName := cfg.ConverterConfig.Kubernetes.AuthenticationConfigurationConfigMap

	shootAdminClient, shootClientError := GetShootClient(ctx, m.Client, s.instance)
	if shootClientError != nil {
		return shootClientError
	}

	authenticationConfig := toAuthenticationConfiguration(runtime)

	authConfigBytes, err := yaml.Marshal(authenticationConfig)
	if err != nil {
		return err
	}

	return shootAdminClient.Create(ctx, &v1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      cmName,
			Namespace: s.instance.Namespace,
		},
		Data: map[string]string{
			"config.yaml": string(authConfigBytes),
		},
	})

}

func toAuthenticationConfiguration(runtime *imv1.Runtime) apiserver.AuthenticationConfiguration {

	toJWTAuthenticator := func(oidcConfig gardener.OIDCConfig) apiserver.JWTAuthenticator {
		return apiserver.JWTAuthenticator{
			Issuer: apiserver.Issuer{
				URL:       *oidcConfig.IssuerURL,
				Audiences: []string{*oidcConfig.ClientID},
			},
			ClaimMappings: apiserver.ClaimMappings{
				Username: apiserver.PrefixedClaimOrExpression{
					Claim:  *oidcConfig.UsernameClaim,
					Prefix: oidcConfig.UsernamePrefix,
				},
				Groups: apiserver.PrefixedClaimOrExpression{
					Claim:  *oidcConfig.GroupsClaim,
					Prefix: oidcConfig.GroupsPrefix,
				},
			},
		}
	}

	jwtAuthenticators := make([]apiserver.JWTAuthenticator, 0)
	jwtAuthenticators = append(jwtAuthenticators, toJWTAuthenticator(runtime.Spec.Shoot.Kubernetes.KubeAPIServer.OidcConfig))

	for _, oidcConfig := range *runtime.Spec.Shoot.Kubernetes.KubeAPIServer.AdditionalOidcConfig {
		jwtAuthenticators = append(jwtAuthenticators, toJWTAuthenticator(oidcConfig))
	}

	return apiserver.AuthenticationConfiguration{
		TypeMeta: metav1.TypeMeta{
			Kind:       "AuthenticationConfiguration",
			APIVersion: "apiserver.config.k8s.io/v1beta1",
		},
		JWT: jwtAuthenticators,
	}
}
