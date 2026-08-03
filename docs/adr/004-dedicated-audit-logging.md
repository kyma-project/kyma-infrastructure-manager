# Context

This document defines the architecture for integrating dedicated BTP audit logging infrastructure with Kyma Runtime provisioning via the Kyma Audit Log Manager (KALM).

# Status

Proposed

# Background

Currently, Kyma Runtimes use a shared audit logging infrastructure where multiple runtimes share common audit log tenants, configured via a static mapping file that maps provider type and region to tenant IDs. This approach has limitations:

- **No self-service access**: Users cannot directly access their audit logs without SRE assistance
- **Shared infrastructure**: Multiple runtimes share the same audit logging tenant
- **Static configuration**: Audit log configuration is managed via static file updates

The Kyma Audit Log Manager (KALM) introduces a pool-based approach for provisioning dedicated BTP audit logging infrastructure per runtime. KALM runs as part of the Kyma Control Plane and manages the complete lifecycle of dedicated audit log stacks through the `AuditLog` custom resource.

## KALM Pool Architecture

KALM maintains a pool of pre-provisioned `AuditLog` CRs representing ready to use Audit Logging infrastructure. CRs become available for reservation once they reach the `RegistrationReady` or `SiemApproved` state. These CRs contain:
- BTP subaccount with audit log service provisioned
- Service credentials stored in Gardener secrets
- SIEM registration completed (for `SiemApproved`) or registration pending (for `RegistrationReady`)
- **Regions list**: `spec.regions` array containing Kyma Runtime hyperscaler regions this CR can serve (e.g., `["eu-central-1", "eu-west-2"]`)
- Ready to be assigned to a Kyma Runtime

Key AuditLog CR states:
- `Pending`: Initial state, BTP resources being provisioned
- `RegistrationReady`: BTP resources ready, awaiting SIEM registration — **available for reservation**
- `SiemApproved`: SIEM registration completed, in the pool — **available for reservation**
- `Assigned`: Claimed by a runtime, in use
- `Orphaned`: Runtime deleted, in retention period (default: 90 days)

# Decision

## Architectural Approach

We implement a **validation-first, migrate-last provisioning model** where dedicated audit logging availability is validated before shoot creation, the shoot is created with shared audit logging, the entire provisioning completes successfully, and only then is the runtime migrated to dedicated logging as the final step.

### FSM State Flow

```
sFnCreateShoot (validates dedicated config availability if requested)
    ↓
sFnWaitForShootCreation
    ↓
sFnHandleKubeconfig
    ↓
sFnCreateKymaNamespace
    ↓
sFnInitializeRuntimeBootstrapper
    ↓
sFnCleanupRegistryCacheGardenSecrets
    ↓
sFnConfigureSKR
    ↓
sFnApplyClusterRoleBindings
    ↓
sFnMigrateToDedicatedAuditLog  (new state - absolute final step)
    ↓
Complete (updateStatusAndStop)
```

**Note**: After `sFnMigrateToDedicatedAuditLog` patches the shoot with dedicated config, it requeues the reconciliation. On the next reconciliation, the state will have no effect (because the shoot already has dedicated logging configured), and provisioning completes successfully.

### Phase 1: Validate and Create Shoot with Shared Audit Logging

The `sFnCreateShoot` state first validates that dedicated audit logging configuration is available (if requested), then creates the Gardener shoot using the existing shared audit log configuration from the static mapping file.

**Validation**: If the runtime requests dedicated audit logging (`auditLogAccessEnabled: true`) and the global feature flag is enabled, the state checks that a dedicated AuditLogCR is available in the pool. If not available, provisioning fails immediately.

**Rationale for validation-first approach**:
- Fail fast: Don't waste time and resources provisioning a runtime if dedicated logging cannot be provided as requested
- Clear error: User gets immediate feedback that their request cannot be fulfilled
- No orphaned resources: Shoot is never created if dedicated logging requirement cannot be met

**Rationale for using shared logging initially**:
- Allows shoot to be created and tested before claiming expensive dedicated resources
- If the shoot fails for other reasons (quota, config, infrastructure), no dedicated resources are wasted

### Phase 2: Complete Provisioning

