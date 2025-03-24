package fsm

import (
	"context"
	"fmt"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/yaml"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/internal/log_level"
	gardener_shoot "github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	msgFailedToConfigureAuditlogs = "Failed to configure audit logs"
	msgFailedStructuredConfigMap  = "Failed to create structured authentication config map"
)

func sFnCreateShoot(ctx context.Context, m *fsm, s *systemState) (stateFn, *ctrl.Result, error) {
	if s.instance.Spec.Shoot.EnforceSeedLocation != nil && *s.instance.Spec.Shoot.EnforceSeedLocation {
		seedAvailable, regionsWithSeeds, err := seedForRegionAvailable(ctx, m.ShootClient, s.instance.Spec.Shoot.Provider.Type, s.instance.Spec.Shoot.Region)
		if err != nil {
			msg := fmt.Sprintf("Failed to verify whether seed is available for the region %s.", s.instance.Spec.Shoot.Region)
			m.log.Error(err, msg)
			s.instance.UpdateStatePending(
				imv1.ConditionTypeRuntimeProvisioned,
				imv1.ConditionReasonGardenerError,
				"False",
				msg,
			)
			return updateStatusAndRequeueAfter(m.GardenerRequeueDuration)
		}

		if !seedAvailable {
			msg := fmt.Sprintf("Cannot find available seed for the region %s. The followig regions have seeds ready: %v.", s.instance.Spec.Shoot.Region, regionsWithSeeds)
			m.log.Error(nil, msg)
			m.Metrics.IncRuntimeFSMStopCounter()
			return updateStatePendingWithErrorAndStop(
				&s.instance,
				imv1.ConditionTypeRuntimeProvisioned,
				imv1.ConditionReasonSeedNotFound,
				msg)
		}
	}

	oidcConfig := s.instance.Spec.Shoot.Kubernetes.KubeAPIServer.OidcConfig

	if oidcConfig.IssuerURL == nil || oidcConfig.ClientID == nil {
		oidcConfig = createDefaultOIDCConfig(m.RCCfg.ClusterConfig.DefaultSharedIASTenant)
	}

	err := createOrUpdateOIDCConfigMap(ctx, oidcConfig, m, s)
	if err != nil {
		m.Metrics.IncRuntimeFSMStopCounter()
		return updateStatePendingWithErrorAndStop(
			&s.instance,
			imv1.ConditionTypeRuntimeProvisioned,
			imv1.ConditionReasonOidcError,
			msgFailedStructuredConfigMap)
	}

	data, err := m.AuditLogging.GetAuditLogData(
		s.instance.Spec.Shoot.Provider.Type,
		s.instance.Spec.Shoot.Region)

	if err != nil {
		m.log.Error(err, msgFailedToConfigureAuditlogs)
	}

	if err != nil && m.RCCfg.AuditLogMandatory {
		m.Metrics.IncRuntimeFSMStopCounter()
		return updateStatePendingWithErrorAndStop(
			&s.instance,
			imv1.ConditionTypeRuntimeProvisioned,
			imv1.ConditionReasonAuditLogError,
			msgFailedToConfigureAuditlogs)
	}

	shoot, err := convertCreate(&s.instance, gardener_shoot.CreateOpts{
		ConverterConfig:       m.ConverterConfig,
		AuditLogData:          data,
		MaintenanceTimeWindow: getMaintenanceTimeWindow(s, m),
	})
	if err != nil {
		m.log.Error(err, "Failed to convert Runtime instance to shoot object")
		m.Metrics.IncRuntimeFSMStopCounter()
		return updateStatePendingWithErrorAndStop(
			&s.instance,
			imv1.ConditionTypeRuntimeProvisioned,
			imv1.ConditionReasonConversionError,
			"Runtime conversion error")
	}

	err = m.ShootClient.Create(ctx, &shoot)
	if err != nil {
		m.log.Error(err, "Failed to create new gardener Shoot")
		s.instance.UpdateStatePending(
			imv1.ConditionTypeRuntimeProvisioned,
			imv1.ConditionReasonGardenerError,
			"False",
			fmt.Sprintf("Gardener API create error: %v", err),
		)
		return updateStatusAndRequeueAfter(m.GardenerRequeueDuration)
	}

	m.log.V(log_level.DEBUG).Info(
		"Gardener shoot for runtime initialised successfully",
		"name", shoot.Name,
		"Namespace", shoot.Namespace,
	)

	s.instance.UpdateStatePending(
		imv1.ConditionTypeRuntimeProvisioned,
		imv1.ConditionReasonShootCreationPending,
		"Unknown",
		"Shoot is pending",
	)

	return updateStatusAndRequeueAfter(m.GardenerRequeueDuration)
}

func convertCreate(instance *imv1.Runtime, opts gardener_shoot.CreateOpts) (gardener.Shoot, error) {
	if err := instance.ValidateRequiredLabels(); err != nil {
		return gardener.Shoot{}, err
	}

	converter := gardener_shoot.NewConverterCreate(opts)
	newShoot, err := converter.ToShoot(*instance)
	if err != nil {
		return newShoot, err
	}

	return newShoot, nil
}

func createOrUpdateOIDCConfigMap(ctx context.Context, oidcConfig gardener.OIDCConfig, m *fsm, s *systemState) error {
	cmName := fmt.Sprintf("structured-auth-config-%s", s.instance.Spec.Shoot.Name)

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
				Name:      cmName,
				Namespace: m.ShootNamesapace,
			},
			Data: map[string]string{
				"config.yaml": string(authConfigBytes),
			},
		}, err
	}

	var existingCM v1.ConfigMap
	err := m.ShootClient.Get(ctx, types.NamespacedName{Name: cmName, Namespace: m.ShootNamesapace}, &existingCM)

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
		return m.ShootClient.Update(ctx, &existingCM)
	}

	return m.ShootClient.Create(ctx, &newConfigMap)
}

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
		return JWTAuthenticator{
			Issuer: Issuer{
				URL:       *oidcConfig.IssuerURL,
				Audiences: []string{*oidcConfig.ClientID},
			},
			ClaimMappings: ClaimMappings{
				Username: PrefixedClaim{
					Claim:  *oidcConfig.UsernameClaim,
					Prefix: oidcConfig.UsernamePrefix,
				},
				Groups: PrefixedClaim{
					Claim:  *oidcConfig.GroupsClaim,
					Prefix: ptr.To("-"),
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
