# Design: Split `GetDedicatedAuditLogData` into Two Methods

## Problem

`GetDedicatedAuditLogData(ctx, runtimeID, claim bool)` is a boolean-trap: callers must pass a literal `true` or `false` with an explanatory comment, and the function body has two divergent code paths that share almost no logic. This makes the interface harder to read and the implementation harder to reason about.

## Solution

Split the single method into two named methods on the `DataProvider` interface.

### Interface change

**Remove:**
```go
GetDedicatedAuditLogData(ctx context.Context, runtimeID string, claim bool) (AuditLogData, error)
```

**Add:**
```go
// ClaimDedicatedAuditLogData performs Phase 2 of the two-phase claim: upgrades the reservation
// to a full claim by setting AssignedToRuntimeID, then returns the audit log configuration.
ClaimDedicatedAuditLogData(ctx context.Context, runtimeID string) (AuditLogData, error)

// GetDedicatedAuditLogData returns audit log configuration from an already claimed or reserved
// AuditLogCR. Read-only — does not modify the CR. Falls back from claim lookup to reservation
// lookup so it works in the window between Phase 1 and Phase 2.
GetDedicatedAuditLogData(ctx context.Context, runtimeID string) (AuditLogData, error)
```

### Implementation

`ClaimDedicatedAuditLogData` — contains the current `claim=true` path:
1. Find AuditLogCR by reservation label (`LabelReservedForRuntimeID`)
2. If not found, return error
3. If `Spec.AssignedToRuntimeID != runtimeID`, set it and call `client.Update` (idempotent)
4. Return `AuditLogData` from the CR

`GetDedicatedAuditLogData` — contains the current `claim=false` path:
1. Find AuditLogCR by `Spec.AssignedToRuntimeID`
2. If not found, fall back to find by reservation label
3. If still not found, return error
4. Return `AuditLogData` from the CR (no writes)

### Caller updates

| File | Before | After |
|------|--------|-------|
| `runtime_fsm_migrate_dedicated_auditlog.go:47` | `GetDedicatedAuditLogData(ctx, runtimeID, true)` | `ClaimDedicatedAuditLogData(ctx, runtimeID)` |
| `runtime_fsm_patch_shoot.go:45` | `GetDedicatedAuditLogData(ctx, runtimeID, false)` | `GetDedicatedAuditLogData(ctx, runtimeID)` |

### Files touched

- `pkg/auditlog/provider.go` — interface + implementation
- `pkg/auditlog/mocks/data_provider.go` — hand-written mock updated to match new interface
- `pkg/auditlog/provider_test.go` — test cases split across the two methods
- `internal/controller/runtime/fsm/runtime_fsm_migrate_dedicated_auditlog.go` — call site update
- `internal/controller/runtime/fsm/runtime_fsm_patch_shoot.go` — call site update
- `internal/controller/runtime/fsm/runtime_fsm_migrate_dedicated_auditlog_test.go` — test updates
- `internal/controller/runtime/fsm/runtime_fsm_patch_shoot_test.go` — test updates

## Out of scope

- Switching the mock to mockery generation
- Any changes to `ReserveAuditLog`, `GetSharedAuditLogData`, or `ReleaseDedicated`
- Changes to the two-phase claim logic itself
