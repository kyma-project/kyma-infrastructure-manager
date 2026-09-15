# Copy Audit Log Read Credentials - Implementation Details

This document provides detailed implementation guidance for the read credentials copy feature described in [ADR 005: Copy Audit Log Read Credentials](./005-copy-auditlog-read-credentials.md).

## Table of Contents

- [AuditLogData Structure Extension](#auditlogdata-structure-extension)
- [FSM State Implementation](#fsm-state-implementation)
- [State Transition Flow](#state-transition-flow)
- [Condition Types](#condition-types)
- [Error Handling](#error-handling)
- [Idempotency](#idempotency)

## AuditLogData Structure Extension

The `AuditLogData` struct in `pkg/auditlog/shared.go` is extended with the read credentials secret name:

```go
type AuditLogData struct {
    TenantID            string `json:"tenantID" validate:"required"`
    ServiceURL          string `json:"serviceURL" validate:"required,url"`
    SecretName          string `json:"secretName" validate:"required"`
    ReadCredsSecretName string `json:"readCredsSecretName,omitempty"` // Only used for dedicated audit logging
}
```

The `GetDedicatedAuditLogData` method in `pkg/auditlog/provider.go` populates this field from the AuditLogCR:

```go
return AuditLogData{
    TenantID:            reserved.Spec.SubaccountID,
    ServiceURL:          reserved.Spec.Config.ServiceURL,
    SecretName:          reserved.Spec.Config.GardenerSecretName,
    ReadCredsSecretName: reserved.Spec.Config.ReadCredsSecretName,
}, nil
```

## FSM State Implementation

### File Location

`internal/controller/runtime/fsm/runtime_fsm_copy_auditlog_credentials.go`

### State Function

```go
func sFnCopyAuditLogReadCredentials(ctx context.Context, m *fsm, s *systemState) (stateFn, *ctrl.Result, error) {
    m.log.V(log_level.DEBUG).Info("Copying audit log read credentials to SKR")

    runtimeID := s.instance.Labels[imv1.LabelKymaRuntimeID]

    // Get audit log data - uses cached client, essentially free
    auditLogData, err := m.AuditLogDataProvider.GetDedicatedAuditLogData(ctx, runtimeID, false)
    if err != nil {
        // Handle error - update condition, requeue
    }

    // Skip if no read credentials configured
    if auditLogData.ReadCredsSecretName == "" {
        m.log.Info("No read credentials secret configured, skipping copy", "runtimeID", runtimeID)
        completeProvisioning(&s.instance)
        return updateStatusAndStop()
    }

    // Copy credentials to SKR
    if err := copyReadCredentialsToSKR(ctx, m, s, auditLogData.ReadCredsSecretName); err != nil {
        // Handle error - update condition, requeue
    }

    // Success - update condition, complete provisioning
    s.instance.UpdateStateReady(
        imv1.ConditionTypeAuditLogCredentialsCopied,
        imv1.ConditionReasonCredentialsCopied,
        "Audit log read credentials copied to runtime",
    )

    completeProvisioning(&s.instance)
    return updateStatusAndStop()
}
```

### Helper Function: copyReadCredentialsToSKR

```go
func copyReadCredentialsToSKR(ctx context.Context, m *fsm, s *systemState, sourceSecretName string) error {
    // 1. Get SKR client
    runtimeClient, err := m.RuntimeClientGetter.Get(ctx, s.instance)
    if err != nil {
        return fmt.Errorf("failed to get runtime client: %w", err)
    }

    // 2. Read source secret from KCP namespace
    var sourceSecret corev1.Secret
    secretKey := types.NamespacedName{
        Name:      sourceSecretName,
        Namespace: s.instance.Namespace,
    }
    if err := m.KcpClient.Get(ctx, secretKey, &sourceSecret); err != nil {
        return fmt.Errorf("failed to get source secret %s: %w", secretKey, err)
    }

    // 3. Create target secret for SKR
    targetSecret := corev1.Secret{
        TypeMeta: metav1.TypeMeta{
            APIVersion: "v1",
            Kind:       "Secret",
        },
        ObjectMeta: metav1.ObjectMeta{
            Name:      "auditlog-read-credentials",
            Namespace: "kyma-system",
            Labels: map[string]string{
                imv1.LabelKymaManagedBy: "infrastructure-manager",
            },
        },
        Data: sourceSecret.Data,
        Type: sourceSecret.Type,
    }

    // 4. Apply secret with server-side apply (idempotent)
    if err := runtimeClient.Patch(ctx, &targetSecret, client.Apply, &client.PatchOptions{
        FieldManager: fieldManagerName,
        Force:        ptr.To(true),
    }); err != nil {
        return fmt.Errorf("failed to apply secret to SKR: %w", err)
    }

    return nil
}
```

### Helper Function: completeProvisioning

```go
func completeProvisioning(instance *imv1.Runtime) {
    if !instance.IsProvisioningCompletedStatusSet() {
        instance.UpdateStateProvisioningCompleted()
    }
}
```

## State Transition Flow

### Modified sFnMigrateToDedicatedAuditLog

The `sFnMigrateToDedicatedAuditLog` state now chains to the new state instead of completing provisioning:

```go
// When configs are equal (no patch needed)
if configsEqual {
    s.instance.UpdateStateReady(
        imv1.ConditionTypeCustomAuditLogConfigured,
        imv1.ConditionReasonCustomAuditLogConfigured,
        "Custom AuditLog shoot configuration completed",
    )
    
    // Chain to credentials copy state (instead of updateStatusAndStop)
    return switchState(sFnCopyAuditLogReadCredentials)
}
```

### Complete Flow Diagram

```
sFnApplyClusterRoleBindings
    │
    ├─── (dedicated audit logging enabled) ───────────────────────┐
    │                                                              │
    │                                                              ▼
    │                                              sFnMigrateToDedicatedAuditLog
    │                                                              │
    │                                    ┌─────────────────────────┼─────────────────────────┐
    │                                    │                         │                         │
    │                              (claim failed)            (configs differ)          (configs equal)
    │                                    │                         │                         │
    │                                    ▼                         ▼                         │
    │                               FAIL + STOP              Patch shoot                     │
    │                                                              │                         │
    │                                    ┌─────────────────────────┤                         │
    │                                    │                         │                         │
    │                              (patch failed)           (patch success)                  │
    │                                    │                         │                         │
    │                                    ▼                         ▼                         │
    │                               Requeue              Requeue (wait Gardener)             │
    │                                                              │                         │
    │                                                              │ (next reconciliation)   │
    │                                                              │                         │
    │                                                              └─────────────────────────┤
    │                                                                                        │
    │                                                                                        ▼
    │                                                              sFnCopyAuditLogReadCredentials
    │                                                                                        │
    │                                    ┌───────────────────────────────────────────────────┤
    │                                    │                         │                         │
    │                             (data fetch failed)     (no read creds)              (copy success)
    │                                    │                         │                         │
    │                                    ▼                         │                         │
    │                               Requeue                        │                         │
    │                                                              │                         │
    │                                    ┌─────────────────────────┼─────────────────────────┤
    │                                    │                         │                         │
    │                              (copy failed)                   │                         │
    │                                    │                         │                         │
    │                                    ▼                         ▼                         ▼
    │                               Requeue              Complete Provisioning    Complete Provisioning
    │                                                              │                         │
    │                                                              └───────────┬─────────────┘
    │                                                                          │
    ├─── (dedicated audit logging disabled) ───────────────────────────────────┤
    │                                                                          │
    ▼                                                                          ▼
Complete Provisioning                                              updateStatusAndStop()
```

## Condition Types

### New Condition Type

Added to `api/v1/runtime_types.go`:

```go
ConditionTypeAuditLogCredentialsCopied RuntimeConditionType = "AuditLogCredentialsCopied"
```

### New Condition Reasons

```go
ConditionReasonCredentialsCopyError = RuntimeConditionReason("CredentialsCopyErr")
ConditionReasonCredentialsCopied    = RuntimeConditionReason("CredentialsCopied")
```

### Runtime Status Conditions Example

After successful provisioning with dedicated audit logging:

```yaml
status:
  conditions:
    - type: CustomAuditLogConfigured
      status: "True"
      reason: CustomAuditLogConfigured
      message: "Custom AuditLog shoot configuration completed"
    - type: AuditLogCredentialsCopied
      status: "True"
      reason: CredentialsCopied
      message: "Audit log read credentials copied to runtime"
```

## Error Handling

| Scenario | Condition Status | Reason | Action |
|----------|-----------------|--------|--------|
| GetDedicatedAuditLogData fails | False | CredentialsCopyErr | Requeue with ControlPlaneRequeueDuration |
| Runtime client unavailable | False | CredentialsCopyErr | Requeue with ControlPlaneRequeueDuration |
| Source secret not found | False | CredentialsCopyErr | Requeue with ControlPlaneRequeueDuration |
| Apply to SKR fails | False | CredentialsCopyErr | Requeue with ControlPlaneRequeueDuration |
| ReadCredsSecretName empty | (no condition) | - | Skip copy, complete provisioning |
| Success | True | CredentialsCopied | Complete provisioning |

## Idempotency

The implementation is fully idempotent:

1. **GetDedicatedAuditLogData with claim=false**: Re-fetches data without side effects using cached client
2. **Server-side apply**: Using `client.Apply` with `Force: true` ensures:
   - Secret is created if it doesn't exist
   - Secret is updated if it exists with different data
   - No error if secret already exists with same data
3. **completeProvisioning helper**: Checks `IsProvisioningCompletedStatusSet()` before updating
4. **Safe re-entry**: State can be re-entered on any reconciliation without issues

## Target Secret Specification

| Property | Value |
|----------|-------|
| Name | `auditlog-read-credentials` |
| Namespace | `kyma-system` |
| Label | `operator.kyma-project.io/managed-by: infrastructure-manager` |
| Content | Copied from source secret (OAuth credentials) |

## Cleanup

Secret cleanup is automatic:

1. Runtime deletion triggers shoot deletion
2. SKR cluster deletion includes `kyma-system` namespace deletion
3. All secrets in `kyma-system` are garbage collected
4. No explicit cleanup code required in KIM

## Testing

### Unit Tests

File: `internal/controller/runtime/fsm/runtime_fsm_copy_auditlog_credentials_test.go`

Test cases:
1. `should copy credentials and complete provisioning` - Happy path
2. `should skip copy and complete when ReadCredsSecretName is empty` - No-op case
3. `should requeue on GetDedicatedAuditLogData error` - Data fetch failure
4. `should requeue on runtime client error` - SKR client failure
5. `should requeue when source secret not found` - Source secret missing
6. `should handle idempotent re-runs when secret already exists` - Update existing secret

### Integration Testing

Manual verification steps:
1. Create Runtime CR with `auditLogAccessEnabled: true`
2. Wait for provisioning to complete
3. Verify `CustomAuditLogConfigured` condition is True
4. Verify `AuditLogCredentialsCopied` condition is True
5. Connect to SKR cluster
6. Verify secret exists: `kubectl get secret auditlog-read-credentials -n kyma-system`
7. Verify secret has correct labels and data

## References

- [ADR 005: Copy Audit Log Read Credentials](./005-copy-auditlog-read-credentials.md)
- [ADR 004: Dedicated Audit Logging](./004-dedicated-audit-logging.md)
- [ADR 004: Implementation Details](./004-dedicated-audit-logging-implementation.md)
- [AuditLog Types Definition](../../pkg/auditlog/v1beta1/auditlog_types.go)
