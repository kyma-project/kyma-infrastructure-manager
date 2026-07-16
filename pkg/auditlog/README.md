# Audit Log Package

This package provides an abstraction layer for audit logging configuration in the Kyma Infrastructure Manager.

## Overview

The `auditlog` package encapsulates the complexity of managing audit log configurations from multiple sources:
- **Shared configuration**: Traditional file-based mapping (tenant ID mapped by provider/region)
- **Dedicated configuration**: Pool-based BTP audit logging infrastructure via AuditLog custom resources

## Package Structure

```
pkg/auditlog/
├── provider.go               # DataProvider interface and implementation
├── types.go                  # Common types (AuditLogData, Configuration)
├── types_test.go             # Tests for shared configuration
├── dedicated_client.go       # AuditLogCR operations (claim, release, find)
├── README.md                 # This file
└── v1beta1/                  # Vendored AuditLog CRD
    ├── groupversion_info.go      # Kubernetes API group/version info
    ├── auditlog_types.go         # AuditLog CR type definitions
    └── zz_generated.deepcopy.go  # Generated DeepCopy methods
```

## Vendored AuditLog CRD

The `v1beta1` sub-package contains vendored copies of the AuditLog custom resource definition from `github.tools.sap/kyma/kyma-auditlog-manager`. This is necessary because:

1. The infrastructure-manager is in public GitHub (github.com/kyma-project)
2. The kyma-auditlog-manager is in internal SAP GitHub (github.tools.sap)
3. We cannot add a dependency on an internal repository in a public project

The types are kept in `pkg/` rather than `internal/` to allow external projects to import both the provider interface and CR types from a single package location.

**Important**: When the AuditLog CRD is updated in kyma-auditlog-manager, these files should be re-synced.

## Usage

### Creating a DataProvider

```go
import (
    "github.com/kyma-project/infrastructure-manager/pkg/auditlog"
)

// Load shared configuration from file
sharedConfig, err := loadAuditLogDataMap(configPath)

// Create provider
provider := auditlog.NewDataProvider(
    kubeClient,
    sharedConfig,
    logger,
    namespace, // e.g., "kcp-system"
)
```

### Getting Audit Log Data

The DataProvider interface provides methods for both shared and dedicated audit logging:

```go
// Get shared audit log data from configuration file
sharedData, err := provider.GetSharedAuditLogData(
    ctx,
    providerType, // e.g., "aws", "azure", "gcp"
    region,       // e.g., "us-east-1", "westeurope"
)

// Get dedicated audit log data from AuditLogCR
// When claim=true, performs Phase 2 of two-phase claim (upgrades reservation to full claim)
// When claim=false, only retrieves data from already claimed/reserved resource
dedicatedData, err := provider.GetDedicatedAuditLogData(
    ctx,
    runtimeID,
    claim, // true to upgrade reservation to claim, false to just retrieve
)
```

### Claiming Process

Dedicated audit logging supports two approaches depending on the scenario:

#### Two-Phase Process (New Runtime Creation)

**Phase 1: Reserve** (during shoot creation in `sFnCreateShoot`)
```go
// Reserve an AuditLogCR by adding reservation labels (light lock)
err := provider.ReserveAuditLog(ctx, providerRegion, runtimeID)
```

**Phase 2: Claim** (after successful provisioning in `sFnMigrateToDedicatedAuditLog`)
```go
// Upgrade reservation to full claim by setting assignedToRuntimeID (heavy lock)
data, err := provider.GetDedicatedAuditLogData(ctx, runtimeID, true)
```

#### Direct Claim (Existing Runtime Upgrade)

For upgrading existing runtimes from shared to dedicated audit logging:

```go
// Claim AuditLogCR directly without reservation phase
data, err := provider.ClaimAuditLog(ctx, providerRegion, runtimeID)
```

This is used in `sFnPatchExistingShoot` when:
- Global feature flag (`DedicatedAuditLoggingEnabled`) is enabled
- Runtime has `spec.auditLogAccessEnabled: true`
- No AuditLog is already assigned to the runtime

The direct claim is safe for upgrades because:
- The shoot already exists (no risk of creation failure)
- If patch fails, claim persists for retry
- Saves one Gardener reconciliation cycle (~10 minutes)

**Phase 1: Reserve** (during shoot creation in `sFnCreateShoot`)
```go
// Reserve an AuditLogCR by adding reservation labels (light lock)
err := provider.ReserveAuditLog(ctx, providerRegion, runtimeID)
```

**Phase 2: Claim** (after successful provisioning in `sFnMigrateToDedicatedAuditLog`)
```go
// Upgrade reservation to full claim by setting assignedToRuntimeID (heavy lock)
data, err := provider.GetDedicatedAuditLogData(ctx, runtimeID, true)
```

#### Direct Claim (Existing Runtime Upgrade)

For upgrading existing runtimes from shared to dedicated audit logging:

```go
// Claim AuditLogCR directly without reservation phase
data, err := provider.ClaimAuditLog(ctx, providerRegion, runtimeID)
```

This is used in `sFnPatchExistingShoot` when:
- Global feature flag (`DedicatedAuditLoggingEnabled`) is enabled
- Runtime has `spec.auditLogAccessEnabled: true`
- No AuditLog is already assigned to the runtime

The direct claim is safe for upgrades because:
- The shoot already exists (no risk of creation failure)
- If patch fails, claim persists for retry
- Saves one Gardener reconciliation cycle (~10 minutes)

### Releasing Dedicated Resources

```go
// Called when runtime is being deleted
err := provider.ReleaseDedicated(ctx, runtimeID)
```

