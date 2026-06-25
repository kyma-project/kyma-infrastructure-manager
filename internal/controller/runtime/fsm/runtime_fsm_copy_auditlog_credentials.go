package fsm

import (
	"context"
	"fmt"

	imv1 "github.com/kyma-project/infrastructure-manager/api/v1"
	"github.com/kyma-project/infrastructure-manager/internal/log_level"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const auditLogReadCredentialsSecretName = "auditlog-read-credentials"

// sFnCopyAuditLogReadCredentials copies the audit log read credentials from KCP to SKR
// This state executes after sFnMigrateToDedicatedAuditLog successfully configures the shoot
// with dedicated audit logging. It copies the read credentials secret to the runtime cluster
// to enable users to access their audit logs.
func sFnCopyAuditLogReadCredentials(ctx context.Context, m *fsm, s *systemState) (stateFn, *ctrl.Result, error) {
	m.log.V(log_level.DEBUG).Info("Copying audit log read credentials to SKR")

	runtimeID := s.instance.Labels[imv1.LabelKymaRuntimeID]

	// Get audit log data - uses cached client, essentially free
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

	// Skip if no read credentials configured
	if auditLogData.ReadCredsSecretName == "" {
		m.log.Info("No read credentials secret configured, skipping copy", "runtimeID", runtimeID)
		completeProvisioning(&s.instance)
		return updateStatusAndStop()
	}

	// Copy credentials to SKR
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
	// Get SKR client
	runtimeClient, err := m.RuntimeClientGetter.Get(ctx, s.instance)
	if err != nil {
		return fmt.Errorf("failed to get runtime client: %w", err)
	}

	// Read source secret from KCP namespace
	var sourceSecret corev1.Secret
	secretKey := types.NamespacedName{
		Name:      sourceSecretName,
		Namespace: s.instance.Namespace,
	}
	if err := m.KcpClient.Get(ctx, secretKey, &sourceSecret); err != nil {
		return fmt.Errorf("failed to get source secret %s: %w", secretKey, err)
	}

	// Create target secret for SKR
	targetSecret := corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
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

	// Apply secret with server-side apply (idempotent)
	//nolint:staticcheck // SA1019: client.Apply is used with Patch, which is the correct API for this version
	if err := runtimeClient.Patch(ctx, &targetSecret, client.Apply, &client.PatchOptions{
		FieldManager: fieldManagerName,
		Force:        ptr.To(true),
	}); err != nil {
		return fmt.Errorf("failed to apply secret to SKR: %w", err)
	}

	return nil
}

func completeProvisioning(instance *imv1.Runtime) {
	if !instance.IsProvisioningCompletedStatusSet() {
		instance.UpdateStateProvisioningCompleted()
	}
}
