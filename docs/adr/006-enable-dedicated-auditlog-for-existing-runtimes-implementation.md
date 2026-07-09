# Implementation Details: Enable Dedicated Audit Logging for Existing Runtimes

This document provides detailed implementation guidance for [ADR 006](./006-enable-dedicated-auditlog-for-existing-runtimes.md).

## Overview

The implementation uses a **direct claim approach** in `sFnPatchExistingShoot` to enable dedicated audit logging for existing runtimes. This is simpler and faster than the two-phase reservation used for new runtime creation.

## Modified sFnPatchExistingShoot Logic

### File Location

`internal/controller/runtime/fsm/runtime_fsm_patch_shoot.go`

### Decision Matrix

```
┌─────────────────────────────┬──────────────────────┬──────────────────────┬──────────────┐
│ Global Feature Flag         │ Runtime Flag         │ Existing AuditLog    │ Action       │
│ DedicatedAuditLoggingEnabled│ AuditLogAccessEnabled│ Assigned?            │              │
├─────────────────────────────┼──────────────────────┼──────────────────────┼──────────────┤
│ false                       │ any                  │ any                  │ Use shared   │
├─────────────────────────────┼──────────────────────┼──────────────────────┼──────────────┤
│ true                        │ true                 │ yes                  │ Use existing │
│                             │                      │                      │ dedicated    │
├─────────────────────────────┼──────────────────────┼──────────────────────┼──────────────┤
│ true                        │ false                │ yes (irreversible)   │ Use existing │
│                             │                      │                      │ dedicated    │
│                             │                      │                      │ (log warning)│
├─────────────────────────────┼──────────────────────┼──────────────────────┼──────────────┤
│ true                        │ false                │ no                   │ Use shared   │
├─────────────────────────────┼──────────────────────┼──────────────────────┼──────────────┤
│ true                        │ true                 │ no                   │ UPGRADE:     │
│                             │                      │                      │ Claim &      │
│                             │                      │                      │ configure    │
└─────────────────────────────┴──────────────────────┴──────────────────────┴──────────────┘
```

### Updated Implementation

The implementation extracts audit log data resolution into a dedicated function `resolveAuditLogData`:

```go
func sFnPatchExistingShoot(ctx context.Context, m *fsm, s *systemState) (stateFn, *ctrl.Result, error) {

    auditLogConfig, nextState, res, err := resolveAuditLogData(ctx, m, s)
    if nextState != nil {
        return nextState, res, err
    }

    // ... rest of the function (OIDC, shoot conversion, patch, etc.) ...
}
```

The `resolveAuditLogData` function encapsulates all audit log configuration logic:

