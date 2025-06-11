package fsm

import (
	"context"
	"fmt"
	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/internal/log_level"
	gardener_shoot "github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot/extender"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/skrdetails"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/structuredauth"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	msgFailedToConfigureAuditlogs      = "Failed to configure audit logs"
	msgFailedStructuredConfigMap       = "Failed to create structured authentication config map"
	msgFailedProvisioningInfoConfigMap = "Failed to create kyma-provisioning-info config map"
	msgFailedToConfigureRegistryCache  = "Failed to configure registry cache"
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

	if m.StructuredAuthEnabled {
		cmName := fmt.Sprintf(extender.StructuredAuthConfigFmt, s.instance.Spec.Shoot.Name)
		oidcConfig := structuredauth.GetOIDCConfigOrDefault(s.instance, m.ConverterConfig.Kubernetes.DefaultOperatorOidc.ToOIDCConfig())

		err := structuredauth.CreateOrUpdateStructuredAuthConfigMap(ctx, m.ShootClient, types.NamespacedName{Name: cmName, Namespace: m.ShootNamesapace}, oidcConfig)
		if err != nil {
			m.log.Error(err, "Failed to create structured authentication config map")

			m.Metrics.IncRuntimeFSMStopCounter()
			return updateStatePendingWithErrorAndStop(
				&s.instance,
				imv1.ConditionTypeRuntimeProvisioned,
				imv1.ConditionReasonOidcError,
				msgFailedStructuredConfigMap)
		}
	}

	skrDetailsErr:= createKymaProvisioningInfoCM(ctx, m, s)
	if skrDetailsErr != nil {
		return handleConfigMapCreationError(m, skrDetailsErr, s)
	}
	m.log.V(log_level.DEBUG).Info("kyma-provisioning-info config map is created")

	data, err := m.AuditLogging.GetAuditLogData(
		s.instance.Spec.Shoot.Provider.Type,
		s.instance.Spec.Shoot.Region)

	if err != nil {
		m.log.Error(err, msgFailedToConfigureAuditlogs)
	}

	if err != nil && m.AuditLogMandatory {
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
		StructuredAuthEnabled: m.StructuredAuthEnabled,
	})
	if err != nil {
		m.log.Error(err, "Failed to convert Runtime instance to shoot object")
		m.Metrics.IncRuntimeFSMStopCounter()
		return updateStatePendingWithErrorAndStop(
			&s.instance,
			imv1.ConditionTypeRuntimeProvisioned,
			imv1.ConditionReasonConversionError,
			fmt.Sprintf("Runtime conversion error %v", err))
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

func handleConfigMapCreationError(m *fsm, skrDetailsErr error, s *systemState) (stateFn, *ctrl.Result, error) {
	m.log.Error(skrDetailsErr, "Failed to create SKR details config map")
	m.Metrics.IncRuntimeFSMStopCounter()
	return updateStatePendingWithErrorAndStop(
		&s.instance,
		imv1.ConditionTypeRuntimeProvisioned,
		imv1.ConditionReasonConfigurationErr,
		msgFailedProvisioningInfoConfigMap)
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

func createKymaProvisioningInfoCM(ctx context.Context, m *fsm, s *systemState) error {
	configMap, skrDetailsErr := skrdetails.ToKymaProvisioningInfoConfigMap(s.instance, s.shoot)

	if skrDetailsErr != nil {
		return skrDetailsErr
	}

	shootAdminClient, shootClientError := GetShootClient(ctx, m.Client, s.instance)
	if shootClientError != nil {
		return shootClientError
	}

	errResourceCreation := shootAdminClient.Create(ctx, &configMap)

	return errResourceCreation
}
