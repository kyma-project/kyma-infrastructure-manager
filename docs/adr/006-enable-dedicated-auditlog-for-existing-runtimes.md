# ADR 006: Enable Dedicated Audit Logging for Existing Runtimes

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

## Architectural Approach: Direct Claim in sFnPatchExistingShoot

For upgrades, we use a **direct claim approach** instead of the two-phase reservation used for new runtime creation. This is because the risk profile for upgrades is fundamentally different:

| Concern | New Runtime Creation | Existing Runtime Upgrade |
|---------|---------------------|-------------------------|
| **Shoot exists?** | No - needs to be created | Yes - already running |
| **Provisioning can fail?** | Yes - quota, config, infrastructure | Minimal - shoot already proven |
| **Risk of resource waste?** | High - entire provisioning may fail | Low - shoot is stable |
| **States to traverse?** | Many (create → kubeconfig → namespace → ...) | Few (patch shoot only) |

### Why Direct Claim is Safe for Upgrades

The two-phase reservation in ADR 004 was designed to prevent wasting dedicated AuditLogCR resources when shoot creation might fail. For upgrades:

1. **The shoot already exists and is running** - no risk of creation failure
2. **If patch fails, the claim persists** - next reconciliation retries with same dedicated config
3. **Same recovery behavior as `sFnMigrateToDedicatedAuditLog`** - claim first, patch can retry

### Detection Logic

In `sFnPatchExistingShoot`, detect when dedicated audit logging flag is set on Runtime CR but no AuditLog is assigned. This means we need to claim and configure dedicated logging.

### FSM State Flow for Upgrade

```
sFnSelectShootProcessing (detects Runtime CR change)
    ↓
sFnSyncRegistryCacheGardenSecrets
    ↓
sFnPatchExistingShoot (NEW: claims AuditLogCR and patches with dedicated config directly)
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
sFnMigrateToDedicatedAuditLog (no-op: already configured in sFnPatchExistingShoot)
    ↓
sFnCopyAuditLogReadCredentials (copies read credentials to SKR)
    ↓
Complete (updateStatusAndStop)
```

**Key Insight**: Unlike new runtime creation, `sFnPatchExistingShoot` handles the complete dedicated audit logging configuration in a single step. When the flow reaches `sFnMigrateToDedicatedAuditLog`, it detects that the shoot already has the correct dedicated config and becomes a no-op.

### Benefits of Direct Claim for Upgrades

1. **Single shoot reconciliation** - no double Gardener reconciliation cycles
2. **No "brief shared logging period"** - dedicated config applied immediately
3. **Faster upgrade** - saves ~10 minutes compared to two-phase approach
4. **Simpler code path** - all upgrade logic in one state

## Rejected Alternatives

### Alternative 1: Reuse Two-Phase Reservation (Original Proposal)

**Approach**: Use the same reserve-then-claim pattern as creation:
1. Reserve in `sFnPatchExistingShoot`
2. Continue with shared config
3. Claim in `sFnMigrateToDedicatedAuditLog`

**Rejected because**:
- Unnecessary for upgrades - the shoot already exists and is stable
- Causes double Gardener reconciliation (~10-15 minutes extra)
- Creates "brief shared logging period" during upgrade
- The protection that two-phase provides (against wasted resources on failed creation) doesn't apply to upgrades

### Alternative 2: New Dedicated Upgrade State

**Approach**: Create a new FSM state `sFnUpgradeToDedicatedAuditLog` specifically for upgrades.

**Rejected because**:
- Duplicates logic that can be handled in `sFnPatchExistingShoot`
- Increases FSM complexity unnecessarily
- The upgrade is essentially a patch operation - keep it in the patch state

# Problem Analysis

## Problem 1: What if Claim Succeeds but Patch Fails?

**Issue**: The AuditLogCR is claimed but the shoot patch fails.

**Solution**: This is handled correctly:
- The claim persists (AuditLogCR has `AssignedToRuntimeID` set)
- On next reconciliation, `GetDedicatedAuditLogData(ctx, runtimeID, false)` finds the claimed CR
- Code retries the patch with the same dedicated config
- This is **identical** to how `sFnMigrateToDedicatedAuditLog` handles patch failures today

## Problem 2: Pool Exhaustion During Upgrade

**Issue**: What if no AuditLogCR is available when user requests upgrade?

**Decision**: **Always fail when dedicated logging is explicitly requested but unavailable**, regardless of `AuditLogMandatory` flag. User explicitly requested the feature - silent fallback violates the no-implicit-fallback principle from ADR 004.

## Problem 3: Downgrade Scenario

**Issue**: What if user disables dedicated audit logging after upgrade?

**Decision**: **Downgrade is not supported.** Enabling dedicated audit logging is an irreversible operation.

**Rationale**:
- Dedicated AuditLogCR resources are provisioned specifically for the runtime
- Returning to shared logging would leave the dedicated resource in an ambiguous state
- Audit log continuity and compliance requirements favor a stable, predictable configuration
- Simplifies implementation and reduces edge cases

**Implementation**: The `auditLogAccessEnabled` field change from `true` to `false` should be ignored. If a user attempts to disable dedicated logging after it was enabled, the reconciliation should log a warning and continue with dedicated logging.

## Problem 4: Interaction with sFnMigrateToDedicatedAuditLog

**Issue**: How does this interact with the existing migration state?

**Solution**: `sFnMigrateToDedicatedAuditLog` already handles the case where the shoot is already configured with dedicated logging - it compares configs and skips the patch if they match. For upgrades, when the flow reaches this state, it will:
1. Find the already-claimed AuditLogCR
2. Compare shoot config with desired config
3. Find they match (because `sFnPatchExistingShoot` already configured it)
4. Skip to `sFnCopyAuditLogReadCredentials`

# Consequences

## Positive

1. **Enables migration path**: Existing customers can opt-in to dedicated audit logging
2. **Single reconciliation**: No double Gardener shoot reconciliation
3. **Immediate dedicated logging**: No "brief shared logging period" during upgrade
4. **Faster upgrades**: Saves ~10 minutes compared to two-phase approach
5. **Simpler implementation**: All upgrade logic in one place

## Negative

1. **Claim-before-patch risk**: If patch fails, AuditLogCR remains claimed
   - Mitigated by: Retry behavior identical to existing `sFnMigrateToDedicatedAuditLog`
2. **No log migration**: Historical logs remain in shared infrastructure
3. **Irreversible**: Once dedicated audit logging is enabled, it cannot be disabled

## Neutral

1. **Same pool dependency**: Requires KALM pool to have available capacity
2. **Feature flag dependency**: Both global and per-runtime flags must be set

# Implementation

See [Implementation Details](./006-enable-dedicated-auditlog-for-existing-runtimes-implementation.md) for comprehensive implementation guidance including:

- Modified `sFnPatchExistingShoot` logic with direct claim
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
