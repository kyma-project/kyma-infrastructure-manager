package fsm

import (
	"context"
	"fmt"

	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	gardener_shoot "github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot"
	ctrl "sigs.k8s.io/controller-runtime"
)

const msgFailedToConfigureAuditlogs = "Failed to configure audit logs"

func sFnCreateShoot(ctx context.Context, m *fsm, s *systemState) (stateFn, *ctrl.Result, error) {
	m.log.Info("Create shoot state")

	if s.instance.Spec.Shoot.EnforceSeedLocation != nil && *s.instance.Spec.Shoot.EnforceSeedLocation {
		seedAvailable, regionsWithSeeds, err := seedForRegionAvailable(ctx, m.ShootClient, s.instance.Spec.Shoot.Region)
		if err != nil {
			msg := fmt.Sprintf("Failed to verify whether seed for region %s exists.", s.instance.Spec.Shoot.Region)
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
			msg := fmt.Sprintf("Seed for region %s doesn't exist. The followig regions have seeds assigned: %v.", s.instance.Spec.Shoot.Region, regionsWithSeeds)
			m.log.Error(nil, msg)
			m.Metrics.IncRuntimeFSMStopCounter()
			return updateStatePendingWithErrorAndStop(
				&s.instance,
				imv1.ConditionTypeRuntimeProvisioned,
				imv1.ConditionReasonSeedNotFound,
				msg)
		}
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
		ConverterConfig: m.ConverterConfig,
		AuditLogData:    data,
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

	m.log.Info(
		"Gardener shoot for runtime initialised successfully",
		"Name", shoot.Name,
		"Namespace", shoot.Namespace,
	)

	s.instance.UpdateStatePending(
		imv1.ConditionTypeRuntimeProvisioned,
		imv1.ConditionReasonShootCreationPending,
		"Unknown",
		"Shoot is pending",
	)

	// it will be executed only once because created shoot is executed only once
	shouldDumpShootSpec := m.PVCPath != ""
	if shouldDumpShootSpec {
		s.shoot = shoot.DeepCopy()
		return switchState(sFnDumpShootSpec)
	}

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
