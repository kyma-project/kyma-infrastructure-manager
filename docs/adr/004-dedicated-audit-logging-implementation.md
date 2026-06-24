# Dedicated Audit Logging - Implementation Details

This document provides detailed implementation guidance for the dedicated audit logging feature described in [ADR 004: Dedicated Audit Logging](./004-dedicated-audit-logging.md).

## Table of Contents

- [AuditLog Data Provider](#auditlog-data-provider)
- [Claiming Algorithm: Two-Phase Reservation](#claiming-algorithm-two-phase-reservation)
- [Migration State Implementation](#migration-state-implementation)
- [Cleanup on Runtime Deletion](#cleanup-on-runtime-deletion)
- [Configuration Changes](#configuration-changes)
- [Vendored AuditLog CRD](#vendored-auditlog-crd)
- [Error Handling and Edge Cases](#error-handling-and-edge-cases)
- [Monitoring and Observability](#monitoring-and-observability)

## AuditLog Data Provider

All audit log operations are abstracted behind the `auditlog.DataProvider` interface:

```go
type DataProvider interface {
    // ReserveAuditLog performs Phase 1 of the two-phase claim: reserves an AuditLogCR by adding labels
    // This should be called before shoot creation to ensure a resource is available
    // The providerRegion is the hyperscaler region (e.g., "eu-central-1") that must match
    // one of the AuditLogCR's spec.regions entries
    // Returns error if no available AuditLogCR is found for the given region
    ReserveAuditLog(ctx context.Context, providerRegion string, runtimeID string) error

    // GetDedicatedAuditLogData returns audit log configuration from AuditLogCR
    // When claim=true, performs Phase 2: upgrades reservation to full claim (sets assignedToRuntimeID)
    // When claim=false, only retrieves data from already claimed/reserved resource
    GetDedicatedAuditLogData(ctx context.Context, runtimeID string, claim bool) (AuditLogData, error)

    // GetSharedAuditLogData returns audit log configuration from shared configuration file
    GetSharedAuditLogData(ctx context.Context, providerType, region string) (AuditLogData, error)

    // ReleaseDedicated releases the claimed AuditLogCR (marks as orphaned)
    ReleaseDedicated(ctx context.Context, runtimeID string) error
}
```

**Benefits**:
- Clear separation between shared and dedicated audit log sources
- Two-phase claim with explicit `ReserveAuditLog` and `GetDedicatedAuditLogData(claim=true)`
- Feature flag handling at FSM level, not in provider
- Easy to mock for testing
- No implicit fallback - dedicated requests fail explicitly if unavailable

## Claiming Algorithm: Two-Phase Reservation

The claiming algorithm uses a **two-phase approach** to solve the race condition between validation (pre-creation) and actual claiming (post-provisioning):

### Problem Statement

Between the time we validate that an AuditLogCR is available (in `sFnCreateShoot`) and when we actually claim it (in `sFnMigrateToDedicatedAuditLog` ~5-10 minutes later), another runtime could claim that resource. This would cause the migration to fail even though validation passed.

### Solution: Label-Based Reservation (Light Lock)

We use Kubernetes labels as a "light lock" mechanism to reserve an AuditLogCR during the provisioning window:

#### Phase 1: Reserve (in `sFnCreateShoot`)

```go
func reserveAuditLogCR(ctx, runtimeID, region string) error {
    // 1. Check if we already have a reservation
    reserved := findAuditLogCRByLabel(ctx, "reserved-for-runtime-id", runtimeID)
    if reserved != nil {
        return nil // Already reserved for us
    }
    
    // 2. Find available CR matching the region
    // CR must be: state = RegistrationReady or SiemApproved, assignedToRuntimeID = "", 
    // no reservation label, AND region must be in CR's spec.regions array
    available := findAvailableAuditLogCR(ctx, region)
    if available == nil {
        return fmt.Errorf("no available AuditLogCR in the pool for region %s", region)
    }
    
    // 3. Add reservation labels (light lock)
    if available.Labels == nil {
        available.Labels = make(map[string]string)
    }
    available.Labels["reserved-for-runtime-id"] = runtimeID
    available.Labels["reserved-for-runtime-at"] = fmt.Sprintf("%d", time.Now().UTC().Unix())
    
    // 4. Update with optimistic concurrency
    err := client.Update(ctx, available)
    if IsConflict(err) {
        // Another runtime reserved it concurrently, caller should retry
        return fmt.Errorf("conflict reserving AuditLogCR: another runtime reserved it concurrently")
    }
    
    return nil
}

// findAvailableAuditLogCR finds an available AuditLogCR that serves the given region
func findAvailableAuditLogCR(ctx, region string) *AuditLog {
    list := client.List(ctx, &AuditLogList{})
    
    for _, cr := range list.Items {
        // Check state (RegistrationReady or SiemApproved) and not assigned
        if (cr.Status.State != StateRegistrationReady && cr.Status.State != StateSiemApproved) ||
            cr.Spec.AssignedToRuntimeID != "" {
            continue
        }
        
        // Check for existing reservation
        if _, hasReservation := cr.Labels["reserved-for-runtime-id"]; hasReservation {
            continue
        }
        
        // Check if region matches one of CR's regions
        if !containsRegion(cr.Spec.Regions, region) {
            continue
        }
        
        return &cr
    }
    return nil
}

func containsRegion(regions []string, region string) bool {
    for _, r := range regions {
        if r == region {
            return true
        }
    }
    return false
}
```

#### Phase 2: Claim (in `sFnMigrateToDedicatedAuditLog`)

```go
func claimReservedAuditLogCR(ctx, runtimeID) (*AuditLog, error) {
    // 1. Find our reserved CR
    reserved := findAuditLogCRByLabel(ctx, "reserved-for-runtime-id", runtimeID)
    if reserved == nil {
        return nil, ErrReservationNotFound
    }
    
    // 2. Check if already claimed (idempotent)
    if reserved.Spec.AssignedToRuntimeID == runtimeID {
        return reserved, nil // Already claimed by us
    }
    
    // 3. Upgrade reservation to full claim (heavy lock)
    reserved.Spec.AssignedToRuntimeID = runtimeID
    
    // 4. Update with optimistic concurrency
    err := client.Update(ctx, reserved)
    if IsConflict(err) {
        // Unlikely but possible, retry
        return nil, ErrConflictClaiming
    }
    
    return reserved, nil
}
```

### Reservation Labels

Two labels are added to the AuditLogCR during reservation:

- **`reserved-for-runtime-id`**: The Runtime CR name (e.g., `"1234-5678-90ab-cdef"`)
  - Used to find the reserved resource in Phase 2
  - Identifies which runtime has reserved this CR
  
- **`reserved-for-runtime-at`**: Unix timestamp (e.g., `"1719238800"`)
  - Records when the reservation was made (as seconds since Unix epoch)
  - Enables detection of stale reservations
  - Can be used for automated cleanup of abandoned reservations
  - **Important**: Must be numeric only to comply with Kubernetes label validation rules

### Resource States

An AuditLogCR can be in one of these states:

1. **Available**: `state=RegistrationReady` or `state=SiemApproved`, `assignedToRuntimeID=""`, no reservation labels, serves one or more regions via `spec.regions`
2. **Reserved (Light Lock)**: Has reservation labels, `assignedToRuntimeID=""`
3. **Claimed (Heavy Lock)**: Has reservation labels, `assignedToRuntimeID=<runtimeID>`
4. **Orphaned**: `spec.orphaned=true`, retention period active

**Region Matching**: During reservation, the runtime's hyperscaler region (e.g., `eu-central-1`) must match at least one entry in the AuditLogCR's `spec.regions` array. An AuditLogCR can serve multiple regions.

**Note on RegistrationReady**: CRs in `RegistrationReady` state have all BTP resources provisioned and credentials available, but are awaiting SIEM team approval. Allowing reservation of these CRs enables faster provisioning since the SIEM approval process can complete in parallel with shoot provisioning.

### State Transitions

```
Available → Reserved (by sFnCreateShoot adding labels)
    ↓
Reserved → Claimed (by sFnMigrateToDedicatedAuditLog setting assignedToRuntimeID)
    ↓
Claimed → Orphaned (by sFnDeleteShoot setting orphaned=true)
    ↓
Orphaned → Cleaned up (by KALM after retention period)

Special case:
Reserved → Available (manual label removal if provisioning abandoned)
```

### Cleanup of Abandoned Reservations

**Manual Cleanup (Recommended for MVP)**:
- Operators can manually remove reservation labels from AuditLogCRs that are reserved but not claimed
- Query: `kubectl get auditlog -l reserved-for-runtime-id,!assignedToRuntimeID`
- This returns resources that are reserved (have label) but never claimed (no assignment)
- Safe to remove labels and return to pool:
  ```bash
  kubectl label auditlog <name> reserved-for-runtime-id- reserved-for-runtime-at-
  ```

**Automated Cleanup (Future Enhancement)**:
A KALM controller or separate cleanup job could automate this:

```go
// Pseudo-code for automated cleanup
func cleanupStaleReservations(ctx) {
    // Find reserved but not claimed CRs
    // Note: Cannot use client.MatchingFields for spec fields (not indexed by default)
    // Must filter in loop instead
    list := client.List(ctx, &AuditLogList{}, 
        client.MatchingLabels{"reserved-for-runtime-id": "*"})
    
    for _, cr := range list.Items {
        // Skip CRs that are already claimed (heavy lock applied)
        if cr.Spec.AssignedToRuntimeID != "" {
            continue
        }
        
        // Parse Unix timestamp from label
        reservedAtUnix, err := strconv.ParseInt(cr.Labels["reserved-for-runtime-at"], 10, 64)
        if err != nil {
            continue
        }
        reservedAt := time.Unix(reservedAtUnix, 0)
        
        // If reserved for more than 1 hour without claim, release it
        if time.Since(reservedAt) > 1*time.Hour {
            delete(cr.Labels, "reserved-for-runtime-id")
            delete(cr.Labels, "reserved-for-runtime-at")
            client.Update(ctx, &cr)
            log.Info("Released stale reservation", "auditlog", cr.Name, "reservedFor", 1*time.Hour)
        }
    }
}
```

**Cleanup Parameters**:
- **Timeout threshold**: 1 hour (configurable)
  - Normal provisioning takes 5-15 minutes
  - 1 hour provides generous buffer for retries and delays
  - Prevents indefinite pool exhaustion from failed provisions
- **Cleanup frequency**: Every 30 minutes
- **Safety**: Only removes labels from CRs with `assignedToRuntimeID=""` (never disrupts claimed resources)

### Why Labels Instead of Spec Fields?

1. **No CRD changes required**: Labels can be added without modifying KALM's AuditLog CRD
2. **Kubernetes-native**: Label selectors are efficient and well-supported
3. **Non-intrusive**: Labels don't affect KALM's state machine or business logic
4. **Easy cleanup**: Labels can be removed without validation or reconciliation
5. **Observable**: `kubectl get auditlog -l reserved-for-runtime-id` shows all reservations

### Coordination with KALM

KALM should be updated to respect reservations:

1. **Pool Management**: When counting available AuditLogCRs, exclude those with reservation labels:
   ```go
   availableCount := count(state=SiemApproved AND assignedToRuntimeID="" AND no reservation labels)
   ```

2. **State Transitions**: KALM's state machine should ignore reservation labels
   - Labels don't affect `SiemApproved` → `Assigned` transition
   - Only `assignedToRuntimeID` field triggers state change

3. **Cleanup (Optional)**: KALM could run the automated cleanup job described above

### Key Properties

- **Solves race condition**: Reservation prevents other runtimes from selecting the same CR during provisioning
- **Idempotent**: Both reserve and claim operations can be safely retried
- **Concurrent-safe**: Kubernetes optimistic concurrency prevents double-reservation
- **Fail-fast**: Validation + reservation happens before expensive shoot creation
- **Minimal waste**: If provisioning fails, only a label needs cleanup (not a fully assigned resource)
- **Observable**: Easy to query reserved vs available resources
- **Manual override**: Operators can manually release stale reservations if needed

### Error Scenarios

| Scenario | Handling                                                                                                                        |
|----------|---------------------------------------------------------------------------------------------------------------------------------|
| No available CR for the region | Fail provisioning immediately in `sFnCreateShoot` with error: "no available AuditLogCR in the pool for region X"                |
| Conflict during reservation | Retry with different available CR from pool                                                                                     |
| Reservation not found in Phase 2 | Should never happen if Phase 1 succeeded; fail provisioning with clear error; The `status.state` of Runtime CR set to Failed |
| Conflict during claim | Retry claim operation                                                                                                           |
| Provisioning fails after reservation | Label remains; cleaned up manually or by automated job                                                                          |
| Runtime deleted before migration | Label remains; cleaned up manually or by automated job                                                                          |

## Migration State Implementation

```go
func sFnMigrateToDedicatedAuditLog(ctx context.Context, m *fsm, s *systemState) (stateFn, *ctrl.Result, error) {
    // Check if global feature flag is enabled
    if !m.DedicatedAuditLoggingEnabled {
        if !s.instance.IsProvisioningCompletedStatusSet() {
            s.instance.UpdateStateProvisioningCompleted()
        }
        return updateStatusAndStop()
    }

    // Check if runtime-specific audit log access is enabled
    if s.instance.Spec.AuditLogAccessEnabled == nil || !*s.instance.Spec.AuditLogAccessEnabled {
        if !s.instance.IsProvisioningCompletedStatusSet() {
            s.instance.UpdateStateProvisioningCompleted()
        }
        return updateStatusAndStop()
    }

    // Step 1: Get desired audit log data and claim the resource
    // This performs Phase 2 of the two-phase claim (upgrade from light lock to heavy lock)
    auditLogData, err := m.AuditLogDataProvider.GetDedicatedAuditLogData(
        ctx,
        s.instance.GetName(),
        true, // claim=true to upgrade reservation to full claim
    )
    if err != nil {
        // Claim failed - fail provisioning
        msg := fmt.Sprintf("Failed to get and claim dedicated audit log configuration: %v", err)
        s.instance.UpdateStateFailed(
            imv1.ConditionTypeCustomAuditlogConfigured,
            imv1.ConditionReasonCustomAuditLogError,
            msg,
        )
        m.Metrics.IncRuntimeFSMStopCounter()
        return updateStatusAndStop()
    }

    // Step 2: Get current shoot audit log config (read-only)
    shootAuditLogData, err := getShootAuditLogConfig(s.shoot)
    if err != nil {
        // If we can't get current config, assume we need to patch
        shootAuditLogData = nil
    }

    // Step 3: Compare configurations
    configsEqual := shootAuditLogData != nil && auditLogConfigsEqual(shootAuditLogData, auditLogData)

    // Step 4: Patch only if configurations differ
    if configsEqual {
        // Complete provisioning - configs already match
        s.instance.UpdateStateReady(
            imv1.ConditionTypeCustomAuditLogConfigured,
            imv1.ConditionReasonCustomAuditLogConfigured,
            "Custom AuditLog shoot configuration completed",
        )
        
        if !s.instance.IsProvisioningCompletedStatusSet() {
            s.instance.UpdateStateProvisioningCompleted()
        }
        return updateStatusAndStop()
    }

    // Step 5: PATCH shoot with dedicated config
    if err := patchShootAuditLog(ctx, m, s, auditLogData); err != nil {
        // AuditLogCR is claimed, we'll retry the patch on next reconciliation
        s.instance.UpdateStatePending(
            imv1.ConditionTypeCustomAuditLogConfigured,
            imv1.ConditionReasonCustomAuditLogConfigured,
            metav1.ConditionFalse,
            "Custom AuditLog shoot configuration could not be patched, will retry",
        )
        return updateStatusAndRequeueAfter(m.GardenerRequeueDuration)
    }

    // Patch successful, requeue to wait for Gardener to reconcile
    s.instance.UpdateStatePending(
        imv1.ConditionTypeCustomAuditLogConfigured,
        imv1.ConditionReasonCustomAuditLogConfigured,
        metav1.ConditionUnknown,
        "Custom AuditLog shoot configuration completed",
    )
    return updateStatusAndRequeueAfter(m.GardenerRequeueDuration)
}
```

**Key Properties**:
- **Claim first**: `GetDedicatedAuditLogData(claim=true)` immediately claims the reserved resource
- **Compare then patch**: Only patches shoot if configuration actually differs
- **Uses dedicated condition type**: `ConditionTypeCustomAuditlogConfigured` for audit log specific status
- **No fallback**: If dedicated config cannot be obtained, provisioning fails (not falls back to shared)
- **Idempotent**: Safe to retry - claim is idempotent, patch is idempotent
- **Two-reconciliation completion**: After successful patch, requeues with `ConditionUnknown`; next reconciliation detects equal configs and completes with `updateStatusAndStop()`

## Cleanup on Runtime Deletion

When a runtime is deleted, the claimed AuditLogCR must be released:

```go
func sFnDeleteShoot(ctx, m *fsm, s *systemState) (stateFn, *ctrl.Result, error) {
    // Release the claimed AuditLogCR if we have one
    if m.RCCfg.DedicatedAuditLoggingEnabled {
        err := m.AuditLogDataProvider.ReleaseDedicated(ctx, s.instance.GetName())
        if err != nil {
            m.log.Error(err, "Failed to release dedicated audit log")
            // Continue with shoot deletion anyway
        }
    }
    
    // ... continue with shoot deletion ...
}
```

**ReleaseDedicated** marks the AuditLogCR as orphaned by setting `Spec.Orphaned = true`. KALM then:
1. Transitions the CR to `Orphaned` state
2. Maintains the audit logs for the retention period (default: 90 days)
3. Cleans up BTP resources after retention period expires

## Configuration Changes

### Two-Level Flag System

Dedicated audit logging requires **two flags** to be enabled:

1. **Global Feature Flag** (Infrastructure Manager level)
2. **Runtime-Specific Flag** (Runtime CR level)

This two-level approach allows:
- Global control over the feature availability
- Per-runtime opt-in for audit log access
- Gradual rollout to specific customers

#### Global Feature Flag

A new command-line flag controls the feature at the infrastructure manager level:

```go
flag.BoolVar(&dedicatedAuditLoggingEnabled, 
    "dedicated-audit-logging-enabled", 
    false, 
    "Feature flag to enable dedicated BTP audit logging infrastructure for provisioned Kyma Runtime")
```

When `false` (default), all runtimes use shared audit logging regardless of their individual settings.

#### Runtime-Specific Flag

Each Runtime CR can opt-in to dedicated audit logging:

```yaml
apiVersion: infrastructuremanager.kyma-project.io/v1
kind: Runtime
metadata:
  name: my-runtime
spec:
  auditLogAccessEnabled: true  # Request dedicated audit logging for this runtime
  shoot:
    provider:
      type: aws
      region: us-east-1
    # ...
```

The `spec.auditLogAccessEnabled` field is optional (pointer to bool):
- `true` - Request dedicated audit logging (if global flag also enabled and AuditLogCR available)
- `false` or `nil` (default) - Use shared audit logging

#### Decision Matrix

| Global Flag | Runtime Flag | Result |
|-------------|--------------|---------|
| `false` | `true` | Shared logging (global flag takes precedence) |
| `false` | `false`/`nil` | Shared logging |
| `true` | `true` | Dedicated logging (if available) |
| `true` | `false`/`nil` | Shared logging (runtime didn't opt-in) |

### FSM Configuration

```go
type RCCfg struct {
    // ... existing fields ...
    DedicatedAuditLoggingEnabled bool
    AuditLogDataProvider         auditlog.DataProvider
}
```

The `AuditLogDataProvider` replaces the direct use of `auditlogs.Configuration` map.

## Vendored AuditLog CRD

Since infrastructure-manager is in public GitHub and kyma-auditlog-manager is in internal SAP GitHub, the AuditLog CRD types are vendored:

```
pkg/auditlog/
└── v1beta1/
    ├── auditlog_types.go         # Vendored from KALM
    ├── zz_generated.deepcopy.go  # Vendored from KALM
    └── groupversion_info.go      # API metadata
```

**Key AuditLogCR fields used by KIM**:
- `spec.regions []string` - Hyperscaler regions this CR can serve (e.g., `["eu-central-1", "eu-west-2"]`)
- `spec.assignedToRuntimeID string` - Runtime ID when claimed (heavy lock)
- `spec.config.serviceURL string` - Audit log service URL
- `spec.config.gardenerSecretName string` - Secret name in Gardener
- `spec.subaccountID string` - Used as tenant ID
- `spec.orphaned bool` - Set to true when runtime is deleted
- `status.state string` - Current state (SiemApproved, Assigned, etc.)

**Scheme Registration**: The vendored types must be registered with the controller-runtime scheme in `cmd/main.go`:

```go
import auditlogv1 "github.com/kyma-project/infrastructure-manager/pkg/auditlog/v1beta1"

func init() {
    utilruntime.Must(clientgoscheme.AddToScheme(scheme))
    utilruntime.Must(infrastructuremanagerv1.AddToScheme(scheme))
    utilruntime.Must(rbacv1.AddToScheme(scheme))
    utilruntime.Must(auditlogv1.AddToScheme(scheme))  // Required for AuditLog CR operations
}
```

**Maintenance**: When KALM updates the AuditLog CRD, these files must be re-synced.

## Error Handling and Edge Cases

### No Available AuditLogCR (Phase 1 - Reservation)

**Scenario**: Pool is exhausted or no `SiemApproved` CRs serve the requested region

**Handling**: 
- Fail provisioning immediately in `sFnCreateShoot`
- User gets clear error: "no available AuditLogCR in the pool for region X"
- No shoot resources created or wasted
- User can retry later when pool has capacity for their region

### Concurrent Reservations (Phase 1)

**Scenario**: Two runtimes try to reserve the same AuditLogCR simultaneously

**Handling**:
- Kubernetes resourceVersion causes conflict error for second runtime
- Second runtime retries with different available AuditLogCR from pool
- Eventually both runtimes get unique reservations
- No double-booking possible

### Reservation Not Found (Phase 2 - Claim)

**Scenario**: In `sFnMigrateToDedicatedAuditLog`, the reserved CR cannot be found

**Handling**:
- Should never happen if Phase 1 succeeded and runtime completed provisioning
- If it happens: Fail provisioning with clear error
- Possible causes: Manual label removal, KALM bug, race condition
- Operator can investigate and either restore label or retry provisioning

### Concurrent Claims (Phase 2)

**Scenario**: Conflict during claim operation (unlikely since CR is already reserved)

**Handling**:
- Requeue and retry claim operation
- Reserved CR won't be taken by another runtime (has our label)
- Retry will succeed on next reconciliation

### Shoot Update Failure (Phase 2)

**Scenario**: Shoot patch operation fails after claiming AuditLogCR

**Handling**:
- AuditLogCR remains claimed with RuntimeID
- Next reconciliation finds existing claim (idempotent)
- Retries shoot update
- No duplicate claims, no resource waste

### Provisioning Fails After Reservation (Phase 1 Complete, Before Phase 2)

**Scenario**: Runtime provisioning fails (e.g., kubeconfig issues, namespace creation) after reservation but before claim

**Handling**:
- Reserved AuditLogCR has label but no `assignedToRuntimeID`
- Label remains on CR, marking it as "reserved but unused"
- **Manual cleanup**: Operator removes labels to return CR to pool
- **Automated cleanup** (optional): Job removes labels after 1 hour timeout
- CR safely returns to available pool

### Reconciliation Interrupted

**Scenario**: Controller crashes between reservation and claim

**Handling**:
- **After Phase 1, before shoot creation**: Next reconciliation finds existing reservation, continues
- **After Phase 2, before shoot patch**: Next reconciliation finds existing claim, retries patch
- Idempotent operations ensure correctness throughout

### KALM Unavailable

**Scenario**: KALM controller is down or CRD not installed

**Handling**:
- List/Get operations fail in `sFnCreateShoot`
- If user requested dedicated logging: Provisioning fails with clear error
- If user didn't request dedicated: Uses shared logging, unaffected

## Logging

Log events:
- Reservation success/failure with RuntimeID
- Claim success/failure with RuntimeID
- Migration start/complete
- Stale reservation detected (if automated cleanup implemented)
- Release on runtime deletion
