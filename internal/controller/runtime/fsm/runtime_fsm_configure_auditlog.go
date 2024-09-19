package fsm

import (
	"context"
	"strconv"

	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/internal/auditlogging"
	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
)

func sFnConfigureAuditLog(ctx context.Context, m *fsm, s *systemState) (stateFn, *ctrl.Result, error) {
	m.log.Info("Configure Audit Log state")

	shootNeedsToBeReconciled, err := m.AuditLogging.Enable(ctx, s.shoot)

	if shootNeedsToBeReconciled && err == nil {
		m.log.Info("Audit Log configured for shoot: " + s.shoot.Name)
		s.instance.UpdateStatePending(
			imv1.ConditionTypeAuditLogConfigured,
			imv1.ConditionReasonAuditLogConfigured,
			"Unknown",
			"Waiting for Gardener shoot to be Ready state after configuration of the Audit Logs",
		)

		return updateStatusAndRequeueAfter(gardenerRequeueDuration)
	}

	if err == nil {
		s.instance.UpdateStateReady(
			imv1.ConditionTypeAuditLogConfigured,
			imv1.ConditionReasonAuditLogConfigured,
			"Audit Log state completed successfully",
		)

		return updateStatusAndStop()
	}

	return handleError(err, m, s)
}

func handleError(err error, m *fsm, s *systemState) (stateFn, *ctrl.Result, error) {
	setStateForAuditLogError := func(reason imv1.RuntimeConditionReason, pendingMsg string, readyMsg string) {
		if m.RCCfg.AuditLogMandatory {
			s.instance.UpdateStatePending(
				imv1.ConditionTypeAuditLogConfigured,
				reason,
				"False",
				pendingMsg,
			)
		} else {
			s.instance.UpdateStateReady(
				imv1.ConditionTypeAuditLogConfigured,
				reason,
				readyMsg)
		}
	}

	logError := func(err error, keysAndValues ...any) {
		if m.RCCfg.AuditLogMandatory {
			m.log.Error(nil, err.Error(), keysAndValues...)
		} else {
			m.log.Info(err.Error(), keysAndValues...)
		}
	}

	if k8serrors.IsConflict(err) {
		m.log.Error(err, "Conflict while updating Shoot object after applying Audit Log configuration, retrying")
		s.instance.UpdateStatePending(
			imv1.ConditionTypeAuditLogConfigured,
			imv1.ConditionReasonAuditLogError,
			"True",
			err.Error(),
		)

		return updateStatusAndRequeue()
	}

	auditLogMandatoryString := strconv.FormatBool(m.RCCfg.AuditLogMandatory)

	if errors.Is(err, auditlogging.ErrMissingMapping) {
		pendingStatusMsg := err.Error()
		readyStatusMsg := "Missing region mapping for this shoot. Audit Log is not mandatory. Skipping configuration"
		setStateForAuditLogError(imv1.ConditionReasonAuditLogMissingRegionMapping, pendingStatusMsg, readyStatusMsg)

		logError(err, "AuditLogMandatory", auditLogMandatoryString, "providerType", s.shoot.Spec.Provider.Type, "region", s.shoot.Spec.Region)

		return updateStatusAndStop()
	}

	pendingStatusMsg := err.Error()
	readyStatusMsg := "Configuration of Audit Log is not mandatory, error for context: " + err.Error()
	setStateForAuditLogError(imv1.ConditionReasonAuditLogError, pendingStatusMsg, readyStatusMsg)

	logError(err, "AuditLogMandatory", auditLogMandatoryString)

	return updateStatusAndStop()
}
