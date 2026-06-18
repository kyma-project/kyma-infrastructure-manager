package fsm

import (
	"context"
	"fmt"

	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/internal/log_level"
	ctrl "sigs.k8s.io/controller-runtime"
)

// sFnMigrateToDedicatedAuditLog migrates the shoot from shared to dedicated audit logging
// This state is executed as the FINAL step after successful runtime provisioning to claim an AuditLogCR
// from the pool and patch the shoot with dedicated audit logging configuration.
// This ensures dedicated resources are only claimed after the entire provisioning succeeds.
func sFnMigrateToDedicatedAuditLog(ctx context.Context, m *fsm, s *systemState) (stateFn, *ctrl.Result, error) {
	m.log.V(log_level.DEBUG).Info("Migrating to dedicated audit logging state (final step)")

	// Check if global feature flag is enabled
	if !m.DedicatedAuditLoggingEnabled {
		m.log.V(log_level.DEBUG).Info("Dedicated audit logging feature disabled globally, completing provisioning")

		if !s.instance.IsProvisioningCompletedStatusSet() {
			s.instance.UpdateStateProvisioningCompleted()
		}

		return updateStatusAndStop()
	}

	// Check if runtime-specific audit log access is enabled
	if s.instance.Spec.AuditLogAccessEnabled == nil || !*s.instance.Spec.AuditLogAccessEnabled {
		m.log.V(log_level.DEBUG).Info("Audit log access not enabled for this runtime, completing provisioning",
			"runtimeID", s.instance.GetName())

		if !s.instance.IsProvisioningCompletedStatusSet() {
			s.instance.UpdateStateProvisioningCompleted()
		}

		return updateStatusAndStop()
	}

	// Step 1: Get desired audit log data from reservation (read-only, no side effects)
	auditLogData, err := m.AuditLogDataProvider.GetReservedAuditLogData(
		ctx,
		s.instance.GetName(),
	)
	if err != nil {
		// No reservation found - fail provisioning
		msg := fmt.Sprintf("Failed to get reserved audit log configuration: %v", err)
		m.log.Error(err, "Cannot retrieve reserved audit log configuration")

		s.instance.UpdateStateFailed(
			imv1.ConditionTypeRuntimeProvisioned,
			imv1.ConditionReasonAuditLogError,
			msg,
		)

		m.Metrics.IncRuntimeFSMStopCounter()
		return updateStatusAndStop()
	}

	m.log.Info("Successfully retrieved reserved audit log configuration",
		"runtimeID", s.instance.GetName(),
		"tenantID", auditLogData.TenantID)

	// Step 2: Get current shoot audit log config (read-only, no side effects)
	shootAuditLogData, err := getShootAuditLogConfig(s.shoot)
	if err != nil {
		m.log.Error(err, "Failed to get current shoot audit log configuration, will attempt to patch")
		// If we can't get current config, assume we need to patch
		shootAuditLogData = nil
	}

	// Step 3: Compare configurations
	configsEqual := shootAuditLogData != nil && auditLogConfigsEqual(shootAuditLogData, auditLogData)

	// Step 4: Claim resource (upgrade from light lock to heavy lock)
	// We ALWAYS claim, even if configs are equal, to ensure the reservation is confirmed
	m.log.Info("Confirming reservation for runtime", "runtimeID", s.instance.GetName())

	err = m.AuditLogDataProvider.ConfirmReservation(ctx, s.instance.GetName())
	if err != nil {
		// Claim failed - fail provisioning
		msg := fmt.Sprintf("Failed to confirm audit log reservation: %v", err)
		m.log.Error(err, "Cannot complete runtime provisioning - reservation confirmation failed")

		s.instance.UpdateStateFailed(
			imv1.ConditionTypeRuntimeProvisioned,
			imv1.ConditionReasonAuditLogError,
			msg,
		)

		m.Metrics.IncRuntimeFSMStopCounter()
		return updateStatusAndStop()
	}

	m.log.Info("Successfully confirmed reservation (upgraded to claim)", "runtimeID", s.instance.GetName())

	// Step 5: Patch only if configurations differ
	if configsEqual {
		m.log.Info("Shoot already configured with correct dedicated audit logging, skipping patch",
			"runtimeID", s.instance.GetName())

		// Complete provisioning
		if !s.instance.IsProvisioningCompletedStatusSet() {
			s.instance.UpdateStateProvisioningCompleted()
		}

		return updateStatusAndStop()
	}

	m.log.Info("Shoot audit log configuration differs, patching shoot",
		"runtimeID", s.instance.GetName())

	// Step 6: PATCH shoot with dedicated config
	if err := patchShootAuditLog(ctx, m, s, auditLogData); err != nil {
		// AuditLogCR is claimed, we'll retry the patch on next reconciliation
		m.log.Error(err, "Failed to patch shoot with dedicated audit log, will retry",
			"runtimeID", s.instance.GetName())

		s.instance.UpdateStatePending(
			imv1.ConditionTypeRuntimeProvisioned,
			imv1.ConditionReasonProcessing,
			"True",
			fmt.Sprintf("Migrating to dedicated audit logging: %v", err),
		)

		return updateStatusAndRequeueAfter(m.GardenerRequeueDuration)
	}

	m.log.Info("Successfully patched shoot with dedicated audit logging",
		"runtimeID", s.instance.GetName(),
		"tenantID", auditLogData.TenantID)

	// Update provisioning completed status
	if !s.instance.IsProvisioningCompletedStatusSet() {
		s.instance.UpdateStateProvisioningCompleted()
	}

	// Complete without requeue - Gardener shoot reconciliation will trigger if needed
	return updateStatusAndStop()
}