```go
// resolveAuditLogData determines which audit log configuration to use based on:
// - Global feature flag (DedicatedAuditLoggingEnabled)
// - Runtime flag (spec.auditLogAccessEnabled)
// - Existing AuditLog assignment
// Returns the audit log data in extender format and optional state transition in case of error
func resolveAuditLogData(ctx context.Context, m *fsm, s *systemState) (auditlogs.AuditLogData, stateFn, *ctrl.Result, error) {
    var data auditlog.AuditLogData
    var err error

    if !m.DedicatedAuditLoggingEnabled {
        // Global feature disabled → use shared config
        data, err = m.AuditLogDataProvider.GetSharedAuditLogData(
            ctx,
            s.instance.Spec.Shoot.Provider.Type,
            s.instance.Spec.Shoot.Region,
        )
        if err != nil {
            m.log.Error(err, msgFailedToConfigureAuditlogs)
        }

        if err != nil && m.AuditLogMandatory {
            m.Metrics.IncRuntimeFSMStopCounter()
            nextState, res, stateErr := updateStateFailedWithErrorAndStop(
                &s.instance,
                imv1.ConditionTypeRuntimeProvisioned,
                imv1.ConditionReasonAuditLogError,
                msgFailedToConfigureAuditlogs)
            return auditlogs.AuditLogData{}, nextState, res, stateErr
        }
        return toExtenderAuditLogData(data), nil, nil, err
    }

    runtimeID := s.instance.Labels[imv1.LabelKymaRuntimeID]
    dedicatedFlagEnabled := s.instance.Spec.AuditLogAccessEnabled != nil &&
        *s.instance.Spec.AuditLogAccessEnabled

    // Check if AuditLog is already assigned to this runtime
    existingData, existingErr := m.AuditLogDataProvider.GetDedicatedAuditLogData(ctx, runtimeID, false)
    hasExistingAuditLog := existingErr == nil

    if hasExistingAuditLog {
        // Already has dedicated → use it (irreversibility enforced)
        if !dedicatedFlagEnabled {
            m.log.Info("Dedicated audit logging is irreversible - ignoring attempt to disable",
                "runtimeID", runtimeID)
        }
        return toExtenderAuditLogData(existingData), nil, nil, nil
    }

    if !dedicatedFlagEnabled {
        // No existing dedicated, flag not set → use shared
        data, err = m.AuditLogDataProvider.GetSharedAuditLogData(
            ctx,
            s.instance.Spec.Shoot.Provider.Type,
            s.instance.Spec.Shoot.Region,
        )
        if err != nil {
            m.log.Error(err, msgFailedToConfigureAuditlogs)
        }

        if err != nil && m.AuditLogMandatory {
            m.Metrics.IncRuntimeFSMStopCounter()
            nextState, res, stateErr := updateStateFailedWithErrorAndStop(
                &s.instance,
                imv1.ConditionTypeRuntimeProvisioned,
                imv1.ConditionReasonAuditLogError,
                msgFailedToConfigureAuditlogs)
            return auditlogs.AuditLogData{}, nextState, res, stateErr
        }
        return toExtenderAuditLogData(data), nil, nil, err
    }

    // UPGRADE: flag is set but no AuditLog assigned → claim directly
    m.log.Info("Upgrading to dedicated audit logging",
        "runtimeID", runtimeID,
        "region", s.instance.Spec.Shoot.Region)

    data, err = m.AuditLogDataProvider.ClaimAuditLog(
        ctx,
        s.instance.Spec.Shoot.Region,
        runtimeID,
    )
    if err != nil {
        msg := fmt.Sprintf("Dedicated audit logging requested but no available configuration found: %v", err)
        m.log.Error(err, "Cannot upgrade runtime to dedicated audit logging")
        m.Metrics.IncRuntimeFSMStopCounter()
        nextState, res, stateErr := updateStateFailedWithErrorAndStop(
            &s.instance,
            imv1.ConditionTypeRuntimeProvisioned,
            imv1.ConditionReasonCustomAuditLogError,
            msg)
        return auditlogs.AuditLogData{}, nextState, res, stateErr
    }

    m.log.Info("Successfully claimed dedicated audit log for runtime upgrade",
        "runtimeID", runtimeID,
        "tenantID", data.TenantID)

    s.instance.UpdateStatePending(
        imv1.ConditionTypeCustomAuditLogConfigured,
        imv1.ConditionReasonCustomAuditLogConfigured,
        metav1.ConditionUnknown,
        "Dedicated audit logging claimed, configuring shoot",
    )

    return toExtenderAuditLogData(data), nil, nil, nil
}

// toExtenderAuditLogData converts auditlog.AuditLogData to extender auditlogs.AuditLogData
func toExtenderAuditLogData(data auditlog.AuditLogData) auditlogs.AuditLogData {
    return auditlogs.AuditLogData{
        TenantID:   data.TenantID,
        ServiceURL: data.ServiceURL,
        SecretName: data.SecretName,
    }
}
```

**Key points:**
- Extracted into `resolveAuditLogData` function for better separation of concerns
- Returns `auditlogs.AuditLogData` (extender format) directly via `toExtenderAuditLogData` helper
- Error handling for shared config failures integrated within the function
- Error handling for upgrade failure returns state transition immediately
- Handles both mandatory and optional audit log configurations

### Key Differences from Two-Phase Approach

| Aspect | Two-Phase (Creation) | Direct Claim (Upgrade) |
|--------|---------------------|------------------------|
| **First step** | Reserve with labels | Claim with `AssignedToRuntimeID` |
| **Shoot config** | Shared initially | Dedicated immediately |
| **Migration state** | Patches shoot | No-op (already configured) |
| **Gardener reconciliations** | Two | One |

## New DataProvider Method: ClaimAuditLog

### Interface Addition

Add a new method to the `DataProvider` interface:

```go
// DataProvider provides audit logging configuration data
type DataProvider interface {
    // ... existing methods ...

    // ClaimAuditLog finds an available AuditLogCR for the region and claims it directly
    // This is used for upgrade scenarios where we don't need two-phase reservation
    // Returns the audit log data and sets AssignedToRuntimeID on the CR
    ClaimAuditLog(ctx context.Context, providerRegion string, runtimeID string) (AuditLogData, error)
}
```

### Implementation

