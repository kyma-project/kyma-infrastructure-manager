package fsm

import (
	"context"

	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/pkg/gardener/shoot"
	ctrl "sigs.k8s.io/controller-runtime"
)

func sFnCreateShootDryRun(_ context.Context, m *fsm, s *systemState) (stateFn, *ctrl.Result, error) {
	m.log.Info("Create shoot [dry-run]")

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

	newShoot, err := convertCreate(&s.instance, shoot.CreateOpts{
		ConverterConfig: m.ConverterConfig,
		AuditLogData:    data,
	})
	if err != nil {
		m.log.Error(err, "Failed to convert Runtime instance to shoot object [dry-run]")
		m.Metrics.IncRuntimeFSMStopCounter()
		return updateStatePendingWithErrorAndStop(
			&s.instance,
			imv1.ConditionTypeRuntimeProvisionedDryRun,
			imv1.ConditionReasonConversionError,
			"Runtime conversion error")
	}

	s.shoot = &newShoot
	s.instance.UpdateStateReady(
		imv1.ConditionTypeRuntimeProvisionedDryRun,
		imv1.ConditionReasonConfigurationCompleted,
		"Runtime processing completed successfully [dry-run]")

	// stop machine if persistence not enabled
	if m.PVCPath != "" {
		return switchState(sFnDumpShootSpec)
	}

	return updateStatusAndStop()
}
