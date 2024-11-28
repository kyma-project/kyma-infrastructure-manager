package fsm

import (
	"context"
	"fmt"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	authenticationv1alpha1 "github.com/gardener/oidc-webhook-authenticator/apis/authentication/v1alpha1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/pkg/config"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/extender"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	k8s_client "sigs.k8s.io/controller-runtime/pkg/client"
)

func sFnConfigureOidc(ctx context.Context, m *fsm, s *systemState) (stateFn, *ctrl.Result, error) {
	m.log.Info("Configure OIDC state")

	if !isOidcExtensionEnabled(*s.shoot) {
		m.log.Info("OIDC extension is disabled")
		(&s.instance).UpdateStatePending(
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
		(&s.instance).UpdateStatePending(
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
		m.log.Error(err, "Failed to create OpenIDConnect resource")
		updateConditionFailed(&s.instance)
		return updateStatusAndStopWithError(err)
	}

	m.log.Info("OIDC has been configured", "Name", s.shoot.Name)
	(&s.instance).UpdateStatePending(
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
		if extension.Type == extender.OidcExtensionType {
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
