# Context

This document defines the architecture for copying dedicated audit log read credentials to the Kyma Runtime cluster (SKR) as part of the dedicated audit logging feature described in [ADR 004: Dedicated Audit Logging](./004-dedicated-audit-logging.md).

# Status

Proposed

# Background

The dedicated audit logging feature (ADR 004) implements a two-phase process:
1. **Phase 1**: Reserve an AuditLogCR before shoot creation
2. **Phase 2**: Claim the AuditLogCR and patch the shoot with dedicated write credentials in `sFnMigrateToDedicatedAuditLog`

However, according to KALM architecture specifications, there is a third step missing: **copying read credentials to the runtime cluster** to enable users to access their audit logs directly.

## Current Gap

The `sFnMigrateToDedicatedAuditLog` function currently:
- ✅ Claims the AuditLogCR (upgrades reservation to full claim)
- ✅ Reads write credentials configuration from AuditLogCR
- ✅ Patches shoot with audit log extension configuration (destination URL, tenant ID, Gardener secret)
- ❌ **Missing**: Copy read credentials to the runtime cluster

## Read Credentials Overview

The AuditLogCR contains a reference to read credentials in `spec.config.readCredsSecretName`. This secret:
- Lives in the KCP namespace alongside the AuditLogCR
- Contains OAuth credentials for read-only access to audit logs
- Is managed by KALM (Kyma Audit Log Manager)
- Needs to be copied to the SKR's `kyma-system` namespace as `auditlog-read-credentials`

Users can then use these credentials to query their dedicated audit logs via the audit log service API.

# Decision

## Architectural Approach

We extend the `sFnMigrateToDedicatedAuditLog` state to include read credentials copy as a **final step** after successfully patching the shoot. This approach:

1. Groups all dedicated audit log operations in a single FSM state
2. Maintains the existing flow without adding new FSM states
3. Copies credentials only after shoot configuration succeeds

### Extended FSM State Flow

```
sFnMigrateToDedicatedAuditLog:
    Step 1: Claim AuditLogCR (existing)
    Step 2: Get current shoot config (existing)
    Step 3: Compare configurations (existing)
    Step 4: Patch shoot if needed (existing)
    Step 5: Copy read credentials to SKR (NEW)
    Step 6: Complete provisioning
```

## Implementation Details

### Extending AuditLogData Structure

Add read credentials secret name to the `AuditLogData` struct:

```go
type AuditLogData struct {
    TenantID              string
    ServiceURL            string
    SecretName            string  // Write credentials (Gardener secret)
    ReadCredsSecretName   string  // Read credentials (KCP namespace) - NEW
}
```

### Extending GetDedicatedAuditLogData

The `GetDedicatedAuditLogData` method in `pkg/auditlog/provider.go` already returns data from the AuditLogCR. We extend it to include the read credentials secret name:

```go
return AuditLogData{
    TenantID:            auditLogCR.Spec.SubaccountID,
    ServiceURL:          auditLogCR.Spec.Config.ServiceURL,
    SecretName:          auditLogCR.Spec.Config.GardenerSecretName,
    ReadCredsSecretName: auditLogCR.Spec.Config.ReadCredsSecretName,  // NEW
}, nil
```

### New Helper Function: copyReadCredentialsToSKR

```go
func copyReadCredentialsToSKR(ctx context.Context, m *fsm, s *systemState, auditLogData auditlog.AuditLogData) error {
    // Skip if no read credentials configured
    if auditLogData.ReadCredsSecretName == "" {
        m.log.Info("No read credentials secret configured, skipping copy")
        return nil
    }

    // Get SKR client
    runtimeClient, err := m.RuntimeClientGetter.Get(ctx, s.instance)
    if err != nil {
        return fmt.Errorf("failed to get runtime client: %w", err)
    }

    // Read source secret from KCP namespace
    var sourceSecret corev1.Secret
    secretKey := types.NamespacedName{
        Name:      auditLogData.ReadCredsSecretName,
        Namespace: s.instance.Namespace,  // KCP namespace where AuditLogCR lives
    }
    if err := m.Client.Get(ctx, secretKey, &sourceSecret); err != nil {
        return fmt.Errorf("failed to get read credentials secret %s: %w", secretKey, err)
    }

    // Create target secret for SKR
    targetSecret := corev1.Secret{
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

    // Apply secret with server-side apply (idempotent)
    if err := runtimeClient.Patch(ctx, &targetSecret, client.Apply, &client.PatchOptions{
        FieldManager: fieldManagerName,
        Force:        ptr.To(true),
    }); err != nil {
        return fmt.Errorf("failed to apply read credentials secret to SKR: %w", err)
    }

    return nil
}
```

### Extended sFnMigrateToDedicatedAuditLog

