# Split GetDedicatedAuditLogData Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace `GetDedicatedAuditLogData(ctx, runtimeID, claim bool)` with two clearly-named methods — `ClaimDedicatedAuditLogData` and `GetDedicatedAuditLogData` (no bool param) — eliminating the boolean trap.

**Architecture:** Split the single method's two divergent code paths into two separate interface methods and implementations. Update the mock and both FSM callers to use the new names. All logic stays identical — this is a pure refactor.

**Tech Stack:** Go, controller-runtime fake client, testify/mock (hand-written mock), Ginkgo/Gomega (FSM tests), standard testify (provider tests).

## Global Constraints

- Go module: `github.com/kyma-project/infrastructure-manager`
- Run tests with: `make test` (unit tests only, excludes /test dir)
- Do not switch the mock to mockery generation
- Do not change any logic in `ReserveAuditLog`, `GetSharedAuditLogData`, or `ReleaseDedicated`
- Do not change the two-phase claim logic itself — only rename/split

---

### Task 1: Split interface and implementation in `provider.go`

**Files:**
- Modify: `pkg/auditlog/provider.go`

**Interfaces:**
- Produces: `ClaimDedicatedAuditLogData(ctx context.Context, runtimeID string) (AuditLogData, error)` and `GetDedicatedAuditLogData(ctx context.Context, runtimeID string) (AuditLogData, error)` on both the `DataProvider` interface and `DefaultDataProvider` struct

- [ ] **Step 1: Update the `DataProvider` interface**

In `pkg/auditlog/provider.go`, replace:
```go
// GetDedicatedAuditLogData returns audit log configuration from AuditLogCR
// When claim=true, performs Phase 2 of two-phase claim (upgrades reservation to full claim by setting assignedToRuntimeID)
// When claim=false, only retrieves data from already claimed/reserved resource
GetDedicatedAuditLogData(ctx context.Context, runtimeID string, claim bool) (AuditLogData, error)
```
with:
```go
// ClaimDedicatedAuditLogData performs Phase 2 of the two-phase claim: upgrades the reservation
// to a full claim by setting AssignedToRuntimeID, then returns the audit log configuration.
ClaimDedicatedAuditLogData(ctx context.Context, runtimeID string) (AuditLogData, error)

// GetDedicatedAuditLogData returns audit log configuration from an already claimed or reserved
// AuditLogCR. Read-only — does not modify the CR. Falls back from claim lookup to reservation
// lookup so it works in the window between Phase 1 and Phase 2.
GetDedicatedAuditLogData(ctx context.Context, runtimeID string) (AuditLogData, error)
```

- [ ] **Step 2: Replace the implementation method**

In `pkg/auditlog/provider.go`, replace the entire `GetDedicatedAuditLogData` method on `DefaultDataProvider`:

```go
// ClaimDedicatedAuditLogData performs Phase 2 of two-phase claim (upgrades reservation to full claim)
func (p *DefaultDataProvider) ClaimDedicatedAuditLogData(ctx context.Context, runtimeID string) (AuditLogData, error) {
	reserved, err := p.findAuditLogCRByReservation(ctx, runtimeID)
	if err != nil {
		return AuditLogData{}, fmt.Errorf("failed to find reserved AuditLogCR: %w", err)
	}
	if reserved == nil {
		return AuditLogData{}, fmt.Errorf("no reservation found for runtime %s", runtimeID)
	}

	// Upgrade to claim if not already claimed (idempotent)
	if reserved.Spec.AssignedToRuntimeID != runtimeID {
		reserved.Spec.AssignedToRuntimeID = runtimeID
		if err := p.client.Update(ctx, reserved); err != nil {
			return AuditLogData{}, fmt.Errorf("failed to claim AuditLogCR: %w", err)
		}
		p.logger.Info("Successfully claimed AuditLogCR", "runtimeID", runtimeID, "auditLogCR", reserved.Name)
	} else {
		p.logger.Info("AuditLogCR already claimed", "runtimeID", runtimeID)
	}

	return AuditLogData{
		TenantID:   reserved.Spec.SubaccountID,
		ServiceURL: reserved.Spec.Config.ServiceURL,
		SecretName: reserved.Spec.Config.GardenerSecretName,
	}, nil
}

// GetDedicatedAuditLogData returns audit log configuration from an already claimed or reserved AuditLogCR (read-only)
func (p *DefaultDataProvider) GetDedicatedAuditLogData(ctx context.Context, runtimeID string) (AuditLogData, error) {
	auditLogCR, err := p.findAuditLogCRByRuntimeID(ctx, runtimeID)
	if err != nil {
		return AuditLogData{}, fmt.Errorf("failed to find claimed AuditLogCR: %w", err)
	}
	if auditLogCR == nil {
		auditLogCR, err = p.findAuditLogCRByReservation(ctx, runtimeID)
		if err != nil {
			return AuditLogData{}, fmt.Errorf("failed to find reserved AuditLogCR: %w", err)
		}
		if auditLogCR == nil {
			return AuditLogData{}, fmt.Errorf("no AuditLogCR found for runtime %s", runtimeID)
		}
	}

	return AuditLogData{
		TenantID:   auditLogCR.Spec.SubaccountID,
		ServiceURL: auditLogCR.Spec.Config.ServiceURL,
		SecretName: auditLogCR.Spec.Config.GardenerSecretName,
	}, nil
}
```

