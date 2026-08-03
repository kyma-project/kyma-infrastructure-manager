package fsm

import (
	"context"
	"fmt"

	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/internal/log_level"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const auditLogReadCredentialsSecretName = "auditlog-read-credentials"

// sFnCopyAuditLogReadCredentials copies the audit log read credentials from KCP to SKR
// This state executes after sFnMigrateToDedicatedAuditLog successfully configures the shoot
// with dedicated audit logging. It copies the read credentials secret to the runtime cluster
// to enable users to access their audit logs.
func sFnCopyAuditLogReadCredentials(ctx context.Context, m *fsm, s *systemState) (stateFn, *ctrl.Result, error) {
	m.log.V(log_level.DEBUG).Info("Copying audit log read credentials to SKR")

	runtimeID := s.instance.Labels[imv1.LabelKymaRuntimeID]

	auditLogData, err := m.AuditLogDataProvider.GetDedicatedAuditLogData(ctx, runtimeID, false)
	if err != nil {
		m.log.Error(err, "Failed to get audit log data for credentials copy", "runtimeID", runtimeID)
		s.instance.UpdateStatePending(
			imv1.ConditionTypeAuditLogCredentialsCopied,
			imv1.ConditionReasonCredentialsCopyError,
			metav1.ConditionFalse,
			fmt.Sprintf("Failed to get audit log data: %v", err),
		)
		return updateStatusAndRequeueAfter(m.ControlPlaneRequeueDuration)
	}

	// Requeue if no read credentials configured - KALM may not have populated the secret yet
	if auditLogData.ReadCredsSecretName == "" {
		m.log.Info("No read credentials secret configured, waiting for KALM to populate", "runtimeID", runtimeID)
		s.instance.UpdateStatePending(
			imv1.ConditionTypeAuditLogCredentialsCopied,
			imv1.ConditionReasonCredentialsCopyError,
			metav1.ConditionFalse,
			"Waiting for read credentials secret to be configured",
		)
		return updateStatusAndRequeueAfter(m.ControlPlaneRequeueDuration)
	}

	if err := copyReadCredentialsToSKR(ctx, m, s, auditLogData.ReadCredsSecretName); err != nil {
		m.log.Error(err, "Failed to copy read credentials to SKR", "runtimeID", runtimeID)
		s.instance.UpdateStatePending(
			imv1.ConditionTypeAuditLogCredentialsCopied,
			imv1.ConditionReasonCredentialsCopyError,
			metav1.ConditionFalse,
			fmt.Sprintf("Failed to copy read credentials: %v", err),
		)
		return updateStatusAndRequeueAfter(m.ControlPlaneRequeueDuration)
	}

	m.log.Info("Successfully copied read credentials to SKR",
		"runtimeID", runtimeID,
		"sourceSecret", auditLogData.ReadCredsSecretName)

	s.instance.UpdateStateReady(
		imv1.ConditionTypeAuditLogCredentialsCopied,
		imv1.ConditionReasonCredentialsCopied,
		"Audit log read credentials copied to runtime",
	)

	completeProvisioning(&s.instance)
	return updateStatusAndStop()
}

func copyReadCredentialsToSKR(ctx context.Context, m *fsm, s *systemState, sourceSecretName string) error {
	runtimeClient, err := m.RuntimeClientGetter.Get(ctx, s.instance)
	if err != nil {
		return fmt.Errorf("failed to get runtime client: %w", err)
	}

	var sourceSecret corev1.Secret
	secretKey := k8stypes.NamespacedName{
		Name:      sourceSecretName,
		Namespace: s.instance.Namespace,
	}
	if err := m.KcpClient.Get(ctx, secretKey, &sourceSecret); err != nil {
		return fmt.Errorf("failed to get source secret %s: %w", secretKey, err)
	}

	targetKey := k8stypes.NamespacedName{
		Name:      auditLogReadCredentialsSecretName,
		Namespace: "kyma-system",
	}
	var existingSecret corev1.Secret
	err = runtimeClient.Get(ctx, targetKey, &existingSecret)

	if err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to check existing secret: %w", err)
	}

	if apierrors.IsNotFound(err) {
		newSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      auditLogReadCredentialsSecretName,
				Namespace: "kyma-system",
				Labels: map[string]string{
					imv1.LabelKymaManagedBy: "infrastructure-manager",
				},
			},
			Data: sourceSecret.Data,
			Type: sourceSecret.Type,
		}
		if err := runtimeClient.Create(ctx, newSecret); err != nil {
			return fmt.Errorf("failed to create secret in SKR: %w", err)
		}
		return nil
	}

	existingSecret.Data = sourceSecret.Data
	existingSecret.Type = sourceSecret.Type
	if existingSecret.Labels == nil {
		existingSecret.Labels = make(map[string]string)
	}
	existingSecret.Labels[imv1.LabelKymaManagedBy] = "infrastructure-manager"

	if err := runtimeClient.Update(ctx, &existingSecret); err != nil {
		return fmt.Errorf("failed to update secret in SKR: %w", err)
	}
	return nil
}

func completeProvisioning(instance *imv1.Runtime) {
	if !instance.IsProvisioningCompletedStatusSet() {
		instance.UpdateStateProvisioningCompleted()
	}
}