```go
func sFnMigrateToDedicatedAuditLog(ctx context.Context, m *fsm, s *systemState) (stateFn, *ctrl.Result, error) {
    // ... existing checks for feature flags ...

    // Step 1: Claim AuditLogCR (existing)
    auditLogData, err := m.AuditLogDataProvider.GetDedicatedAuditLogData(ctx, runtimeID, true)
    if err != nil {
        // ... existing error handling ...
    }

    // Step 2-4: Compare and patch shoot (existing)
    // ... existing code ...

    // Step 5: Copy read credentials to SKR (NEW)
    if err := copyReadCredentialsToSKR(ctx, m, s, auditLogData); err != nil {
        m.log.Error(err, "Failed to copy read credentials to SKR, will retry",
            "runtimeID", runtimeID)
        
        s.instance.UpdateStatePending(
            imv1.ConditionTypeCustomAuditLogConfigured,
            imv1.ConditionReasonCustomAuditLogError,
            metav1.ConditionFalse,
            fmt.Sprintf("Failed to copy read credentials: %v", err),
        )
        
        return updateStatusAndRequeueAfter(m.ControlPlaneRequeueDuration)
    }

    m.log.Info("Successfully copied read credentials to SKR",
        "runtimeID", runtimeID,
        "secretName", auditLogData.ReadCredsSecretName)

    // Step 6: Complete provisioning (existing)
    // ... existing completion code ...
}
```

## Target Secret Specification

| Property | Value |
|----------|-------|
| Name | `auditlog-read-credentials` |
| Namespace | `kyma-system` |
| Label | `kyma-project.io/managed-by: infrastructure-manager` |
| Content | OAuth credentials (copied from source secret data) |

## Error Handling

### Source Secret Not Found

If `AuditLogCR.spec.config.readCredsSecretName` references a non-existent secret:
- Log error with details
- Update condition with error message
- Requeue for retry (KALM may not have created the secret yet)

### SKR Client Unavailable

If runtime client cannot be obtained (kubeconfig issues):
- Log error
- Update condition with error message
- Requeue for retry

### Apply Failure

If server-side apply fails:
- Log error with details
- Update condition with error message
- Requeue for retry

### No Read Credentials Configured

If `readCredsSecretName` is empty:
- Log info message (not an error)
- Skip copy operation
- Continue with provisioning completion

## Idempotency

The implementation is fully idempotent:

1. **Server-side apply**: Using `client.Apply` with force ensures the secret is created or updated correctly
2. **No duplicate creates**: Patch-based approach handles both create and update scenarios
3. **Safe retries**: Failed copy operations can be retried without side effects
4. **Reconciliation-safe**: Next reconciliation will apply the same secret state

## Cleanup

Read credentials secret cleanup is **automatic** and does not require explicit implementation:

1. When a runtime is deleted, the `kyma-system` namespace in SKR is deleted
2. All secrets in `kyma-system`, including `auditlog-read-credentials`, are garbage collected
3. No orphaned secrets remain

This aligns with the existing behavior for other SKR resources managed by KIM.

# Rejected Alternatives

## Alternative 1: Separate FSM State

**Approach**: Create a new state `sFnCopyAuditLogReadCredentials` after `sFnMigrateToDedicatedAuditLog`

**Rejected because**:
- Adds complexity with additional FSM state
- Requires extra reconciliation iteration
- Logically part of the same "migrate to dedicated" operation
- No clear benefit from separation

## Alternative 2: Copy Before Shoot Patch

**Approach**: Copy read credentials before patching the shoot

**Rejected because**:
- Credentials should only be available after configuration succeeds
- If shoot patch fails, credentials would exist without valid audit log setup
- Conceptually, users get access after everything is configured

## Alternative 3: Sync During Every Reconciliation

**Approach**: Always sync read credentials during reconciliation, not just during provisioning

**Rejected because**:
- Unnecessary overhead for runtime operations
- Credentials don't change during runtime lifecycle
- KALM credentials are immutable once set

## Alternative 4: Copy Credentials to Gardener Secret Store

**Approach**: Store read credentials in Gardener secrets (like write credentials)

**Rejected because**:
- Write credentials are for Gardener's audit log extension
- Read credentials are for user access, not Gardener
- Different lifecycle and purpose

# Consequences

## Positive

1. **Complete feature**: Users can access dedicated audit logs with provided credentials
2. **Simple implementation**: Extends existing state without new FSM states
3. **Idempotent**: Server-side apply ensures safe retries
4. **Self-cleaning**: Namespace deletion handles cleanup
5. **Consistent pattern**: Follows existing SKR secret management patterns (registry cache)

## Negative

1. **Additional network call**: Requires runtime client connection to SKR
2. **Retry loop**: Failed copy triggers requeue, may delay provisioning completion
3. **Cross-cluster dependency**: Relies on SKR availability

## Neutral

1. **Conditional behavior**: Only executes when `readCredsSecretName` is set
2. **KALM dependency**: Requires KALM to populate read credentials secret

# Implementation Checklist

- [ ] Extend `AuditLogData` struct with `ReadCredsSecretName` field
- [ ] Update `GetDedicatedAuditLogData` to return read credentials secret name
- [ ] Implement `copyReadCredentialsToSKR` helper function
- [ ] Extend `sFnMigrateToDedicatedAuditLog` with read credentials copy
- [ ] Add unit tests for `copyReadCredentialsToSKR`
- [ ] Add integration tests for end-to-end flow
- [ ] Update documentation

# References

- [Issue #1496: Copy dedicated audit log read credentials](https://github.com/kyma-project/kyma-infrastructure-manager/issues/1496)
- [ADR 004: Dedicated Audit Logging](./004-dedicated-audit-logging.md)
- [ADR 004: Implementation Details](./004-dedicated-audit-logging-implementation.md)
- [AuditLog Types Definition](../../pkg/auditlog/v1beta1/auditlog_types.go)
