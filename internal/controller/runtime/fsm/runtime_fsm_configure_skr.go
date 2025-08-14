package fsm

import (
	"context"
	"fmt"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/extender/extensions"
	v1 "k8s.io/api/core/v1"
	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/utils/ptr"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	authenticationv1alpha1 "github.com/gardener/oidc-webhook-authenticator/apis/authentication/v1alpha1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/internal/log_level"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/skrdetails"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	k8s_client "sigs.k8s.io/controller-runtime/pkg/client"
)

type additionalOIDCState struct {
	hasEmptyArray bool
}

const (
	msgFailedProvisioningInfoConfigMap = "Failed to apply kyma-provisioning-info config map, scheduling for retry - %s"
	oidcErrorMessage                   = "Failed to create OpenIDConnect resource. Scheduling for retry"
	kymaNamespaceCreationErrorMessage  = "Failed to create kyma-system namespace. Scheduling for retry"
)

func sFnConfigureSKR(ctx context.Context, m *fsm, s *systemState) (stateFn, *ctrl.Result, error) {
	kymaNsCreationErr := createKymaSystemNamespace(ctx, m, s)
	if kymaNsCreationErr != nil {
		return handleProvisioningInfoError(m, s, kymaNsCreationErr, imv1.ConditionReasonKymaSystemNSError)
	}

	skrDetailsErr := applyKymaProvisioningInfoCM(ctx, m, s)
	if skrDetailsErr != nil {
		return handleProvisioningInfoError(m, s, skrDetailsErr, imv1.ConditionReasonOidcAndCMsConfigured)
	}
	m.log.V(log_level.DEBUG).Info("kyma-provisioning-info config map is created/updated")

	if !isOidcExtensionEnabled(*s.shoot) {
		m.log.V(log_level.DEBUG).Info("OIDC extension is disabled")
		s.instance.UpdateStatePending(
			imv1.ConditionTypeOidcAndCMsConfigured,
			imv1.ConditionReasonOidcAndCMsConfigured,
			"True",
			"OIDC extension disabled",
		)

		return switchState(sFnApplyClusterRoleBindings)
	}

	additionalOIDCStatus := additionalOidcEmptyOrUndefined(&s.instance, m.RCCfg)
	err := recreateOpenIDConnectResources(ctx, m, s, additionalOIDCStatus)

	if err != nil {
		updateConditionFailed(&s.instance, imv1.ConditionReasonOidcError, oidcErrorMessage)
		m.log.Error(err, oidcErrorMessage)
		return updateStatusAndRequeueAfter(m.ControlPlaneRequeueDuration)
	}
	m.log.V(log_level.DEBUG).Info("OIDC has been configured", "name", s.shoot.Name)

	s.instance.UpdateStatePending(
		imv1.ConditionTypeOidcAndCMsConfigured,
		imv1.ConditionReasonOidcAndCMsConfigured,
		"True",
		"OIDC and kyma-provisioning-info configuration completed",
	)

	return switchState(sFnApplyClusterRoleBindings)
}

func handleProvisioningInfoError(m *fsm, s *systemState, error error, conditionReason imv1.RuntimeConditionReason) (stateFn, *ctrl.Result, error) {
	finalErrorMsg := fmt.Sprintf(msgFailedProvisioningInfoConfigMap, error.Error())
	m.log.Error(error, finalErrorMsg)
	updateConditionFailed(&s.instance, conditionReason, kymaNamespaceCreationErrorMessage)
	return updateStatusAndRequeueAfter(m.ControlPlaneRequeueDuration)
}

func createKymaSystemNamespace(ctx context.Context, m *fsm, s *systemState) error {
	kymaSystemNs := v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kyma-system",
			Namespace: "",
		},
	}

	runtimeClient, runtimeClientError := m.RuntimeClientGetter.Get(ctx, s.instance)

	if runtimeClientError != nil {
		return runtimeClientError
	}
	kymaNsCreationErr := runtimeClient.Create(ctx, &kymaSystemNs)

	if kymaNsCreationErr != nil {
		if k8s_errors.IsAlreadyExists(kymaNsCreationErr) {
			// we're expecting the namespace to already exist after first reconciliation, so we can ignore this error
			return nil
		}
	}
	return kymaNsCreationErr
}