The runtime goes through the full provisioning flow:
- `sFnWaitForShootCreation` - Wait for shoot to be ready
- `sFnHandleKubeconfig` - Handle kubeconfig
- `sFnCreateKymaNamespace` - Create Kyma namespace
- `sFnInitializeRuntimeBootstrapper` - Initialize bootstrapper
- `sFnCleanupRegistryCacheGardenSecrets` - Cleanup secrets
- `sFnConfigureSKR` - Configure SKR
- `sFnApplyClusterRoleBindings` - Apply cluster role bindings

All these steps complete successfully before any dedicated audit logging resources are claimed.

### Phase 3: Migrate to Dedicated Audit Logging (Final Step)

After **all provisioning is complete**, the `sFnMigrateToDedicatedAuditLog` state executes as the absolute final step. This state claims the reserved resource and patches the shoot:

**Step 1: Get Audit Log Data and Claim** (atomic operation)
- Calls `GetDedicatedAuditLogData(ctx, runtimeID, claim=true)`
- Finds the reserved AuditLogCR by label
- Upgrades reservation to claim by setting `AssignedToRuntimeID`
- Returns audit log configuration
- Fails provisioning immediately if claim fails (no fallback)

**Step 2: Get Current Shoot Configuration** (read-only)
- Extracts current audit log config from shoot's extension
- Uses helper function `getShootAuditLogConfig(shoot)`

**Step 3: Compare Configurations**
- Uses `auditLogConfigsEqual()` to compare current and desired
- Compares TenantID, ServiceURL, and SecretName

**Step 4: Conditional Patch** (only if configurations differ)
- If configs are equal: Skip patch, complete provisioning immediately (Step 5a)
- If configs differ: Patch shoot with dedicated audit log configuration (Step 5b)
- Uses `patchShootAuditLog()` helper function
- If patch fails: Requeue (claim persists, retry patch on next reconciliation)
- If patch succeeds: Requeue to allow Gardener to reconcile the shoot changes

**Step 5a: Complete Provisioning** (when configs are equal)
- Sets `ConditionTypeCustomAuditlogConfigured` to Ready status
- Sets `ProvisioningCompleted` status
- Returns `updateStatusAndStop()`

**Step 5b: Requeue After Patch** (when configs differ and patch succeeds)
- Sets `ConditionTypeCustomAuditlogConfigured` to Unknown status
- Returns `updateStatusAndRequeueAfter(m.GardenerRequeueDuration)`
- On next reconciliation: configs will be equal → proceeds to Step 5a

**Condition Types Used**:
- `ConditionTypeCustomAuditlogConfigured` - Dedicated condition for audit log status
- `ConditionReasonCustomAuditLogError` - When claim or config retrieval fails
- `ConditionReasonCustomAuditLogConfigured` - When successfully configured

**Key Properties**:
- **No fallback**: If dedicated config cannot be obtained, provisioning fails explicitly
- **Claim first**: Resource is claimed before any patch attempt
- **Efficient**: Skips unnecessary patches when configs already match
- **Idempotent at every step**: Safe to retry from any failure point
- **Clean completion**: No unnecessary requeues after success

## Rejected Alternatives

### Alternative 1: Single-Phase Claim (Validate Only, No Reservation)

**Approach**: Validate availability in `sFnCreateShoot`, claim in `sFnMigrateToDedicatedAuditLog` without reservation

**Rejected because**:
- Race condition: Between validation and claim (~5-10 minutes), another runtime could take the resource
- User gets false positive: Validation succeeds but migration fails
- Poor user experience: Provisioning fails at the very end after all resources created
- Wastes shoot resources: Entire shoot is created then has to be torn down

### Alternative 2: Claim Before Shoot Creation (Heavy Lock First)

**Approach**: Claim AuditLogCR → Create shoot with dedicated config

**Rejected because**:
- If shoot creation fails (quota exceeded, invalid config), the claimed AuditLogCR is wasted
- The AuditLogCR remains locked to a non-existent runtime
- Pool resources are depleted unnecessarily
- Requires complex rollback logic

### Alternative 3: Claim and Create Atomically

**Approach**: Claim AuditLogCR and create shoot in a single transaction, with rollback on failure

**Rejected because**:
- Kubernetes doesn't support cross-resource transactions
- Rollback logic is complex and error-prone
- Shoot creation can fail at various stages (Gardener API, infrastructure provider)
- Still risks wasting resources during transient failures

### Alternative 4: Lock-Based Claiming

**Approach**: Add lock fields to AuditLog CR spec for claiming

