# Context

This document defines the architecture for enabling dedicated audit logging on existing Kyma Runtimes that were initially provisioned without this feature. This extends the dedicated audit logging implementation described in [ADR 004](./004-dedicated-audit-logging.md).

# Status

Proposed

# Background

The current dedicated audit logging feature (ADR 004) only supports provisioning new runtimes with dedicated audit logging enabled from the start. The implementation uses a two-phase reservation algorithm:

1. **Phase 1 (Reserve)**: In `sFnCreateShoot`, before shoot creation, an AuditLogCR is reserved by adding labels
2. **Phase 2 (Claim)**: In `sFnMigrateToDedicatedAuditLog`, the reserved resource is claimed and the shoot is patched with dedicated config
3. **Phase 3 (Copy Credentials)**: In `sFnCopyAuditLogReadCredentials`, read credentials are copied to SKR

## Current Gap

Existing runtimes provisioned with shared audit logging cannot upgrade to dedicated audit logging. When a user updates their Runtime CR to set `auditLogAccessEnabled: true`, the current implementation in `sFnPatchExistingShoot`:

1. Calls `GetDedicatedAuditLogData(ctx, runtimeID, false)` to retrieve dedicated config
2. **Fails because no AuditLogCR was ever reserved for this runtime**
3. Falls back to shared config if audit logging is not mandatory

## Requirements

**Migration Path for Existing Kyma Runtimes**:

- Migration is **opt-in** during Kyma Instance upgrade
- Existing runtimes can enable dedicated audit logging by setting `spec.auditLogAccessEnabled: true`
- Old audit logs from shared regional instance will NOT be migrated
- Only new logs (generated after upgrade) will be stored in the dedicated Audit Log Service instance
- Historical logs remain in shared regional instance (access through SRE support)
- **Enabling dedicated audit logging is irreversible** - once enabled, it cannot be disabled

# Decision

## Architectural Approach

We extend the update flow to support enabling dedicated audit logging using the same two-phase reservation algorithm as creation:

1. **Detection**: In `sFnPatchExistingShoot`, detect when dedicated audit logging flag is set on Runtime CR but no AuditLog is assigned. This means we need to make a new reservation.
2. **Phase 1 (Reserve)**: Reserve an available AuditLogCR for this runtime
3. **Continue with shared**: Patch the shoot using shared audit log config (same as creation flow)
4. **Phase 2 & 3**: After provisioning completes, `sFnMigrateToDedicatedAuditLog` and `sFnCopyAuditLogReadCredentials` handle the migration

### FSM State Flow for Upgrade

```
sFnSelectShootProcessing (detects Runtime CR change)
    ↓
sFnSyncRegistryCacheGardenSecrets
    ↓
sFnPatchExistingShoot (NEW: reserves AuditLogCR if upgrade, patches with shared config)
    ↓
sFnWaitForShootReconcile
    ↓
sFnHandleKubeconfig
    ↓
sFnCreateKymaNamespace (skip if exists)
    ↓
sFnInitializeRuntimeBootstrapper (skip if already done)
    ↓
sFnCleanupRegistryCacheGardenSecrets
    ↓
sFnConfigureSKR
    ↓
sFnApplyClusterRoleBindings
    ↓
sFnMigrateToDedicatedAuditLog (claims reserved AuditLogCR, patches shoot with dedicated config)
    ↓
sFnCopyAuditLogReadCredentials (copies read credentials to SKR)
    ↓
Complete (updateStatusAndStop)
```

**Key Insight**: After `sFnPatchExistingShoot` patches with shared config and Gardener reconciles, the flow continues through all states. When it reaches `sFnMigrateToDedicatedAuditLog`:
1. It finds the reservation made in `sFnPatchExistingShoot`
2. Claims the resource (upgrades from light lock to heavy lock)
3. Patches the shoot AGAIN with dedicated config
4. Gardener reconciles the shoot with new audit log config

## Rejected Alternatives

### Alternative 1: Single-Phase Claim in sFnPatchExistingShoot

**Approach**: Claim AuditLogCR directly in `sFnPatchExistingShoot` and patch shoot with dedicated config immediately.

**Rejected because**:
- If shoot patch fails, the AuditLogCR is wasted (fully claimed but runtime not using it)
- Inconsistent with creation flow which uses two-phase approach
- No validation that the rest of provisioning will succeed

### Alternative 2: Skip Shared Config Phase During Upgrade

**Approach**: When upgrading, reserve and immediately use dedicated config in `sFnPatchExistingShoot`.