func additionalOidcEmptyOrUndefined(runtime *imv1.Runtime, cfg RCCfg) additionalOIDCState {
	additionalOidcConfig := runtime.Spec.Shoot.Kubernetes.KubeAPIServer.AdditionalOidcConfig
	additionalOIDCConfigUndefined := func() bool {
		if additionalOidcConfig == nil {
			return true
		}
		for _, oidcConfig := range *additionalOidcConfig {
			if oidcConfig.ClientID == nil || oidcConfig.IssuerURL == nil {
				return true
			}
		}
		return false
	}

	additionalOIDCConfigEmpty := func() bool {
		return len(*additionalOidcConfig) == 0
	}

	if additionalOIDCConfigUndefined() {
		additionalOidcConfig = &[]imv1.OIDCConfig{}
		defaultOIDCConfig := cfg.ClusterConfig.DefaultSharedIASTenant.ToOIDCConfig()
		*additionalOidcConfig = append(*additionalOidcConfig, imv1.OIDCConfig{OIDCConfig: defaultOIDCConfig})
		runtime.Spec.Shoot.Kubernetes.KubeAPIServer.AdditionalOidcConfig = additionalOidcConfig
		return additionalOIDCState{}
	}

	if additionalOIDCConfigEmpty() {
		return additionalOIDCState{hasEmptyArray: true}
	}

	return additionalOIDCState{}

}

func recreateOpenIDConnectResources(ctx context.Context, m *fsm, s *systemState, additionalOIDC additionalOIDCState) error {
	runtimeClient, runtimeClientError := m.RuntimeClientGetter.Get(ctx, s.instance)
	if runtimeClientError != nil {
		return runtimeClientError
	}

	err := deleteExistingKymaOpenIDConnectResources(ctx, runtimeClient)
	if err != nil {
		return err
	}

	if additionalOIDC.hasEmptyArray {
		return nil
	}

	additionalOidcConfigs := *s.instance.Spec.Shoot.Kubernetes.KubeAPIServer.AdditionalOidcConfig
	var errResourceCreation error
	for id, additionalOidcConfig := range additionalOidcConfigs {
		openIDConnectResource := createOpenIDConnectResource(additionalOidcConfig, id)
		errResourceCreation = runtimeClient.Create(ctx, openIDConnectResource)
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

func createOpenIDConnectResource(additionalOidcConfig imv1.OIDCConfig, oidcID int) *authenticationv1alpha1.OpenIDConnect {
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
			JWKS: authenticationv1alpha1.JWKSSpec{
				Keys: additionalOidcConfig.JWKS,
			},
		},
	}

	return cr
}

func updateConditionFailed(rt *imv1.Runtime, reason imv1.RuntimeConditionReason, message string) {
	rt.UpdateStatePending(
		imv1.ConditionTypeOidcAndCMsConfigured,
		reason,
		string(metav1.ConditionFalse),
		message,
	)
}

func applyKymaProvisioningInfoCM(ctx context.Context, m *fsm, s *systemState) error {
	configMap, conversionErr := skrdetails.ToKymaProvisioningInfoConfigMap(s.instance, s.shoot)
	if conversionErr != nil {
		return errors.Wrap(conversionErr, "failed to convert RuntimeCR and Shoot spec to ToKymaProvisioningInfo config map")
	}

	runtimeClient, runtimeClientError := m.RuntimeClientGetter.Get(ctx, s.instance)
	if runtimeClientError != nil {
		return runtimeClientError
	}

	errResourceCreation := runtimeClient.Patch(ctx, &configMap, k8s_client.Apply, &k8s_client.PatchOptions{
		FieldManager: fieldManagerName,
		Force:        ptr.To(true),
	})

	return errResourceCreation
}
