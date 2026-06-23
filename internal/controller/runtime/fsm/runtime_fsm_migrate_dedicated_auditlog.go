package fsm

import (
	"context"
	"fmt"

	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/internal/log_level"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

// sFnMigrateToDedicatedAuditLog migrates the shoot from shared to dedicated audit logging
// This state is executed as the FINAL step after successful runtime provisioning to claim an AuditLogCR
// from the pool and patch the shoot with dedicated audit logging configuration.
// This ensures dedicated resources are only claimed after the entire provisioning succeeds.
func sFnMigrateToDedicatedAuditLog(ctx context.Context, m *fsm, s *systemState) (stateFn, *ctrl.Result, error) {
	m.log.V(log_level.DEBUG).Info("Migrating to dedicated audit logging state (final step)")
	runtimeID := s.instance.Labels[imv1.LabelKymaRuntimeID]

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
			"runtimeID", runtimeID)

		if !s.instance.IsProvisioningCompletedStatusSet() {
			s.instance.UpdateStateProvisioningCompleted()
		}

		return updateStatusAndStop()
	}

	// Step 1: Get desired audit log data and claim the resource
	// This performs Phase 2 of the two-phase claim (upgrade from light lock to heavy lock)
	auditLogData, err := m.AuditLogDataProvider.GetDedicatedAuditLogData(
		ctx,
		runtimeID,
		true, // claim=true to upgrade reservation to full claim
	)
	if err != nil {
		msg := fmt.Sprintf("Failed to get and claim dedicated audit log configuration: %v", err)
		m.log.Error(err, "Cannot complete runtime provisioning - failed to claim reserved audit log")

		s.instance.UpdateStateFailed(
			imv1.ConditionTypeCustomAuditLogConfigured,
			imv1.ConditionReasonCustomAuditLogError,
			msg,
		)

		m.Metrics.IncRuntimeFSMStopCounter()
		return updateStatusAndStop()
	}

	m.log.Info("Successfully claimed dedicated audit log configuration",
		"runtimeID", runtimeID,
		"tenantID", auditLogData.TenantID)

	// Step 2: Get current shoot audit log config (read-only)
	shootAuditLogData, err := getShootAuditLogConfig(s.shoot)
	if err != nil {
		m.log.Error(err, "Failed to get current shoot audit log configuration, will attempt to patch")
		// If we can't get current config, assume we need to patch
		shootAuditLogData = nil
	}

	// Step 3: Compare configurations
	configsEqual := shootAuditLogData != nil && auditLogConfigsEqual(shootAuditLogData, auditLogData)

	// Step 4: Patch only if configurations differ
	if configsEqual {
		m.log.Info("Shoot already configured with correct dedicated audit logging, exiting",
			"runtimeID", runtimeID)

		s.instance.UpdateStateReady(
			imv1.ConditionTypeCustomAuditLogConfigured,
			imv1.ConditionReasonCustomAuditLogConfigured,
			"Custom AuditLog shoot configuration completed",
		)

		// Complete provisioning
		if !s.instance.IsProvisioningCompletedStatusSet() {
			s.instance.UpdateStateProvisioningCompleted()
		}

		return updateStatusAndStop()
	}

	m.log.Info("Shoot audit log configuration differs, patching shoot",
		"runtimeID", runtimeID)

	// Step 5: PATCH shoot with dedicated config
	if err := patchShootAuditLog(ctx, m, s, auditLogData); err != nil {
		// AuditLogCR is claimed, we'll retry the patch on next reconciliation
		m.log.Error(err, "Failed to patch shoot with dedicated audit log, will retry",
			"runtimeID", runtimeID)

		s.instance.UpdateStatePending(
			imv1.ConditionTypeCustomAuditLogConfigured,
			imv1.ConditionReasonCustomAuditLogConfigured,
			metav1.ConditionFalse,
			"Custom AuditLog shoot configuration could not be patched, will retry",
		)

		return updateStatusAndRequeueAfter(m.GardenerRequeueDuration)
	}

	m.log.Info("Successfully patched shoot with dedicated audit logging",
		"runtimeID", s.instance.GetName(),
		"tenantID", auditLogData.TenantID)

	s.instance.UpdateStatePending(
		imv1.ConditionTypeCustomAuditLogConfigured,
		imv1.ConditionReasonCustomAuditLogConfigured,
		metav1.ConditionUnknown,
		"Custom AuditLog shoot configuration completed",
	)
	return updateStatusAndRequeueAfter(m.GardenerRequeueDuration)
}
