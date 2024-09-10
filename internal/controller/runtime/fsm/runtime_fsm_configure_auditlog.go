package fsm

import (
	"context"

	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/internal/auditlogging"
	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
)

func sFnConfigureAuditLog(ctx context.Context, m *fsm, s *systemState) (stateFn, *ctrl.Result, error) {
	m.log.Info("Configure Audit Log state")

	wasAuditLogEnabled, err := m.AuditLogging.Enable(ctx, s.shoot)

	if wasAuditLogEnabled {
		m.log.Info("Audit Log configured for shoot: " + s.shoot.Name)
		s.instance.UpdateStatePending(
			imv1.ConditionTypeAuditLogConfigured,
			imv1.ConditionReasonAuditLogConfigured,
			"Unknown",
			"Waiting for Gardener shoot to be Ready state after configuration of the Audit Logs",
		)

		return updateStatusAndRequeueAfter(gardenerRequeueDuration)
	}

	if err != nil { //nolint:nestif
		errorMessage := err.Error()
		if errors.Is(err, auditlogging.ErrMissingMapping) {
			if m.RCCfg.AuditLogMandatory {
				m.log.Error(err, "Failed to configure Audit Log, missing region mapping for this shoot", "AuditLogMandatory", m.RCCfg.AuditLogMandatory, "providerType", s.shoot.Spec.Provider.Type, "region", s.shoot.Spec.Region)
				s.instance.UpdateStatePending(
					imv1.ConditionTypeAuditLogConfigured,
					imv1.ConditionReasonAuditLogMissingRegionMapping,
					"False",
					errorMessage,
				)
			} else {
				m.log.Info(errorMessage, "Audit Log was not configured, missing region mapping for this shoot.", "AuditLogMandatory", m.RCCfg.AuditLogMandatory, "providerType", s.shoot.Spec.Provider.Type, "region", s.shoot.Spec.Region)
				s.instance.UpdateStateReady(
					imv1.ConditionTypeAuditLogConfigured,
					imv1.ConditionReasonAuditLogMissingRegionMapping,
					"Missing region mapping for this shoot. Audit Log is not mandatory. Skipping configuration")
			}
		} else {
			if m.RCCfg.AuditLogMandatory {
				m.log.Error(err, "Failed to configure Audit Log", "AuditLogMandatory", m.RCCfg.AuditLogMandatory)
				s.instance.UpdateStatePending(
					imv1.ConditionTypeAuditLogConfigured,
					imv1.ConditionReasonAuditLogError,
					"False",
					errorMessage)
			} else {
				m.log.Info(errorMessage, "AuditLogMandatory", m.RCCfg.AuditLogMandatory)
				s.instance.UpdateStateReady(
					imv1.ConditionTypeAuditLogConfigured,
					imv1.ConditionReasonAuditLogError,
					"Configuration of Audit Log is not mandatory, error for context: "+errorMessage)
			}
		}
	} else {
		s.instance.UpdateStateReady(
			imv1.ConditionTypeAuditLogConfigured,
			imv1.ConditionReasonAuditLogConfigured,
			"Audit Log state completed successfully",
		)
	}

	return updateStatusAndStop()
}