```go
// ClaimAuditLog finds an available AuditLogCR for the region and claims it directly
func (p *DefaultDataProvider) ClaimAuditLog(ctx context.Context, providerRegion string, runtimeID string) (AuditLogData, error) {
    // First check if already claimed (idempotent)
    existingData, err := p.getDedicatedAuditLogDataWithoutClaim(ctx, runtimeID)
    if err == nil {
        p.logger.Info("AuditLogCR already claimed for runtime", "runtimeID", runtimeID)
        return existingData, nil
    }

    // Find an available AuditLogCR
    auditLogCR, err := p.findAvailableAuditLogCR(ctx, providerRegion)
    if err != nil {
        return AuditLogData{}, fmt.Errorf("failed to find available AuditLogCR: %w", err)
    }
    if auditLogCR == nil {
        return AuditLogData{}, fmt.Errorf("no available AuditLogCR for region %s", providerRegion)
    }

    // Claim it directly by setting AssignedToRuntimeID
    auditLogCR.Spec.AssignedToRuntimeID = runtimeID
    if err := p.client.Update(ctx, auditLogCR); err != nil {
        return AuditLogData{}, fmt.Errorf("failed to claim AuditLogCR: %w", err)
    }

    p.logger.Info("Successfully claimed AuditLogCR for upgrade",
        "name", auditLogCR.Name,
        "runtimeID", runtimeID,
        "region", providerRegion)

    return AuditLogData{
        TenantID:            auditLogCR.Spec.SubaccountID,
        ServiceURL:          auditLogCR.Spec.Config.ServiceURL,
        SecretName:          auditLogCR.Spec.Config.GardenerSecretName,
        ReadCredsSecretName: auditLogCR.Spec.Config.ReadCredsSecretName,
    }, nil
}
```

## Interaction with sFnMigrateToDedicatedAuditLog

For upgrades, when the flow reaches `sFnMigrateToDedicatedAuditLog`, the shoot is already configured with dedicated audit logging. The existing logic handles this correctly:

1. Calls `GetDedicatedAuditLogData(ctx, runtimeID, true)` - finds already-claimed CR
2. Gets current shoot audit log config
3. Compares configs - **they match** because `sFnPatchExistingShoot` already configured it
4. Skips patch, proceeds to `sFnCopyAuditLogReadCredentials`

No changes needed to `sFnMigrateToDedicatedAuditLog`.

## Irreversibility Enforcement

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

## Error Handling

### Claim Failure (Pool Exhausted)

```go
if claimErr != nil {
    // Always fail - user explicitly requested dedicated logging
    return updateStateFailedWithErrorAndStop(
        &s.instance,
        imv1.ConditionTypeRuntimeProvisioned,
        imv1.ConditionReasonCustomAuditLogError,
        msg)
}
```

**No fallback to shared** - the user explicitly requested dedicated logging.

### Patch Failure After Claim

If the shoot patch fails after claiming:
1. The claim persists (AuditLogCR has `AssignedToRuntimeID` set)
2. Function returns with requeue
3. On next reconciliation:
   - `GetDedicatedAuditLogData(ctx, runtimeID, false)` finds the claimed CR
   - Code uses the dedicated config for the patch retry

This is **identical** to how `sFnMigrateToDedicatedAuditLog` handles patch failures.

## Complete Flow Diagram

```
                    sFnPatchExistingShoot
                            │
                            ▼
              ┌─────────────────────────────┐
              │ Global feature enabled?     │
              └─────────────────────────────┘
                     │              │
                    no             yes
                     │              │
                     ▼              ▼
              ┌───────────┐  ┌─────────────────────────────┐
              │Use shared │  │ Check existing AuditLog     │
              │   config  │  │ GetDedicatedAuditLogData    │
              └───────────┘  └─────────────────────────────┘
                                    │
                        ┌───────────┴───────────┐
                      found                   not found
                        │                         │
                        ▼                         ▼
              ┌───────────────────┐    ┌─────────────────────┐
              │ Use existing      │    │ Flag enabled?       │
              │ dedicated config  │    └─────────────────────┘
              │ (log warning if   │           │         │
              │  flag is false)   │          no        yes
              └───────────────────┘           │         │
                                              ▼         ▼
                                    ┌───────────┐ ┌─────────────────┐
                                    │Use shared │ │ UPGRADE:        │
                                    │  config   │ │ ClaimAuditLog   │
                                    └───────────┘ │ Use dedicated   │
                                                  └─────────────────┘
                                                          │
                                              ┌───────────┴───────────┐
                                           success                  failure
                                              │                         │
                                              ▼                         ▼
                                    ┌─────────────────┐       ┌─────────────────┐
                                    │ Continue with   │       │ FAIL runtime    │
                                    │ dedicated config│       │ (pool exhausted)│
                                    └─────────────────┘       └─────────────────┘
```