**Rejected because**:
- Violates the "test before claiming" principle from ADR 004
- If upgrade fails for other reasons (kubeconfig, RBAC), dedicated resource is wasted
- Inconsistent behavior between creation and upgrade paths

### Alternative 3: New Dedicated Upgrade State

**Approach**: Create a new FSM state `sFnUpgradeToDedicatedAuditLog` specifically for upgrades.

**Rejected because**:
- Duplicates logic already in `sFnMigrateToDedicatedAuditLog`
- Increases FSM complexity unnecessarily
- The existing states can handle upgrade with minimal modification

# Problem Analysis

## Problem 1: Double Shoot Patch

**Issue**: The shoot gets patched twice during upgrade:
1. First in `sFnPatchExistingShoot` with shared audit log config
2. Second in `sFnMigrateToDedicatedAuditLog` with dedicated audit log config

**Impact**:
- Two Gardener reconciliation cycles (~5-10 minutes each)
- Brief period where shoot has shared config before dedicated config is applied
- Increased load on Gardener API

**Mitigation**:
- This is consistent with the creation flow design (ADR 004) which explicitly chose this approach
- The rationale from ADR 004 applies: "Allows shoot to be created and tested before claiming expensive dedicated resources"
- Audit logs generated during the brief shared period remain accessible via SRE support

## Problem 2: Race Condition Window During Upgrade

**Issue**: Between reservation in `sFnPatchExistingShoot` and claim in `sFnMigrateToDedicatedAuditLog`, the reconciliation could be interrupted.

**Conclusion**: Handled correctly by existing idempotent design - reservation labels persist and are found on retry.

## Problem 3: Pool Exhaustion During Upgrade

**Issue**: What if no AuditLogCR is available when user requests upgrade?

**Decision**: **Always fail when dedicated logging is explicitly requested but unavailable**, regardless of `AuditLogMandatory` flag. User explicitly requested the feature - silent fallback violates no-implicit-fallback principle from ADR 004.

## Problem 4: Downgrade Scenario

**Issue**: What if user disables dedicated audit logging after upgrade?

**Decision**: **Downgrade is not supported.** Enabling dedicated audit logging is an irreversible operation.

**Rationale**:
- Dedicated AuditLogCR resources are provisioned specifically for the runtime
- Returning to shared logging would leave the dedicated resource in an ambiguous state
- Audit log continuity and compliance requirements favor a stable, predictable configuration
- Simplifies implementation and reduces edge cases

**Implementation**: The `auditLogAccessEnabled` field change from `true` to `false` should be ignored. If a user attempts to disable dedicated logging after it was enabled, the reconciliation should log a warning and continue with dedicated logging.

## Problem 5: Condition Status During Upgrade

**Issue**: The `ConditionTypeCustomAuditLogConfigured` condition needs proper status during upgrade.

**Decision**: Set condition to `Unknown` after successful reservation to indicate migration is pending.

# Consequences

## Positive

1. **Enables migration path**: Existing customers can opt-in to dedicated audit logging
2. **Reuses existing algorithm**: Two-phase reservation ensures no wasted resources
3. **Consistent with creation flow**: Same guarantees about idempotency and failure handling
4. **Minimal code changes**: Extends existing `sFnPatchExistingShoot` logic

## Negative

1. **Double reconciliation**: Upgrade requires two Gardener shoot reconciliations
2. **Brief shared logging period**: ~5-15 minutes of logs go to shared infrastructure during upgrade
3. **No log migration**: Historical logs remain in shared infrastructure
4. **Irreversible**: Once dedicated audit logging is enabled, it cannot be disabled

## Neutral

1. **Same pool dependency**: Requires KALM pool to have available capacity
2. **Same cleanup requirements**: Stale reservations need manual/automated cleanup
3. **Feature flag dependency**: Both global and per-runtime flags must be set

# Implementation

See [Implementation Details](./006-enable-dedicated-auditlog-for-existing-runtimes-implementation.md) for comprehensive implementation guidance including:

- Modified `sFnPatchExistingShoot` logic
- Irreversibility enforcement (downgrade prevention)
- Condition status updates
- Error handling and edge cases
- Testing approach

# References

- [ADR 004: Dedicated Audit Logging](./004-dedicated-audit-logging.md)
- [ADR 004: Implementation Details](./004-dedicated-audit-logging-implementation.md)
- [ADR 005: Copy AuditLog Read Credentials](./005-copy-auditlog-read-credentials.md)
- [ADR 005: Implementation Details](./005-copy-auditlog-read-credentials-implementation.md)
- [Implementation Details](./006-enable-dedicated-auditlog-for-existing-runtimes-implementation.md)