Also remove the comment on `GetDedicatedAuditLogData` in `DefaultDataProvider` — it no longer applies. The above replaces it entirely.

- [ ] **Step 3: Verify it compiles (mocks and callers will fail — that's expected)**

```bash
cd /Users/m00g3n/src/github.com/kyma-project/cluster-inventory
go build ./pkg/auditlog/...
```
Expected: compile error about missing interface satisfaction from mock and callers. That's fine — proceed to Task 2.

---

### Task 2: Update the hand-written mock

**Files:**
- Modify: `pkg/auditlog/mocks/data_provider.go`

**Interfaces:**
- Consumes: `ClaimDedicatedAuditLogData(ctx context.Context, runtimeID string) (AuditLogData, error)` and `GetDedicatedAuditLogData(ctx context.Context, runtimeID string) (AuditLogData, error)` from Task 1
- Produces: mock implementations of both new methods, matching testify/mock conventions used by the other methods in the file

- [ ] **Step 1: Replace the `GetDedicatedAuditLogData` mock method with two new ones**

In `pkg/auditlog/mocks/data_provider.go`, replace:
```go
// GetDedicatedAuditLogData provides a mock function with given fields: ctx, runtimeID, claim
func (m *DataProvider) GetDedicatedAuditLogData(ctx context.Context, runtimeID string, claim bool) (auditlog.AuditLogData, error) {
	ret := m.Called(ctx, runtimeID, claim)

	var r0 auditlog.AuditLogData
	if rf, ok := ret.Get(0).(func(context.Context, string, bool) auditlog.AuditLogData); ok {
		r0 = rf(ctx, runtimeID, claim)
	} else {
		r0 = ret.Get(0).(auditlog.AuditLogData)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string, bool) error); ok {
		r1 = rf(ctx, runtimeID, claim)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
```
with:
```go
// ClaimDedicatedAuditLogData provides a mock function with given fields: ctx, runtimeID
func (m *DataProvider) ClaimDedicatedAuditLogData(ctx context.Context, runtimeID string) (auditlog.AuditLogData, error) {
	ret := m.Called(ctx, runtimeID)

	var r0 auditlog.AuditLogData
	if rf, ok := ret.Get(0).(func(context.Context, string) auditlog.AuditLogData); ok {
		r0 = rf(ctx, runtimeID)
	} else {
		r0 = ret.Get(0).(auditlog.AuditLogData)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, runtimeID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetDedicatedAuditLogData provides a mock function with given fields: ctx, runtimeID
func (m *DataProvider) GetDedicatedAuditLogData(ctx context.Context, runtimeID string) (auditlog.AuditLogData, error) {
	ret := m.Called(ctx, runtimeID)

	var r0 auditlog.AuditLogData
	if rf, ok := ret.Get(0).(func(context.Context, string) auditlog.AuditLogData); ok {
		r0 = rf(ctx, runtimeID)
	} else {
		r0 = ret.Get(0).(auditlog.AuditLogData)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, runtimeID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
```

- [ ] **Step 2: Verify mock compiles**

```bash
go build ./pkg/auditlog/...
```
Expected: success (callers still fail — that's fine).

---

### Task 3: Update `provider_test.go`

**Files:**
- Modify: `pkg/auditlog/provider_test.go`

**Interfaces:**
- Consumes: `ClaimDedicatedAuditLogData` and `GetDedicatedAuditLogData` (no bool) from Task 1

- [ ] **Step 1: Split `TestDefaultDataProvider_GetDedicatedAuditLogData` into two test functions**

In `pkg/auditlog/provider_test.go`, replace the entire `TestDefaultDataProvider_GetDedicatedAuditLogData` function with two separate functions.

The existing test cases map as follows:
- `"claims and returns data when claim=true"` → `TestDefaultDataProvider_ClaimDedicatedAuditLogData / "claims and returns data"`
- `"claims and returns data from RegistrationReady CR when claim=true"` → `TestDefaultDataProvider_ClaimDedicatedAuditLogData / "claims and returns data from RegistrationReady CR"`
- `"returns data without claiming when claim=false"` → `TestDefaultDataProvider_GetDedicatedAuditLogData / "returns data without claiming"`
- `"returns data when claim=true and CR already claimed (idempotent)"` → `TestDefaultDataProvider_ClaimDedicatedAuditLogData / "idempotent when already claimed"`
- `"returns error when no reservation found with claim=true"` → `TestDefaultDataProvider_ClaimDedicatedAuditLogData / "returns error when no reservation found"`

Replace with:
```go
func TestDefaultDataProvider_ClaimDedicatedAuditLogData(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, auditlogv1.AddToScheme(scheme))
	logger := zap.New(zap.UseDevMode(true))

	t.Run("claims and returns data", func(t *testing.T) {
		auditLog := createAuditLogCR("al-1", auditlogv1.StateSiemApproved, "", []string{"eu-central-1"}, map[string]string{
			LabelReservedForRuntimeID: "test-runtime",
		})
		auditLog.Spec.SubaccountID = "dedicated-tenant"
		auditLog.Spec.Config.ServiceURL = "https://dedicated.example.com"
		auditLog.Spec.Config.GardenerSecretName = "dedicated-secret"

		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithRuntimeObjects(&auditLog).
			Build()

		provider := NewDataProvider(fakeClient, nil, logger)

		data, err := provider.ClaimDedicatedAuditLogData(context.Background(), "test-runtime")
		require.NoError(t, err)
		assert.Equal(t, "dedicated-tenant", data.TenantID)
		assert.Equal(t, "https://dedicated.example.com", data.ServiceURL)
		assert.Equal(t, "dedicated-secret", data.SecretName)

		var updated auditlogv1.AuditLog
		err = fakeClient.Get(context.Background(), namespacedName("al-1"), &updated)
		require.NoError(t, err)
		assert.Equal(t, "test-runtime", updated.Spec.AssignedToRuntimeID)
	})

	t.Run("claims and returns data from RegistrationReady CR", func(t *testing.T) {
		auditLog := createAuditLogCR("al-1", auditlogv1.StateRegistrationReady, "", []string{"eu-central-1"}, map[string]string{
			LabelReservedForRuntimeID: "test-runtime",
		})
		auditLog.Spec.SubaccountID = "dedicated-tenant"
		auditLog.Spec.Config.ServiceURL = "https://dedicated.example.com"
		auditLog.Spec.Config.GardenerSecretName = "dedicated-secret"

		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithRuntimeObjects(&auditLog).
			Build()

		provider := NewDataProvider(fakeClient, nil, logger)

		data, err := provider.ClaimDedicatedAuditLogData(context.Background(), "test-runtime")
		require.NoError(t, err)
		assert.Equal(t, "dedicated-tenant", data.TenantID)
		assert.Equal(t, "https://dedicated.example.com", data.ServiceURL)
		assert.Equal(t, "dedicated-secret", data.SecretName)

		var updated auditlogv1.AuditLog
		err = fakeClient.Get(context.Background(), namespacedName("al-1"), &updated)
		require.NoError(t, err)
		assert.Equal(t, "test-runtime", updated.Spec.AssignedToRuntimeID)
	})

	t.Run("idempotent when already claimed", func(t *testing.T) {
		auditLog := createAuditLogCR("al-1", auditlogv1.StateSiemApproved, "test-runtime", []string{"eu-central-1"}, map[string]string{
			LabelReservedForRuntimeID: "test-runtime",
		})
		auditLog.Spec.SubaccountID = "dedicated-tenant"
		auditLog.Spec.Config.ServiceURL = "https://dedicated.example.com"
		auditLog.Spec.Config.GardenerSecretName = "dedicated-secret"

		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithRuntimeObjects(&auditLog).
			Build()

		provider := NewDataProvider(fakeClient, nil, logger)

		data, err := provider.ClaimDedicatedAuditLogData(context.Background(), "test-runtime")
		require.NoError(t, err)
		assert.Equal(t, "dedicated-tenant", data.TenantID)
		assert.Equal(t, "https://dedicated.example.com", data.ServiceURL)
		assert.Equal(t, "dedicated-secret", data.SecretName)
	})

	t.Run("returns error when no reservation found", func(t *testing.T) {
		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			Build()

		provider := NewDataProvider(fakeClient, nil, logger)

		_, err := provider.ClaimDedicatedAuditLogData(context.Background(), "test-runtime")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no reservation found")
	})
}

func TestDefaultDataProvider_GetDedicatedAuditLogData(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, auditlogv1.AddToScheme(scheme))
	logger := zap.New(zap.UseDevMode(true))

	t.Run("returns data without claiming", func(t *testing.T) {
		auditLog := createAuditLogCR("al-1", auditlogv1.StateSiemApproved, "test-runtime", []string{"eu-central-1"}, nil)
		auditLog.Spec.SubaccountID = "dedicated-tenant"
		auditLog.Spec.Config.ServiceURL = "https://dedicated.example.com"
		auditLog.Spec.Config.GardenerSecretName = "dedicated-secret"

		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithRuntimeObjects(&auditLog).
			Build()

		provider := NewDataProvider(fakeClient, nil, logger)

		data, err := provider.GetDedicatedAuditLogData(context.Background(), "test-runtime")
		require.NoError(t, err)
		assert.Equal(t, "dedicated-tenant", data.TenantID)
		assert.Equal(t, "https://dedicated.example.com", data.ServiceURL)
		assert.Equal(t, "dedicated-secret", data.SecretName)

		// Verify no write occurred
		var updated auditlogv1.AuditLog
		err = fakeClient.Get(context.Background(), namespacedName("al-1"), &updated)
		require.NoError(t, err)
		assert.Equal(t, "test-runtime", updated.Spec.AssignedToRuntimeID)
	})

	t.Run("falls back to reservation when not yet claimed", func(t *testing.T) {
		auditLog := createAuditLogCR("al-1", auditlogv1.StateSiemApproved, "", []string{"eu-central-1"}, map[string]string{
			LabelReservedForRuntimeID: "test-runtime",
		})
		auditLog.Spec.SubaccountID = "dedicated-tenant"
		auditLog.Spec.Config.ServiceURL = "https://dedicated.example.com"
		auditLog.Spec.Config.GardenerSecretName = "dedicated-secret"

		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithRuntimeObjects(&auditLog).
			Build()

		provider := NewDataProvider(fakeClient, nil, logger)

		data, err := provider.GetDedicatedAuditLogData(context.Background(), "test-runtime")
		require.NoError(t, err)
		assert.Equal(t, "dedicated-tenant", data.TenantID)
		assert.Equal(t, "https://dedicated.example.com", data.ServiceURL)
		assert.Equal(t, "dedicated-secret", data.SecretName)

		// Verify no write occurred
		var updated auditlogv1.AuditLog
		err = fakeClient.Get(context.Background(), namespacedName("al-1"), &updated)
		require.NoError(t, err)
		assert.Empty(t, updated.Spec.AssignedToRuntimeID)
	})

	t.Run("returns error when no CR found", func(t *testing.T) {
		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			Build()

		provider := NewDataProvider(fakeClient, nil, logger)

		_, err := provider.GetDedicatedAuditLogData(context.Background(), "test-runtime")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no AuditLogCR found for runtime")
	})
}
```

- [ ] **Step 2: Run provider tests**

```bash
go test ./pkg/auditlog/... -v -run "TestDefaultDataProvider"
```
Expected: all tests pass.

- [ ] **Step 3: Commit**

```bash
git add pkg/auditlog/provider.go pkg/auditlog/mocks/data_provider.go pkg/auditlog/provider_test.go
git commit -m "refactor: split GetDedicatedAuditLogData into Claim and Get variants"
```

---

### Task 4: Update FSM caller — `runtime_fsm_migrate_dedicated_auditlog.go`

**Files:**
- Modify: `internal/controller/runtime/fsm/runtime_fsm_migrate_dedicated_auditlog.go`
- Modify: `internal/controller/runtime/fsm/runtime_fsm_migrate_dedicated_auditlog_test.go`

**Interfaces:**
- Consumes: `ClaimDedicatedAuditLogData(ctx context.Context, runtimeID string) (AuditLogData, error)` from Task 1

- [ ] **Step 1: Update the call site in `sFnMigrateToDedicatedAuditLog`**

In `runtime_fsm_migrate_dedicated_auditlog.go`, replace:
```go
	auditLogData, err := m.AuditLogDataProvider.GetDedicatedAuditLogData(
		ctx,
		runtimeID,
		true, // claim=true to upgrade reservation to full claim
	)
```
with:
```go
	auditLogData, err := m.AuditLogDataProvider.ClaimDedicatedAuditLogData(ctx, runtimeID)
```

- [ ] **Step 2: Update mock expectations in the test file**

In `runtime_fsm_migrate_dedicated_auditlog_test.go`, replace every occurrence of:
```go
mockAuditLogProvider.On("GetDedicatedAuditLogData", ctx, runtimeID, true).
```
with:
```go
mockAuditLogProvider.On("ClaimDedicatedAuditLogData", ctx, runtimeID).
```

There are 4 such occurrences (lines 152, 252, 349, 443 approximately). Replace all of them.

- [ ] **Step 3: Run the FSM migrate tests**

```bash
go test ./internal/controller/runtime/fsm/... -v -run "TestFSMMigrate"
```
Expected: all tests pass.

- [ ] **Step 4: Commit**

```bash
git add internal/controller/runtime/fsm/runtime_fsm_migrate_dedicated_auditlog.go \
        internal/controller/runtime/fsm/runtime_fsm_migrate_dedicated_auditlog_test.go
git commit -m "refactor: update sFnMigrateToDedicatedAuditLog to use ClaimDedicatedAuditLogData"
```

---

### Task 5: Update FSM caller — `runtime_fsm_patch_shoot.go`

**Files:**
- Modify: `internal/controller/runtime/fsm/runtime_fsm_patch_shoot.go`
- Modify: `internal/controller/runtime/fsm/runtime_fsm_patch_shoot_test.go`

**Interfaces:**
- Consumes: `GetDedicatedAuditLogData(ctx context.Context, runtimeID string) (AuditLogData, error)` (no bool) from Task 1

- [ ] **Step 1: Update the call site in `sFnPatchExistingShoot`**

In `runtime_fsm_patch_shoot.go`, replace:
```go
		data, err = m.AuditLogDataProvider.GetDedicatedAuditLogData(
			ctx,
			runtimeID,
			false, // don't claim, just retrieve
		)
```
with:
```go
		data, err = m.AuditLogDataProvider.GetDedicatedAuditLogData(ctx, runtimeID)
```

- [ ] **Step 2: Update mock setup in test helpers**

In `runtime_fsm_patch_shoot_test.go`, in `newMockAuditLogDataProvider` and `newMockAuditLogDataProviderWithError`, replace:
```go
	mockProvider.On("GetDedicatedAuditLogData", mock.Anything, mock.Anything, mock.Anything).Return(data, nil)
```
with:
```go
	mockProvider.On("GetDedicatedAuditLogData", mock.Anything, mock.Anything).Return(data, nil)
	mockProvider.On("ClaimDedicatedAuditLogData", mock.Anything, mock.Anything).Return(data, nil)
```

And in `newMockAuditLogDataProviderWithError`:
```go
	mockProvider.On("GetDedicatedAuditLogData", mock.Anything, mock.Anything, mock.Anything).Return(auditlog.AuditLogData{}, fmt.Errorf("mock audit log error"))
```
with:
```go
	mockProvider.On("GetDedicatedAuditLogData", mock.Anything, mock.Anything).Return(auditlog.AuditLogData{}, fmt.Errorf("mock audit log error"))
	mockProvider.On("ClaimDedicatedAuditLogData", mock.Anything, mock.Anything).Return(auditlog.AuditLogData{}, fmt.Errorf("mock audit log error"))
```

- [ ] **Step 3: Run the full test suite**

```bash
make test
```
Expected: all tests pass, no compilation errors.

- [ ] **Step 4: Commit**

```bash
git add internal/controller/runtime/fsm/runtime_fsm_patch_shoot.go \
        internal/controller/runtime/fsm/runtime_fsm_patch_shoot_test.go
git commit -m "refactor: update sFnPatchExistingShoot to use GetDedicatedAuditLogData without bool param"
```
