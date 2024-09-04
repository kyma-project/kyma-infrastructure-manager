package fsm

import (
	"context"

	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
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

	if err != nil {
		m.log.Error(err, "Failed to configure Audit Log")
		s.instance.UpdateStatePending(
			imv1.ConditionTypeAuditLogConfigured,
			imv1.ConditionReasonAuditLogError,
			"False",
			err.Error(),
		)
		return updateStatusAndRequeueAfter(gardenerRequeueDuration)
	}

	s.instance.UpdateStateReady(
		imv1.ConditionTypeAuditLogConfigured,
		imv1.ConditionReasonAuditLogConfigured,
		"Audit Log configured successfully",
	)

	return updateStatusAndStop()
}
