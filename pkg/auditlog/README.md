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
    dedicatedAuditLoggingEnabled, // feature flag
    logger,
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

### Two-Phase Claiming Process

Dedicated audit logging uses a two-phase approach:

**Phase 1: Reserve** (during shoot creation)
```go
// Reserve an AuditLogCR by adding reservation labels (light lock)
err := provider.ReserveAuditLog(ctx, providerRegion, runtimeID)
```

**Phase 2: Claim** (after successful provisioning)
```go
// Upgrade reservation to full claim by setting assignedToRuntimeID (heavy lock)
data, err := provider.GetDedicatedAuditLogData(ctx, runtimeID, true)
```

### Releasing Dedicated Resources

```go
// Called when runtime is being deleted
err := provider.ReleaseDedicated(ctx, runtimeID)
```

## How Dedicated Logging Works

### Two-Phase Reservation System

The package implements a two-phase reservation system to prevent race conditions:

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

### Pool-Based Provisioning

The Kyma Audit Log Manager (KALM) maintains a pool of pre-provisioned AuditLog CRs. CRs are available when:
- State is `RegistrationReady` or `SiemApproved`
- `Spec.AssignedToRuntimeID` is empty
- No reservation labels present
- `Spec.Regions` contains the runtime's hyperscaler region

### Idempotent Operations

All operations are designed to be idempotent:

- **reserveAuditLogCR**: Checks for existing reservation before creating new one
- **getOrClaimAuditLogCR**: Checks for existing claim before upgrading reservation
- **findAuditLogCRByRuntimeID**: Uses loop-based filtering (spec fields not indexed by default)
- **releaseAuditLogCR**: Safe to call multiple times

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

1. **sFnCreateShoot**: Reserve AuditLogCR (Phase 1), create shoot with shared audit logging
2. **sFnWaitForShootCreation**: Wait for shoot to be ready
3. **... full provisioning flow ...**
4. **sFnMigrateToDedicatedAuditLog** (final step): 
   - Claim AuditLogCR - Phase 2 (upgrade reservation to full claim)
   - Patch shoot with dedicated config (only if configs differ)
   - Complete provisioning

## Error Handling

The two-phase approach provides fail-fast behavior:

- **Reservation fails**: Provisioning fails immediately, no resources wasted
- **Claim fails**: Should never happen if reservation succeeded; fails provisioning with clear error
- **Patch fails**: Requeues with claimed AuditLogCR (retries on next reconciliation)

No automatic fallback to shared logging - explicit failure when dedicated is requested but unavailable.

## Testing

```bash
go test ./pkg/auditlog/...
```

## Future Enhancements

1. **Metrics**: Track pool utilization, claim/release operations
2. **Regional affinity**: Prefer AuditLogCRs in the same region as the runtime
3. **Webhooks**: Validate AuditLogCR state transitions
4. **Status updates**: Report dedicated logging status in Runtime CR