## Testing

### Unit Tests for sFnPatchExistingShoot

File: `internal/controller/runtime/fsm/runtime_fsm_patch_shoot_test.go`

```go
func TestSFnPatchExistingShoot_DedicatedAuditLogUpgrade(t *testing.T) {
    t.Run("should claim AuditLogCR and use dedicated config when upgrading", func(t *testing.T) {
        // Given: Runtime with auditLogAccessEnabled=true, no existing AuditLog
        // When: sFnPatchExistingShoot is called
        // Then: ClaimAuditLog is called, dedicated config used for patch
    })

    t.Run("should use existing dedicated config when already assigned", func(t *testing.T) {
        // Given: Runtime with auditLogAccessEnabled=true, existing AuditLog claimed
        // When: sFnPatchExistingShoot is called
        // Then: ClaimAuditLog NOT called, existing config used
    })

    t.Run("should fail when pool exhausted and dedicated logging requested", func(t *testing.T) {
        // Given: Runtime with auditLogAccessEnabled=true, ClaimAuditLog returns error
        // When: sFnPatchExistingShoot is called
        // Then: State set to Failed with CustomAuditLogError
    })

    t.Run("should ignore downgrade attempt (irreversibility)", func(t *testing.T) {
        // Given: Runtime with auditLogAccessEnabled=false, existing AuditLog claimed
        // When: sFnPatchExistingShoot is called
        // Then: Warning logged, dedicated config still used
    })

    t.Run("should use shared config when flag not set and no existing dedicated", func(t *testing.T) {
        // Given: Runtime with auditLogAccessEnabled=false, no existing AuditLog
        // When: sFnPatchExistingShoot is called
        // Then: GetSharedAuditLogData called, shared config used
    })

    t.Run("should use shared config when global feature disabled", func(t *testing.T) {
        // Given: DedicatedAuditLoggingEnabled=false
        // When: sFnPatchExistingShoot is called
        // Then: Dedicated check skipped, shared config used
    })
}
```

## Logging

Key log events for observability:

```go
// Upgrade detection and claim
m.log.Info("Upgrading to dedicated audit logging",
    "runtimeID", runtimeID,
    "region", s.instance.Spec.Shoot.Region)

// Successful claim
m.log.Info("Successfully claimed dedicated audit log for runtime upgrade",
    "runtimeID", runtimeID,
    "tenantID", auditLogData.TenantID)

// Claim failure
m.log.Error(claimErr, "Cannot upgrade runtime to dedicated audit logging")

// Downgrade attempt (ignored due to irreversibility)
m.log.Info("Dedicated audit logging is irreversible - ignoring attempt to disable",
    "runtimeID", runtimeID)

// Shared config errors (when mandatory)
m.log.Error(err, msgFailedToConfigureAuditlogs)
```

## Summary

The direct claim approach for upgrades:

1. **Detects** upgrade scenario: flag enabled, no existing AuditLog
2. **Claims** AuditLogCR directly (no reservation phase)
3. **Patches** shoot with dedicated config immediately
4. **Relies on** existing `sFnMigrateToDedicatedAuditLog` for idempotency (it becomes a no-op)
5. **Copies credentials** via existing `sFnCopyAuditLogReadCredentials`

Implementation structure:
- **`resolveAuditLogData()`** - Encapsulates all audit log configuration decision logic
- **`toExtenderAuditLogData()`** - Helper to convert between package types
- **`sFnPatchExistingShoot()`** - Calls `resolveAuditLogData()` and continues with shoot patching

Benefits over two-phase approach:
- Single Gardener reconciliation (saves ~10 minutes)
- No "brief shared logging period" during upgrade
- Cleaner code with extracted function
- Same robustness (claim persists through failures)

## References

- [ADR 006: Enable Dedicated Audit Logging for Existing Runtimes](./006-enable-dedicated-auditlog-for-existing-runtimes.md)
- [ADR 004: Dedicated Audit Logging](./004-dedicated-audit-logging.md)
- [ADR 004: Implementation Details](./004-dedicated-audit-logging-implementation.md)
- [ADR 005: Copy AuditLog Read Credentials](./005-copy-auditlog-read-credentials.md)
- [ADR 005: Implementation Details](./005-copy-auditlog-read-credentials-implementation.md)
