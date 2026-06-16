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

```go
// Get audit log data (tries dedicated first if enabled, falls back to shared)
data, err := provider.GetAuditLogData(
    ctx,
    providerType, // e.g., "aws", "azure", "gcp"
    region,       // e.g., "us-east-1", "westeurope"
    runtimeID,    // e.g., "my-runtime-123"
    useDedicated, // true to prefer dedicated logging
)

// data.IsDedicated tells you which source was used
if data.IsDedicated {
    log.Info("Using dedicated audit logging", "runtimeID", runtimeID)
}
```

### Checking if Runtime Uses Dedicated Logging

```go
isDedicated, err := provider.IsDedicated(ctx, runtimeID)
if isDedicated {
    log.Info("Runtime is using dedicated audit logging")
}
```

### Releasing Dedicated Resources

```go
// Called when runtime is being deleted
err := provider.ReleaseDedicated(ctx, runtimeID)
```

## How Dedicated Logging Works

### Pool-Based Provisioning

The Kyma Audit Log Manager (KALM) maintains a pool of pre-provisioned AuditLog CRs in the `SiemApproved` state. When KIM needs dedicated logging:

1. **Claim**: Find an available AuditLogCR and set `Spec.AssignedToRuntimeID`
2. **Use**: Extract configuration from `Spec.Config` field
3. **Release**: Set `Spec.Orphaned = true` when runtime is deleted

### Idempotent Operations

All operations are designed to be idempotent:

- **getOrClaimAuditLogCR**: Checks for existing claim before creating new one
- **findAuditLogCRByRuntimeID**: Uses indexed field lookup for efficiency  
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

1. **sFnCreateShoot**: Create shoot with shared audit logging
2. **sFnWaitForShootReady**: Wait for shoot to be ready
3. **sFnMigrateToDedicatedAuditLog**: 
   - Claim AuditLogCR (idempotent)
   - Update shoot with dedicated config (idempotent)
4. **sFnApply**: Continue with normal flow

## Error Handling

The provider implements graceful degradation:

```go
data, err := provider.GetAuditLogData(ctx, provider, region, runtimeID, true)
// If dedicated logging fails (no available CRs, claim conflict, etc.),
// it automatically falls back to shared configuration
// Check data.IsDedicated to see which was used
```

## Testing

```bash
go test ./pkg/auditlog/...
```

## Future Enhancements

1. **Metrics**: Track pool utilization, claim/release operations
2. **Regional affinity**: Prefer AuditLogCRs in the same region as the runtime
3. **Webhooks**: Validate AuditLogCR state transitions
4. **Status updates**: Report dedicated logging status in Runtime CR