**Rejected because**:
- Adds unnecessary complexity to the CRD
- Kubernetes optimistic concurrency (resourceVersion) already prevents double-claiming
- Locks can become stale if client crashes
- Requires lock cleanup/timeout logic

# Consequences

## Positive

1. **No wasted resources**: Two-phase approach (light lock → heavy lock) ensures minimal waste
   - Phase 1 reserves with labels (cheap, easy to clean up)
   - Phase 2 claims only after full provisioning succeeds
   - Failed provisions only leave behind labels, not fully-assigned resources

2. **Solves race condition**: Reservation labels prevent the window between validation and claiming
   - Validate + reserve happens atomically in Phase 1
   - CR is guaranteed available for this runtime during entire provisioning

3. **Fail-fast**: User gets immediate feedback if dedicated logging unavailable
   - No time or resources wasted on provisioning that will fail anyway
   - Clear error message guides user to retry when pool has capacity

4. **Idempotent operations**: Safe retries throughout the flow
   - Reservation lookup before creating new reservation
   - Claim lookup before creating new claim
   - Recovery from interruptions at any stage

5. **Manual cleanup option**: Operators have simple path to recover stale reservations
   - No complex state to understand
   - Labels visible with standard kubectl commands
   - Simple label removal returns CR to pool

6. **Concurrent-safe**: Kubernetes optimistic concurrency prevents conflicts
   - Multiple runtimes can provision simultaneously
   - Conflicts automatically trigger retries with different CRs

7. **Observable**: Easy to monitor pool state
   - Available: No labels, no assignment
   - Reserved: Has labels, no assignment
   - Claimed: Has labels, has assignment
   - Query with simple label selectors

8. **No CRD changes**: Label-based approach doesn't require KALM modifications
   - Non-intrusive to KALM's state machine
   - Can be implemented immediately

## Negative

1. **Two-phase complexity**: More complex than single-phase claim
   - Requires understanding of reservation vs claim semantics
   - Two different code paths (reserve in create, claim in migrate)
   - More states to test and document

2. **Manual cleanup required (MVP)**: Stale reservations need operator intervention
   - Automated cleanup can be added later
   - Operators need to monitor for reservations without assignments
   - Process must be documented for on-call

3. **Brief shared logging period**: Runtime uses shared logging during provisioning (~5-15 minutes)
   - Audit logs split between shared and dedicated during this window
   - Not a security issue but may complicate log analysis

4. **Vendored CRD maintenance**: Must sync types when KALM updates AuditLog CRD
   - Creates dependency on KALM release cycle
   - Breaking changes in KALM could require code updates

5. **Migration state complexity**: Additional FSM state increases controller logic
   - More code paths to test
   - Conditional state transitions based on flags

6. **Label namespace coordination**: Need to coordinate label names with KALM team
   - Risk of conflicts if KALM uses same label names
   - Requires documentation and communication

## Neutral

1. **Two-level feature flags**: Both global and per-runtime flags required
   - Provides flexibility for gradual rollout
   - But requires both to be set for feature to work

2. **KALM dependency**: Requires KALM to be installed and maintaining pool
   - Strong coupling between two components
   - Provisioning fails if KALM unavailable (when dedicated requested)

3. **Eventual consistency**: Migration happens asynchronously after provisioning
   - Standard Kubernetes reconciliation pattern
   - Brief window where shoot exists but isn't fully configured

4. **Reservation timeout**: 1-hour timeout is arbitrary
   - Too short: Legitimate slow provisions get cleaned up
   - Too long: Pool exhaustion from stale reservations
   - May need tuning based on real-world data

# Implementation

See [Implementation Details](./004-dedicated-audit-logging-implementation.md) for comprehensive implementation guidance including:

- AuditLog Data Provider interface
- Two-phase reservation algorithm (detailed pseudo-code)
- Migration state implementation
- Configuration changes
- Vendored AuditLog CRD structure
- Error handling and edge cases
- Monitoring and observability metrics

# References

- [Kyma Audit Log Manager Repository](https://github.tools.sap/kyma/kyma-auditlog-manager)
- [KALM Architecture Documentation](https://github.tools.sap/kyma/kyma-auditlog-manager/docs/contributor/architecture)
- [Audit Log Package README](../../pkg/auditlog/README.md)
- [Infrastructure Manager Provisioning ADR](./001-provisioning.md)
- [Implementation Details](./004-dedicated-audit-logging-implementation.md)
