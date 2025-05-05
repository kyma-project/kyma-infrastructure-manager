package fsm

import (
	"context"
	"fmt"

	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/extender/extensions"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	authenticationv1alpha1 "github.com/gardener/oidc-webhook-authenticator/apis/authentication/v1alpha1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/internal/log_level"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	k8s_client "sigs.k8s.io/controller-runtime/pkg/client"
)

func sFnConfigureOidc(ctx context.Context, m *fsm, s *systemState) (stateFn, *ctrl.Result, error) {
	if !isOidcExtensionEnabled(*s.shoot) {
		m.log.V(log_level.DEBUG).Info("OIDC extension is disabled")
		s.instance.UpdateStatePending(
			imv1.ConditionTypeOidcConfigured,
			imv1.ConditionReasonOidcConfigured,
			"True",
			"OIDC extension disabled",
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

	m.log.V(log_level.DEBUG).Info("OIDC has been configured", "name", s.shoot.Name)
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

	additionalOIDCConfigEmpty := func() bool {
		if additionalOidcConfig == nil {
			return true
		}

		for _, oidcConfig := range *additionalOidcConfig {
			if oidcConfig.ClientID != nil && oidcConfig.IssuerURL != nil {
				return false
			}
		}

		return true
	}

	if additionalOIDCConfigEmpty() {
		additionalOidcConfig = &[]gardener.OIDCConfig{}
		defaultOIDCConfig := cfg.ClusterConfig.DefaultSharedIASTenant.ToOIDCConfig()
		*additionalOidcConfig = append(*additionalOidcConfig, defaultOIDCConfig)
		runtime.Spec.Shoot.Kubernetes.KubeAPIServer.AdditionalOidcConfig = additionalOidcConfig
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