## How Dedicated Logging Works

### Claiming Strategies

The package implements two claiming strategies depending on the scenario:

#### 1. Two-Phase Reservation (New Runtime Creation)

Used during initial runtime provisioning to prevent wasting dedicated resources if shoot creation fails:

1. **Phase 1 - Reserve** (in `sFnCreateShoot`):
   - Adds `reserved-for-runtime-id` label to AuditLogCR (light lock)
   - Prevents other runtimes from claiming this CR during provisioning
   - Minimal resource commitment if provisioning fails

2. **Phase 2 - Claim** (in `sFnMigrateToDedicatedAuditLog`):
   - Sets `Spec.AssignedToRuntimeID` field (heavy lock)
   - Only executed after successful runtime provisioning
   - Full resource commitment

3. **Release** (on runtime deletion):
   - Sets `Spec.Orphaned = true`
   - KALM maintains logs for retention period (default: 90 days)

#### 2. Direct Claim (Existing Runtime Upgrade)

Used when upgrading existing runtimes to dedicated audit logging (in `sFnPatchExistingShoot`):

1. **Direct Claim**:
   - Sets `Spec.AssignedToRuntimeID` immediately (no reservation phase)
   - Shoot configuration updated with dedicated config in same operation
   - Safe because shoot already exists (no risk of creation failure)

2. **Benefits**:
   - Single Gardener reconciliation (saves ~10 minutes)
   - No "brief shared logging period" during upgrade
   - Simpler upgrade flow

3. **Release** (same as two-phase):
   - Sets `Spec.Orphaned = true`
   - KALM maintains logs for retention period (default: 90 days)

### Pool-Based Provisioning

The Kyma Audit Log Manager (KALM) maintains a pool of pre-provisioned AuditLog CRs. CRs are available when:
- State is `RegistrationReady` or `SiemApproved`
- `Spec.AssignedToRuntimeID` is empty
- No reservation labels present
- `Spec.Regions` contains the runtime's hyperscaler region

### Idempotent Operations

All operations are designed to be idempotent:

- **ReserveAuditLog**: Checks for existing reservation before creating new one
- **GetDedicatedAuditLogData**: Checks for existing claim before upgrading reservation
- **ClaimAuditLog**: Returns existing data if already claimed, otherwise finds and claims available CR
- **ReleaseDedicated**: Safe to call multiple times

### Optimistic Concurrency

Claims use Kubernetes optimistic concurrency control via `resourceVersion`. If two runtimes try to claim the same AuditLogCR simultaneously, one will get a conflict error and retry with a different CR from the pool.

## Integration with FSM

The DataProvider replaces the direct use of `auditlogs.Configuration` in the FSM:

### Before (Direct map access)
```go
cfg := fsm.RCCfg{
    AuditLogging: auditLogDataMap, // map[provider]map[region]AuditLogData
}
```

### After (Provider interface)
```go
cfg := fsm.RCCfg{
    AuditLogDataProvider: provider, // Abstracts shared vs dedicated
    DedicatedAuditLoggingEnabled: true,
}
```

### FSM State Flow

#### New Runtime Creation (Two-Phase)

1. **sFnCreateShoot**: Reserve AuditLogCR (Phase 1), create shoot with shared audit logging
2. **sFnWaitForShootCreation**: Wait for shoot to be ready
3. **... full provisioning flow ...**
4. **sFnMigrateToDedicatedAuditLog** (final step): 
   - Claim AuditLogCR - Phase 2 (upgrade reservation to full claim)
   - Patch shoot with dedicated config (only if configs differ)
   - Complete provisioning
5. **sFnCopyAuditLogReadCredentials**: Copy read credentials to SKR

#### Existing Runtime Upgrade (Direct Claim)

1. **sFnPatchExistingShoot**: 
   - Detects `spec.auditLogAccessEnabled: true` with no existing AuditLog
   - Calls `ClaimAuditLog` to claim directly
   - Patches shoot with dedicated config immediately
2. **sFnWaitForShootReconcile**: Wait for shoot update
3. **... rest of reconciliation flow ...**
4. **sFnMigrateToDedicatedAuditLog**: Becomes no-op (shoot already configured)
5. **sFnCopyAuditLogReadCredentials**: Copy read credentials to SKR

## Error Handling

### New Runtime Creation (Two-Phase)

- **Reservation fails**: Provisioning fails immediately, no resources wasted
- **Claim fails**: Should never happen if reservation succeeded; fails provisioning with clear error
- **Patch fails**: Requeues with claimed AuditLogCR (retries on next reconciliation)

### Existing Runtime Upgrade (Direct Claim)

- **Claim fails (pool exhausted)**: Runtime CR state set to Failed with `CustomAuditLogError`
- **Patch fails after claim**: Requeues with claimed AuditLogCR (retries on next reconciliation)
- **No automatic fallback**: Explicit failure when dedicated is requested but unavailable

### Irreversibility

Once dedicated audit logging is enabled for a runtime, it **cannot be disabled**:
- Setting `spec.auditLogAccessEnabled: false` is ignored
- Warning logged: "Dedicated audit logging is irreversible - ignoring attempt to disable"
- Runtime continues using dedicated configuration

## Testing

```bash
go test ./pkg/auditlog/...
```

## Future Enhancements

1. **Metrics**: Track pool utilization, claim/release operations
2. **Regional affinity**: Prefer AuditLogCRs in the same region as the runtime
3. **Webhooks**: Validate AuditLogCR state transitions
4. **Status updates**: Report dedicated logging status in Runtime CR
