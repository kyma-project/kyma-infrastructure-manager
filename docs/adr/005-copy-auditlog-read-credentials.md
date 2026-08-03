# Context

This document defines the architecture for copying dedicated audit log read credentials to the Kyma Runtime cluster (SKR) as part of the dedicated audit logging feature described in [ADR 004: Dedicated Audit Logging](./004-dedicated-audit-logging.md).

# Status

Implemented

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

## Current FSM Flow (Provisioning)

```
sFnCreateShoot
    ↓
sFnWaitForShootCreation
    ↓
sFnHandleKubeconfig
    ↓
sFnCreateKymaNamespace          ← Creates kyma-system namespace in SKR
    ↓
sFnInitializeRuntimeBootstrapper
    ↓
sFnCleanupRegistryCacheGardenSecrets
    ↓
sFnConfigureSKR                 ← Applies ConfigMap + OIDC to SKR
    ↓
sFnApplyClusterRoleBindings     ← Applies ClusterRoleBindings to SKR
    ↓
sFnMigrateToDedicatedAuditLog   ← Claims AuditLogCR, patches shoot
    ↓
Complete (updateStatusAndStop)
```

# Decision Options

## Option 1: Extend sFnMigrateToDedicatedAuditLog

Add read credentials copy as a final step within the existing `sFnMigrateToDedicatedAuditLog` state.

### Pros
- Logical grouping of all dedicated audit log operations
- No new FSM states
- Simple flow

### Cons
- Mixed concerns (Gardener vs SKR operations)
- State complexity increases
- Requeue ambiguity

---

## Option 2: Use Existing SKR Configuration State

Extend `sFnConfigureSKR` or `sFnApplyClusterRoleBindings`.

### Pros
- Reuses existing SKR client
- Follows existing patterns

### Cons
- Semantic mismatch
- **Temporal coupling issue**: Credentials copied BEFORE shoot is configured
- Feature flag complexity in unrelated states

---

## Option 3: Create New State sFnCopyAuditLogReadCredentials (Selected)

Create a dedicated state that runs after `sFnMigrateToDedicatedAuditLog`.

### Pros
- Clean separation of concerns
- Correct temporal ordering
- Independent error handling
- Better observability with separate conditions
- Testable in isolation

### Cons
- Additional FSM state
- Extra reconciliation iteration

# Decision

We create a new FSM state `sFnCopyAuditLogReadCredentials` that executes after `sFnMigrateToDedicatedAuditLog` to copy read credentials to SKR.

## New FSM Flow

```
sFnApplyClusterRoleBindings
    ↓
sFnMigrateToDedicatedAuditLog   ← Claims AuditLogCR, patches shoot
    ↓
sFnCopyAuditLogReadCredentials  ← NEW: Copies read credentials to SKR
    ↓
Complete (updateStatusAndStop)
```

## Target Secret Specification

| Property | Value |
|----------|-------|
| Name | `auditlog-read-credentials` |
| Namespace | `kyma-system` |
| Label | `operator.kyma-project.io/managed-by: infrastructure-manager` |
| Content | OAuth credentials (copied from source secret data) |

# Consequences

## Positive

1. **Complete feature**: Users can access dedicated audit logs with provided credentials
2. **Clean architecture**: Each state has single responsibility
3. **Correct ordering**: Credentials available only after shoot configuration
4. **Better observability**: Separate conditions for each operation
5. **Idempotent**: Server-side apply ensures safe retries
6. **Self-cleaning**: Namespace deletion handles cleanup
7. **Testable**: States can be tested independently

## Negative

1. **Additional FSM state**: Increases state count by one
2. **Extra reconciliation iteration**: One more state transition during provisioning

## Neutral

1. **Conditional execution**: State only executed when dedicated logging enabled
2. **KALM dependency**: Requires KALM to populate read credentials secret

# Implementation

See [Implementation Details](./005-copy-auditlog-read-credentials-implementation.md) for comprehensive implementation guidance including:

- AuditLogData structure extension
- FSM state implementation
- State transition flow diagram
- Condition types
- Error handling
- Idempotency guarantees
- Testing approach

# References

- [Issue #1496: Copy dedicated audit log read credentials](https://github.com/kyma-project/kyma-infrastructure-manager/issues/1496)
- [ADR 004: Dedicated Audit Logging](./004-dedicated-audit-logging.md)
- [ADR 004: Implementation Details](./004-dedicated-audit-logging-implementation.md)
- [Implementation Details](./005-copy-auditlog-read-credentials-implementation.md)
