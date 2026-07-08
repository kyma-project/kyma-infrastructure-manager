# Enable Dedicated Audit Logging for Existing Runtimes - Implementation Details

This document provides detailed implementation guidance for enabling dedicated audit logging on existing runtimes as described in [ADR 006: Enable Dedicated Audit Logging for Existing Runtimes](./006-enable-dedicated-auditlog-for-existing-runtimes.md).

## Table of Contents

- [Modified sFnPatchExistingShoot](#modified-sfnpatchexistingshoot)
- [Upgrade Detection Logic](#upgrade-detection-logic)
- [Irreversibility Enforcement](#irreversibility-enforcement)
- [Condition Status Updates](#condition-status-updates)
- [Error Handling and Edge Cases](#error-handling-and-edge-cases)
- [Complete Flow Diagram](#complete-flow-diagram)
- [Testing](#testing)

## Modified sFnPatchExistingShoot

The `sFnPatchExistingShoot` function is extended to handle the upgrade scenario by detecting when the dedicated audit logging flag is set on Runtime CR but no AuditLog is assigned, and then reserving an AuditLogCR.

### File Location

`internal/controller/runtime/fsm/runtime_fsm_patch_shoot.go`

### Updated Implementation

```go
func sFnPatchExistingShoot(ctx context.Context, m *fsm, s *systemState) (stateFn, *ctrl.Result, error) {

    var data auditlog.AuditLogData
    var err error

    // Fast path: if dedicated audit logging feature is disabled globally, use shared config
    if !m.DedicatedAuditLoggingEnabled {
        data, err = m.AuditLogDataProvider.GetSharedAuditLogData(
            ctx,
            s.instance.Spec.Shoot.Provider.Type,
            s.instance.Spec.Shoot.Region,
        )
        return continueWithAuditLogData(ctx, m, s, data, err)
    }

    runtimeID := s.instance.Labels[imv1.LabelKymaRuntimeID]
    dedicatedFlagEnabled := s.instance.Spec.AuditLogAccessEnabled != nil && *s.instance.Spec.AuditLogAccessEnabled

    // Check if AuditLog is already assigned to this runtime (reserved or claimed)
    existingData, existingErr := m.AuditLogDataProvider.GetDedicatedAuditLogData(ctx, runtimeID, false)
    hasExistingAuditLog := existingErr == nil

    // Irreversibility: once dedicated is configured, it cannot be disabled
    if hasExistingAuditLog {
        if !dedicatedFlagEnabled {
            m.log.Info("Dedicated audit logging is irreversible - ignoring attempt to disable",
                "runtimeID", runtimeID)
        }
        data = existingData
        return continueWithAuditLogData(ctx, m, s, data, nil)
    }

    // No existing AuditLog assigned - check if user wants dedicated logging
    if !dedicatedFlagEnabled {
        // User doesn't want dedicated logging and none is assigned - use shared
        data, err = m.AuditLogDataProvider.GetSharedAuditLogData(
            ctx,
            s.instance.Spec.Shoot.Provider.Type,
            s.instance.Spec.Shoot.Region,
        )
        return continueWithAuditLogData(ctx, m, s, data, err)
    }

    // Upgrade scenario: dedicated flag is set but no AuditLog is assigned
    // Phase 1: Reserve an AuditLogCR for this runtime
    m.log.Info("No existing dedicated audit log found, attempting to reserve for upgrade",
        "runtimeID", runtimeID,
        "region", s.instance.Spec.Shoot.Region)

    reserveErr := m.AuditLogDataProvider.ReserveAuditLog(
        ctx,
        s.instance.Spec.Shoot.Region,
        runtimeID,
    )
    if reserveErr != nil {
        msg := fmt.Sprintf("Dedicated audit logging requested but no available configuration found: %v", reserveErr)
        m.log.Error(reserveErr, "Cannot upgrade runtime to dedicated audit logging")
        m.Metrics.IncRuntimeFSMStopCounter()
        return updateStateFailedWithErrorAndStop(
            &s.instance,
            imv1.ConditionTypeRuntimeProvisioned,
            imv1.ConditionReasonCustomAuditLogError,
            msg)
    }

    m.log.Info("Successfully reserved dedicated audit log for runtime upgrade",
        "runtimeID", runtimeID)

    // Set condition to indicate migration is pending
    s.instance.UpdateStatePending(
        imv1.ConditionTypeCustomAuditLogConfigured,
        imv1.ConditionReasonCustomAuditLogConfigured,
        metav1.ConditionUnknown,
        "Dedicated audit logging reserved, migration pending",
    )

    // Use shared config for this patch cycle
    // Migration to dedicated will happen in sFnMigrateToDedicatedAuditLog
    data, err = m.AuditLogDataProvider.GetSharedAuditLogData(
        ctx,
        s.instance.Spec.Shoot.Provider.Type,
        s.instance.Spec.Shoot.Region,
    )

    return continueWithAuditLogData(ctx, m, s, data, err)
}

// continueWithAuditLogData continues the patch flow with the resolved audit log data
func continueWithAuditLogData(ctx context.Context, m *fsm, s *systemState, data auditlog.AuditLogData, err error) (stateFn, *ctrl.Result, error) {
    if err != nil {
        m.log.Error(err, msgFailedToConfigureAuditlogs)
    }

    if err != nil && m.AuditLogMandatory {
        m.Metrics.IncRuntimeFSMStopCounter()
        return updateStateFailedWithErrorAndStop(
            &s.instance,
            imv1.ConditionTypeRuntimeProvisioned,
            imv1.ConditionReasonAuditLogError,
            msgFailedToConfigureAuditlogs)
    }

    // ... rest of the patch logic (OIDC, shoot conversion, patch, etc.) ...
}
```

### Decision Flow Summary

The refactored code follows this decision flow:

1. **Global feature disabled?** → Use shared config
2. **AuditLog already assigned?** → Use existing dedicated config (irreversibility enforced)
3. **Dedicated flag not set?** → Use shared config
4. **Dedicated flag set, no AuditLog?** → Reserve new AuditLog, use shared config for this cycle

## Upgrade Detection Logic

The upgrade scenario is detected when the dedicated audit logging flag is set on Runtime CR but no AuditLog is assigned. This is determined by the failure of `GetDedicatedAuditLogData(claim=false)`:

```go
data, err = m.AuditLogDataProvider.GetDedicatedAuditLogData(ctx, runtimeID, false)
if err != nil {
    // This means no AuditLog is assigned to this runtime:
    // 1. No AuditLogCR is claimed for this runtime (spec.assignedToRuntimeID != runtimeID)
    // 2. No AuditLogCR is reserved for this runtime (no "reserved-for-runtime-id" label)
    // Therefore, we need to make a new reservation
}
```

### Detection Matrix

| Scenario | GetDedicatedAuditLogData Result | Action |
|----------|--------------------------------|--------|
| New runtime, dedicated flag enabled | Error (no AuditLog assigned) | Reserve in `sFnCreateShoot` (existing flow) |
| Existing runtime, dedicated flag enabled, no AuditLog assigned | Error (no AuditLog assigned) | Reserve in `sFnPatchExistingShoot` (new flow) |
| Existing runtime, dedicated flag enabled, AuditLog reserved | Success (returns data) | Use existing reservation, no new reserve |
| Existing runtime, dedicated flag enabled, AuditLog claimed | Success (returns data) | Use existing claim, no action needed |

## Irreversibility Enforcement

**Important**: Enabling dedicated audit logging is an irreversible operation. Once a runtime has dedicated audit logging configured, it cannot be downgraded back to shared logging.

### Implementation

When `auditLogAccessEnabled` is set to `false` but dedicated logging was previously configured:

```go
// In sFnPatchExistingShoot, when dedicatedAuditLogs is false
if m.DedicatedAuditLoggingEnabled {
    runtimeID := s.instance.Labels[imv1.LabelKymaRuntimeID]
    existingData, existingErr := m.AuditLogDataProvider.GetDedicatedAuditLogData(ctx, runtimeID, false)
    
    if existingErr == nil {
        // Dedicated logging was previously configured - downgrade not allowed
        m.log.Info("Dedicated audit logging is irreversible - ignoring attempt to disable",
            "runtimeID", runtimeID)
        
        // Continue with dedicated config instead
        data = existingData
        dedicatedAuditLogs = true // Override the flag
    }
}
```

### Behavior

| User Action | Previous State | Result |
|-------------|---------------|--------|
| Set `auditLogAccessEnabled: true` | Shared logging | Upgrade to dedicated |
| Set `auditLogAccessEnabled: false` | Shared logging | Remains shared |
| Set `auditLogAccessEnabled: false` | Dedicated logging | **Ignored** - remains dedicated |
| Remove `auditLogAccessEnabled` field | Dedicated logging | **Ignored** - remains dedicated |

### Rationale

1. **Resource commitment**: Dedicated AuditLogCR resources are provisioned specifically for the runtime
2. **Compliance**: Audit log continuity is critical for compliance requirements
3. **Simplicity**: Avoiding downgrade eliminates complex state transitions and edge cases
4. **Cost**: Dedicated resources have associated costs that should not be wasted

## Condition Status Updates

### During Upgrade (after reservation)

```go
s.instance.UpdateStatePending(
    imv1.ConditionTypeCustomAuditLogConfigured,
    imv1.ConditionReasonCustomAuditLogConfigured,
    metav1.ConditionUnknown,
    "Dedicated audit logging reserved, migration pending",
)
```

### After Migration Complete (in sFnMigrateToDedicatedAuditLog)

```go
s.instance.UpdateStateReady(
    imv1.ConditionTypeCustomAuditLogConfigured,
    imv1.ConditionReasonCustomAuditLogConfigured,
    "Custom AuditLog shoot configuration completed",
)
```

### Condition Progression

| State | ConditionTypeCustomAuditLogConfigured |
|-------|--------------------------------------|
| Before upgrade | Not present |
| After reservation in `sFnPatchExistingShoot` | Unknown: "migration pending" |
| After shoot patch in `sFnMigrateToDedicatedAuditLog` | Unknown: "configuration completed" |
| After next reconciliation (configs equal) | True: "configuration completed" |
| After `sFnCopyAuditLogReadCredentials` | True (unchanged) |

## Error Handling and Edge Cases

### Upgrade Fails - No Available AuditLogCR

**Scenario**: User requests dedicated logging but pool is exhausted.

**Handling**:
```go
if reserveErr != nil {
    msg := fmt.Sprintf("Dedicated audit logging requested but no available configuration found: %v", reserveErr)
    m.log.Error(reserveErr, "Cannot upgrade runtime to dedicated audit logging")
    m.Metrics.IncRuntimeFSMStopCounter()
    return updateStateFailedWithErrorAndStop(
        &s.instance,
        imv1.ConditionTypeRuntimeProvisioned,
        imv1.ConditionReasonCustomAuditLogError,
        msg)
}
```

**Result**: Runtime CR status shows clear error, user can retry when pool has capacity.

### Upgrade Interrupted After Reservation

**Scenario**: Controller restarts after reservation but before claim.

**Handling**:
1. Next reconciliation calls `GetDedicatedAuditLogData(claim=false)`
2. Finds existing reservation via label
3. Returns data successfully
4. No duplicate reservation attempted
5. Flow continues normally

### Concurrent Upgrade Requests

**Scenario**: Two reconciliation loops try to reserve simultaneously.

**Handling**:
- Kubernetes optimistic concurrency prevents double-reservation
- One succeeds, other gets conflict error and retries
- Retry finds existing reservation

### Shared Config Unavailable After Reservation

**Scenario**: Reservation succeeds but `GetSharedAuditLogData` fails.

**Handling**:
```go
data, err = m.AuditLogDataProvider.GetSharedAuditLogData(...)
if err != nil && m.AuditLogMandatory {
    // Fail - but reservation remains
    // On retry, we'll find the reservation and try shared again
    return updateStateFailedWithErrorAndStop(...)
}
```

**Note**: Reservation is not rolled back. It will either be:
- Used on successful retry
- Cleaned up by automated job after 1-hour timeout

## Complete Flow Diagram

```
sFnSelectShootProcessing (Runtime CR changed)
    │
    ├─── shouldPatchShoot() returns true ────────────────────────────────────┐
    │                                                                         │
    ▼                                                                         │
sFnSyncRegistryCacheGardenSecrets                                            │
    │                                                                         │
    ▼                                                                         │
sFnPatchExistingShoot                                                        │
    │                                                                         │
    ├─── dedicatedAuditLogs=true ───────────────────────────────┐            │
    │                                                            │            │
    │                                                            ▼            │
    │                                          GetDedicatedAuditLogData(claim=false)
    │                                                            │            │
    │                                    ┌───────────────────────┼───────────┐│
    │                                    │                       │           ││
    │                              (success)                  (error)        ││
    │                                    │                       │           ││
    │                                    │                       ▼           ││
    │                                    │               ReserveAuditLog     ││
    │                                    │                       │           ││
    │                                    │           ┌───────────┼──────────┐││
    │                                    │           │           │          │││
    │                                    │      (success)    (error)        │││
    │                                    │           │           │          │││
    │                                    │           │           ▼          │││
    │                                    │           │    FAIL + STOP       │││
    │                                    │           │                      │││
    │                                    │           ▼                      │││
    │                                    │   GetSharedAuditLogData          │││
    │                                    │           │                      │││
    │                                    └─────┬─────┴──────────────────────┘││
    │                                          │                             ││
    ├─── dedicatedAuditLogs=false ─────────────┤                             ││
    │           │                              │                             ││
    │           ▼                              │                             ││
    │   Check for existing dedicated           │                             ││
    │   (irreversibility check)                │                             ││
    │           │                              │                             ││
    │    ┌──────┴──────┐                       │                             ││
    │    │             │                       │                             ││
    │ (exists)    (not exists)                 │                             ││
    │    │             │                       │                             ││
    │    ▼             ▼                       │                             ││
    │ Use dedicated  GetSharedAuditLogData     │                             ││
    │ (ignore flag)    │                       │                             ││
    │    │             │                       │                             ││
    │    └──────┬──────┘                       │                             ││
    │           │                              │                             ││
    │           └──────────────────────────────┤                             ││
    │                                          │                             ││
    │                                          ▼                             ││
    │                                   Patch Shoot                          ││
    │                                          │                             ││
    │                                          ▼                             ││
    │                             sFnWaitForShootReconcile                   ││
    │                                          │                             ││
    │                                          ▼                             ││
    │                               sFnHandleKubeconfig                      ││
    │                                          │                             ││
    │                                          ▼                             ││
    │                             sFnCreateKymaNamespace                     ││
    │                                          │                             ││
    │                                          ▼                             ││
    │                        sFnInitializeRuntimeBootstrapper                ││
    │                                          │                             ││
    │                                          ▼                             ││
    │                     sFnCleanupRegistryCacheGardenSecrets               ││
    │                                          │                             ││
    │                                          ▼                             ││
    │                               sFnConfigureSKR                          ││
    │                                          │                             ││
    │                                          ▼                             ││
    │                        sFnApplyClusterRoleBindings                     ││
    │                                          │                             ││
    │            ┌─────────────────────────────┼────────────────────────────┐││
    │            │                             │                            │││
    │    (dedicated enabled)           (dedicated disabled)                 │││
    │            │                             │                            │││
    │            ▼                             ▼                            │││
    │   sFnMigrateToDedicatedAuditLog   Complete Provisioning               │││
    │            │                                                          │││
    │            ▼                                                          │││
    │   GetDedicatedAuditLogData(claim=true)                               │││
    │            │                                                          │││
    │            ▼                                                          │││
    │   Compare shoot config                                                │││
    │            │                                                          │││
    │    ┌───────┼────────┐                                                 │││
    │    │       │        │                                                 │││
    │ (equal) (differ)    │                                                 │││
    │    │       │        │                                                 │││
    │    │       ▼        │                                                 │││
    │    │  Patch shoot   │                                                 │││
    │    │       │        │                                                 │││
    │    │       ▼        │                                                 │││
    │    │   Requeue      │                                                 │││
    │    │       │        │                                                 │││
    │    │       └────────┤ (next reconciliation)                           │││
    │    │                │                                                 │││
    │    └────────────────┤                                                 │││
    │                     │                                                 │││
    │                     ▼                                                 │││
    │   sFnCopyAuditLogReadCredentials                                      │││
    │                     │                                                 │││
    │                     ▼                                                 │││
    │           Complete Provisioning                                       │││
    │                     │                                                 │││
    └─────────────────────┴─────────────────────────────────────────────────┘││
                                                                             ││
                          updateStatusAndStop()                              ││
                                                                             ││
```

## Testing

### Unit Tests

File: `internal/controller/runtime/fsm/runtime_fsm_patch_shoot_test.go`

#### Test Cases

```go
func TestSFnPatchExistingShoot_UpgradeScenarios(t *testing.T) {
    t.Run("should reserve AuditLogCR when upgrading to dedicated and no reservation exists", func(t *testing.T) {
        // Given: Runtime with auditLogAccessEnabled=true, no existing reservation
        // When: sFnPatchExistingShoot is called
        // Then: ReserveAuditLog is called, GetSharedAuditLogData is used for patch
    })

    t.Run("should not reserve when dedicated reservation already exists", func(t *testing.T) {
        // Given: Runtime with auditLogAccessEnabled=true, existing reservation
        // When: sFnPatchExistingShoot is called
        // Then: ReserveAuditLog is NOT called, existing data is used
    })

    t.Run("should fail when pool exhausted and dedicated logging requested", func(t *testing.T) {
        // Given: Runtime with auditLogAccessEnabled=true, no pool capacity
        // When: sFnPatchExistingShoot is called
        // Then: Returns failed state with CustomAuditLogError reason
    })

    t.Run("should ignore downgrade attempt when dedicated logging already configured", func(t *testing.T) {
        // Given: Runtime with auditLogAccessEnabled=false, existing dedicated claim
        // When: sFnPatchExistingShoot is called
        // Then: Dedicated config is used (downgrade ignored), warning logged
    })

    t.Run("should set condition to Unknown after successful reservation", func(t *testing.T) {
        // Given: Runtime upgrading to dedicated
        // When: Reservation succeeds
        // Then: ConditionTypeCustomAuditLogConfigured is Unknown with "migration pending"
    })
    
    t.Run("should use shared config when no previous dedicated config exists and auditLogAccessEnabled is false", func(t *testing.T) {
        // Given: Runtime with auditLogAccessEnabled=false, no existing dedicated claim
        // When: sFnPatchExistingShoot is called
        // Then: GetSharedAuditLogData is called, shared config is used
    })
}
```

### Integration Tests

```go
func TestUpgradeToDedicatedAuditLogging(t *testing.T) {
    // 1. Create runtime without dedicated audit logging
    runtime := createRuntimeWithoutDedicatedAuditLog()
    
    // 2. Wait for initial provisioning to complete
    waitForProvisioningComplete(runtime)
    
    // 3. Update runtime to enable dedicated audit logging
    runtime.Spec.AuditLogAccessEnabled = ptr.To(true)
    updateRuntime(runtime)
    
    // 4. Wait for upgrade to complete
    waitForCondition(runtime, imv1.ConditionTypeCustomAuditLogConfigured, metav1.ConditionTrue)
    
    // 5. Verify AuditLogCR is claimed
    auditLogCR := getAuditLogCRForRuntime(runtime)
    assert.Equal(t, runtime.Labels[imv1.LabelKymaRuntimeID], auditLogCR.Spec.AssignedToRuntimeID)
    
    // 6. Verify shoot has dedicated audit log config
    shoot := getShootForRuntime(runtime)
    auditLogConfig := getAuditLogConfigFromShoot(shoot)
    assert.Equal(t, auditLogCR.Spec.SubaccountID, auditLogConfig.TenantID)
    
    // 7. Verify read credentials copied to SKR
    secret := getSKRSecret(runtime, "kyma-system", "auditlog-read-credentials")
    assert.NotNil(t, secret)
}
```

## Logging

Key log events for observability:

```go
// Upgrade detection
m.log.Info("No existing dedicated audit log found, attempting to reserve for upgrade",
    "runtimeID", runtimeID,
    "region", s.instance.Spec.Shoot.Region)

// Successful reservation
m.log.Info("Successfully reserved dedicated audit log for runtime upgrade",
    "runtimeID", runtimeID)

// Reservation failure
m.log.Error(reserveErr, "Cannot upgrade runtime to dedicated audit logging",
    "runtimeID", runtimeID)

// Downgrade attempt (ignored due to irreversibility)
m.log.Info("Dedicated audit logging is irreversible - ignoring attempt to disable",
    "runtimeID", runtimeID)
```

## References

- [ADR 006: Enable Dedicated Audit Logging for Existing Runtimes](./006-enable-dedicated-auditlog-for-existing-runtimes.md)
- [ADR 004: Dedicated Audit Logging](./004-dedicated-audit-logging.md)
- [ADR 004: Implementation Details](./004-dedicated-audit-logging-implementation.md)
- [ADR 005: Copy AuditLog Read Credentials](./005-copy-auditlog-read-credentials.md)
- [ADR 005: Implementation Details](./005-copy-auditlog-read-credentials-implementation.md)
