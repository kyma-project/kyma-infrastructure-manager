# KIM Reconciliation Observability — Operator & L3 Support Reference

This document describes how the Kyma Infrastructure Manager (KIM) reconciliation process is observable, what exception-reporting mechanisms exist, and what improvements are recommended to better support operators and Level 3 support.

---

## 1. Reconciliation Process Overview

KIM manages the full lifecycle of Kyma cluster infrastructure (provisioning, updates, deletion) through a **Finite State Machine (FSM)**. Every change to a `Runtime` CR triggers a reconciliation run that steps through states sequentially until the cluster reaches a stable condition or a terminal error occurs.

### 1.1 FSM State Sequence

#### Provisioning (new cluster)

```
sFnTakeSnapshot
  └─► sFnInitialize
        └─► sFnCreateShoot                     (calls Gardener API)
              └─► sFnSelectShootProcessing
                    └─► sFnWaitForShootCreation (polls Gardener, ~60 s requeue)
                          └─► sFnHandleKubeconfig
                                └─► sFnCreateKymaNamespace
                                      └─► sFnInitializeRuntimeBootstrapper
                                            └─► sFnSyncRegistryCacheGardenSecrets
                                                  └─► sFnCleanupRegistryCacheGardenSecrets
                                                        └─► sFnConfigureSKR
                                                              └─► sFnApplyClusterRoleBindings
                                                                    └─► [sFnMigrateToDedicatedAuditLog]
                                                                          └─► sFnCopyAuditLogReadCredentials
                                                                                └─► sFnUpdateStatus
                                                                                      └─► sFnEmitEvent
```

#### Update (existing cluster)

```
sFnTakeSnapshot → sFnInitialize → sFnSelectShootProcessing
  └─► sFnSyncRegistryCacheGardenSecrets
        └─► sFnPatchExistingShoot
              └─► sFnWaitForShootReconcile (polls Gardener, ~30 s requeue)
                    └─► sFnHandleKubeconfig → … → sFnUpdateStatus → sFnEmitEvent
```

#### Deletion

```
sFnTakeSnapshot → sFnInitialize (detects deletionTimestamp)
  └─► sFnDeleteKubeconfig      (waits for GardenerCluster CR deletion)
        └─► sFnDeleteShoot     (adds deletion annotation, deletes Shoot)
              └─► sFnUpdateStatus → sFnEmitEvent
```

### 1.2 Requeue Delays

| Situation | Delay |
|-----------|-------|
| Waiting for Shoot creation | 60 s |
| Waiting for Shoot deletion | 90 s |
| Waiting for Shoot patch reconciliation | 30 s |
| Waiting for control-plane readiness | 10 s |
| General Gardener API retry | 15 s |
| After status write (cache propagation guard) | 1 s |

---

## 2. Exception Reporting — What Is Already in Place

### 2.1 Runtime CR Status (`status.state`)

The coarse-grained lifecycle state. Visible via `kubectl get runtime`.

| Value | Meaning |
|-------|---------|
| `Pending` | Provisioning or update in progress |
| `Ready` | Cluster fully provisioned and configured |
| `Terminating` | Deletion in progress |
| `Failed` | Non-recoverable error; no automatic retry |

### 2.2 Runtime CR Status Conditions (`status.conditions`)

Fine-grained, per-step result. Each condition has `Type`, `Status` (`True`/`False`/`Unknown`), `Reason`, and `Message`.

Visible via `kubectl describe runtime <name>`.

| Condition Type | Set during |
|----------------|-----------|
| `Provisioned` | Shoot creation / reconciliation |
| `KubeconfigReady` | GardenerCluster CR / kubeconfig issuance |
| `OidcAndConfigMapConfigured` | OIDC and ConfigMap application on the Kyma cluster |
| `KymaSystemNSCreated` | `kyma-system` namespace creation |
| `Configured` | ClusterRoleBinding configuration |
| `Deprovisioned` | Shoot / GardenerCluster deletion |
| `RegistryCacheConfigured` | Registry cache extension |
| `RuntimeBootstrapperReady` | Runtime Bootstrapper installation |
| `CustomAuditLogConfigured` | Dedicated audit log setup |
| `AuditLogCredentialsCopied` | Audit log read credential copy to Kyma cluster |

#### Condition Reasons

All condition reason strings, grouped by outcome. The `Category` column indicates whether the reason represents an error, a success/terminal state, or an in-progress state.

| Reason | Condition | Category |
|--------|-----------|----------|
| `Processing` | `Provisioned` | In-progress |
| `Pending` | `Provisioned` | In-progress |
| `ShootCreationCompleted` | `Provisioned` | Success |
| `ConfigurationCompleted` | `Configured` | Success |
| `GardenerErr` | `Provisioned` | Error — Transient / Permanent |
| `ConversionErr` | `Provisioned` | Error — Permanent |
| `CreationErr` | `Provisioned` | Error — Permanent |
| `ProcessingErr` | `Provisioned` | Error — Transient |
| `SeedNotFound` | `Provisioned` | Error — Permanent |
| `AuditLogErr` | `Provisioned` | Error — Permanent (if audit log mandatory) |
| `Initialised` | `Provisioned` | In-progress |
| `GardenerClusterCRCreated` | `KubeconfigReady` | In-progress |
| `GardenerClusterCRReady` | `KubeconfigReady` | Success |
| `KubernetesErr` | `KubeconfigReady` | Error — Transient |
| `OidcAndConfigMapsConfigured` | `OidcAndConfigMapConfigured` | Success |
| `OidcConfigurationErr` | `OidcAndConfigMapConfigured` | Error — Permanent |
| `ConfigMapErr` | `OidcAndConfigMapConfigured` | Error — Permanent |
| `KymaSystemNSReady` | `KymaSystemNSCreated` | Success |
| `KymaSystemNSError` | `KymaSystemNSCreated` | Error — Transient |
| `AdministratorsConfigured` | `Configured` | Success |
| `ConfigurationError` | `Configured` | Error — Transient |
| `GardenerClusterCRDeleted` | `Deprovisioned` | Success |
| `GardenerShootDeleted` | `Deprovisioned` | Success |
| `StructuredConfigDeleted` | `Deprovisioned` | Success |
| `RegistryCacheConfigured` | `RegistryCacheConfigured` | Success |
| `RegistryCacheError` | `RegistryCacheConfigured` | Error — Transient |
| `RegistryCacheGardenClusterConfigurationFailed` | `RegistryCacheConfigured` | Error — Transient |
| `RegistryCacheGardenClusterCleanupFailed` | `RegistryCacheConfigured` | Error — Transient |
| `RuntimeBootstrapperConfigured` | `RuntimeBootstrapperReady` | Success |
| `RuntimeBootstrapperInstallationInProgress` | `RuntimeBootstrapperReady` | In-progress |
| `RuntimeBootstrapperInstallationFailed` | `RuntimeBootstrapperReady` | Error — Transient |
| `RuntimeBootstrapperUpgradeInProgress` | `RuntimeBootstrapperReady` | In-progress |
| `RuntimeBootstrapperUpgradeFailed` | `RuntimeBootstrapperReady` | Error — Transient |
| `RuntimeBootstrapperConfigurationFailed` | `RuntimeBootstrapperReady` | Error — Transient |
| `RuntimeBootstrapperStatusUnknown` | `RuntimeBootstrapperReady` | Unknown |
| `CustomAuditLogConfigured` | `CustomAuditLogConfigured` | Success |
| `CustomAuditLogErr` | `CustomAuditLogConfigured` | Error — Permanent |
| `CredentialsCopied` | `AuditLogCredentialsCopied` | Success |
| `CredentialsCopyErr` | `AuditLogCredentialsCopied` | Error — Transient |

### 2.3 Kubernetes Events

KIM emits a Kubernetes `Event` on the `Runtime` CR whenever **any condition changes**. Events distinguish between normal and warning types:

- `Normal` — condition `Status` is `True` or `Unknown`
- `Warning` — condition `Status` is `False`

Visible via `kubectl describe runtime <name>` in the `Events:` section, or via `kubectl get events --field-selector involvedObject.name=<runtime-name>`.

Event format:
```
Type     Reason                    Message
----     ------                    -------
Normal   ShootCreationCompleted    Shoot creation completed: kcp-system/my-runtime
Warning  AuditLogErr               Failed to configure audit log: kcp-system/my-runtime
Normal   ConfigurationCompleted    Runtime processing completed successfully: kcp-system/my-runtime
```

### 2.4 Prometheus Metrics

KIM exposes metrics at `:8080/metrics`. Three of the four metrics use the subsystem prefix `infrastructure_manager_`; `unexpected_stops_total` is registered without a subsystem prefix and is queried by its bare name.

#### `infrastructure_manager_im_runtime_state` (GaugeVec)

Reports the current `status.state` of each `Runtime` CR. Value is always `1`; the state is encoded in labels.

| Label | Description |
|-------|-------------|
| `runtimeId` | Value of label `kyma-project.io/runtime-id` |
| `runtimeName` | CR name |
| `shoot` | Gardener Shoot name from `spec.shoot.name` |
| `provider` | Cloud provider (`aws`, `azure`, `gcp`, …) |
| `state` | `Pending` / `Ready` / `Failed` / `Terminating` |
| `message` | Last condition message (truncated) |

**Use case:** Alert when `state="Failed"` appears; trend `Pending` duration.

#### `infrastructure_manager_im_gardener_clusters_state` (GaugeVec)

Reports the state of each `GardenerCluster` CR (kubeconfig rotation).

| Label | Description |
|-------|-------------|
| `runtimeId` | Runtime ID |
| `shoot` | Shoot name |
| `state` | GardenerCluster state |
| `reason` | Last condition reason |

#### `infrastructure_manager_im_kubeconfig_expiration` (GaugeVec)

Reports kubeconfig expiration as a Unix epoch timestamp. Enables alerting before expiry.

| Label | Description |
|-------|-------------|
| `runtimeId` | Runtime ID |
| `shoot` | Shoot name |
| `expires` | Expiration epoch (also the gauge value) |
| `rotationDuration` | Configured rotation period (seconds) |
| `expirationDuration` | Effective expiration window (seconds) |

**Formula:** `expirationDuration = rotationPeriod / minimalRotationTimeRatio`

#### `unexpected_stops_total` (Counter)

Counts FSM runs that ended with a permanent (non-retriable) error. Incremented whenever:

- A permanent error causes `status.state = Failed`
- The status update itself fails

**Use case:** This is the primary metric for tracking unrecoverable provisioning failures. An increasing rate indicates a systemic problem.

---

## 3. Structured Logging

KIM uses `logr` with a `zap` backend. Log format is JSON in production mode (default) and console in development mode (`-zap-devel`).

### 3.1 Log Levels

| Level | Value | Content |
|-------|-------|---------|
| `Error` | — | Unexpected failures requiring attention |
| `Info` | 0 | Major events: reconciliation started/completed, errors, state changes, Gardener last errors |
| `Debug` | 1 | Intermediate steps, status checks, retry scheduling |
| `Trace` | 4+ | FSM state transitions (every function switch) |

Configure via `-zap-log-level`. For production, `Info` is the default. Set to `Debug` during incident investigation.

### 3.2 Common Structured Fields

All log entries from the FSM include contextual fields:

| Field | Source |
|-------|--------|
| `RuntimeCR` | Runtime CR name |
| `shoot` | Gardener Shoot name |
| `function` | FSM state function name (`sFnSelectShootProcessing`, etc.) |
| `result` | Requeue result |
| `err` | Error value |

### 3.3 Key Log Messages for Operators

| Message | Level | Meaning |
|---------|-------|---------|
| `Reconciling Runtime` | Info | A reconciliation cycle started |
| `Reconciliation done` | Debug | FSM run completed (normal or error) |
| `switching state from X to Y` | Trace | FSM transition |
| `runtimeID: …, Last operation state: …, last errors: …` | Info | Gardener reported errors in the Shoot's `status.lastErrors` |
| `Shoot indicates active reconciliation despite Runtime state Ready/Failed` | Info | Guard hit: Shoot is still processing; re-routing to wait state |
| `Stopping processing reconcile, exiting with no retry` | Info | No work needed; FSM stopped cleanly |
| `unable to update instance status!` | Error | Runtime CR status write failed; will be retried |
| `Failed to configure runtime bootstrapper` | Info | Runtime Bootstrapper setup failed; check bootstrapper logs |
| `Shoot creation failed, no matching seeds` | Info | No seed available for requested region; permanent failure |
| `Reconciliation is suspended` | Info | Suspension annotation present; manual action required |

---

## 4. How to Investigate a Failed Runtime

### Step 1 — Check the Runtime CR state and conditions

```bash
kubectl get runtime <name> -n kcp-system
kubectl describe runtime <name> -n kcp-system
```

Look at:
- `status.state` — overall state
- `status.conditions` — each condition's `Status`, `Reason`, and `Message`
- `Events` section — timeline of state changes

### Step 2 — Check metrics

```promql
# All runtimes currently in Failed state
infrastructure_manager_im_runtime_state{state="Failed"}

# Trend of unrecoverable errors
rate(unexpected_stops_total[5m])

# Kubeconfigs about to expire (within 1 hour)
infrastructure_manager_im_kubeconfig_expiration < (time() + 3600)
```

### Step 3 — Check KIM logs

```bash
# All log lines for a specific runtime
kubectl logs -n kcp-system -l app=infrastructure-manager | grep <runtime-name>

# All error-level logs
kubectl logs -n kcp-system -l app=infrastructure-manager | jq 'select(.level=="error")'

# Gardener last errors reported by FSM
kubectl logs -n kcp-system -l app=infrastructure-manager | grep "last errors"
```

### Step 4 — Check Gardener Shoot directly

For `Provisioned` condition errors, the root cause is often in the Gardener Shoot:

```bash
kubectl get shoot <shoot-name> -n <gardener-project-namespace> -o jsonpath='{.status.lastOperation}'
kubectl get shoot <shoot-name> -n <gardener-project-namespace> -o jsonpath='{.status.lastErrors}'
```

### Step 5 — Check GardenerCluster CR (kubeconfig issues)

```bash
kubectl get gardenerclusters -n kcp-system -l kyma-project.io/runtime-id=<runtime-id>
kubectl describe gardenercluster <name> -n kcp-system
```

---

## 5. Permanent vs. Transient Errors

Understanding whether an error will self-heal is critical for operator response.

### Permanent errors — `status.state = Failed`, no automatic retry

These require human intervention or a Runtime CR update to retry:

| Root cause | Condition Reason | Operator action |
|------------|-----------------|-----------------|
| No Gardener seed available for region | `SeedNotFound` | Check seed availability; update Runtime spec region |
| Audit log tenant mapping missing (mandatory mode) | `AuditLogErr` | Add mapping in converter config; restart KIM |
| Runtime-to-Shoot conversion failed | `ConversionErr` | Check converter config; fix Runtime spec |
| Shoot creation failed (no matching seeds) | `CreationErr` | Check region; update Runtime spec |
| Missing `runtime-id` label on Runtime CR | N/A (stops early) | Add required label |

**Recovery:** Fix the root cause, then trigger re-reconciliation by patching the Runtime CR (e.g., adding/updating an annotation) or removing the `Failed` state by updating the spec.

### Transient errors — automatic retry with back-off

These resolve automatically unless there is a persistent external problem:

| Root cause | Retry interval |
|------------|---------------|
| Shoot still provisioning | 60 s |
| Shoot still reconciling after patch | 30 s |
| Control plane not ready | 10 s |
| Gardener API conflict / forbidden | 15 s |
| Kubeconfig not yet issued | 10 s |
| ClusterRoleBindings API unavailable | 10 s |

---

## 6. Audit Log Error Path

Audit logging has special behavior controlled by the `-audit-log-mandatory` flag (default: `true`).

| Configuration | No audit log tenant found | Result |
|---------------|--------------------------|--------|
| `mandatory=true` (default) | Provisioning **fails permanently** | `status.state=Failed`, `AuditLogErr` |
| `mandatory=false` | Provisioning **continues** without audit log | Warning logged |

For dedicated audit log (`-dedicated-audit-logging-enabled`):
- If no `AuditLog` CR is available for the region, provisioning fails permanently with `CustomAuditLogErr`.

---

## 7. Reconciliation Suspension

An operator can suspend reconciliation for a specific Runtime by adding the annotation:

```
operator.kyma-project.io/suspend-reconcile: "true"
```

While suspended, KIM logs `Reconciliation is suspended` and takes no action. Remove the annotation to resume.

---

## 8. Plutono / Grafana Dashboard

KIM has a production dashboard at **Infrastructure Manager** (UID `R0wzpHTSk`) hosted on the internal Plutono instance at `plutono.cp.kyma.cloud.sap`. The dashboard is organised into five rows.

### 8.1 Row: Runtime CRs

| Panel | Type | Query | Purpose |
|-------|------|-------|---------|
| **Currently Pending Runtimes** | Gauge | `sum(infrastructure_manager_im_runtime_state{state="Pending"})` | Instant count of clusters in provisioning/update. Green threshold only — any non-zero value is expected during normal operations. |
| **Currently Ready Runtimes** | Gauge | `sum(infrastructure_manager_im_runtime_state{state="Ready"})` | Instant count of healthy clusters. |
| **Currently Terminating Runtimes** | Gauge | `sum(infrastructure_manager_im_runtime_state{state="Terminating"})` | Instant count of clusters under deletion. |
| **Cumulative Failed Runtimes** | Gauge | `sum(infrastructure_manager_im_runtime_state{state="Failed"})` | Turns red at any non-zero value (`threshold > 0.0001`). This is the primary health signal for operators. |
| **Runtime CRs States** | Time-series | `sum by (state)(infrastructure_manager_im_runtime_state{})` | Trend of all four states over time. Lets operators correlate spikes in `Pending` or `Failed` with deployments or external events. |
| **Runtime CR states** (table) | Table | `avg by(provider, runtimeId, runtimeName, shootName, state, message)(infrastructure_manager_im_runtime_state)` | Flat list of every Runtime with provider, Shoot name, current state, and last condition message. Sorted by `Message`. Allows L3 to scan all instances and identify which ones are stuck or failed without `kubectl`. |
| **Unexpected Runtime processing stops** | Time-series (graph) | `rate(unexpected_stops_total{job="infrastructure-manager-metrics"}[$__rate_interval])` | Rate of permanent FSM failures per pod. Annotated "should be always 0". Any non-zero value is an incident trigger. |

### 8.2 Row: GardenerCluster CRs

Covers the kubeconfig-rotation controller (`GardenerCluster` CRs).

| Panel | Type | Query | Purpose |
|-------|------|-------|---------|
| **Counter of different States** | Gauge | `sum by (state)(infrastructure_manager_im_gardener_clusters_state)` | Breakdown of GardenerCluster CRs by state. |
| **Counters of different State.Reasons** | Gauge | `sum by (reason)(infrastructure_manager_im_gardener_clusters_state)` | Breakdown by condition reason. Helps distinguish transient from structural issues. |
| **GardenerCluster CRs States** | Time-series | `sum by (state)(infrastructure_manager_im_gardener_clusters_state{})` | Trend of GardenerCluster states. |
| **GardenerCluster CRs States (current/instant)** | Table | `infrastructure_manager_im_gardener_clusters_state{}` | Per-cluster instant snapshot of state and reason, sorted by state. Columns: `state`, `reason`, `runtimeId`. |

### 8.3 Row: Pod Metrics

Infrastructure health of the KIM process itself.

| Panel | Type | Query | Purpose |
|-------|------|-------|---------|
| **Infrastructure Manager CPU usage** | Graph | `rate(process_cpu_seconds_total{job="infrastructure-manager-metrics"}[$__rate_interval])` | Per-pod CPU consumption. |
| **Infrastructure Manager Memory Usage** | Graph | `process_resident_memory_bytes{job="infrastructure-manager-metrics"}` | Resident memory per pod (bytes). |
| **External HTTP traffic by endpoint** | Graph | `sum(rate(rest_client_requests_total{...,code=~"2.."}[$__rate_interval])) by (host)` | Successful outbound request rate broken down by target host (Gardener, KCP API server, etc.). |
| **External HTTP traffic by code** | Graph | Four series: `2xx`, `3xx`, `4xx`, `5xx` via `rest_client_requests_total` | HTTP error-rate overview. Spikes in `4xx`/`5xx` indicate API connectivity or permission problems. |

### 8.4 Row: Kubebuilder Metrics

Metrics from `controller-runtime`, covering both the `runtime` and `gardenercluster` controllers.

| Panel | Type | Query | Purpose |
|-------|------|-------|---------|
| **Max Workers** | Gauge | `controller_runtime_max_concurrent_reconciles` per controller | Configured worker-pool size for each controller (default 25). |
| **Controller Reconcile Success** | Time-series | `rate(controller_runtime_reconcile_total{...,result=~"success|..."}[$__rate_interval])` | Throughput of successful reconciliations. Legend shows `controller - result`. |
| **Controller Reconcile Errors** | Time-series | `rate(controller_runtime_reconcile_errors_total{...}[$__rate_interval])` | Rate of reconciliation loops that returned an error (triggers a requeue). Sustained non-zero values are a sign of a retrying-but-not-resolving problem. |
| **Reconcile Duration** | Time-series | `rate(controller_runtime_reconcile_time_seconds_sum) / rate(controller_runtime_reconcile_time_seconds_count)` | **Average time spent processing one item from the work queue** (seconds). Covers both `runtime` and `gardenercluster` controllers. This is the closest existing approximation to end-to-end reconciliation latency per queue item (see gap §9.1). |
| **Work Queue Depth** | Time-series | `workqueue_depth{name="runtime"/"gardenercluster"}` | Number of items waiting to be processed. Sustained high depth indicates the controller cannot keep up. |
| **Work Queue Processing Latency** | Graph | `rate(workqueue_queue_duration_seconds_sum) / rate(workqueue_queue_duration_seconds_count)` | Average time an item spends **waiting in the queue** before being picked up (seconds). |
| **Work Queue Processing Duration** | Graph | `rate(workqueue_work_duration_seconds_sum) / rate(workqueue_work_duration_seconds_count)` | Average time spent **actively processing** an item once dequeued (seconds). |

### 8.5 Row: Kubeconfig Expirations

| Panel | Type | Query | Purpose |
|-------|------|-------|---------|
| **Kubeconfig expirations** (table) | Table | `avg by(expires, expirationDuration, rotationDuration, runtimeId, shootName)(infrastructure_manager_im_kubeconfig_expiration)` | Lists every Shoot with its expiration timestamp, rotation period, and effective expiration window. Columns renamed to `expires (timestamp)`, `expires (seconds)`, `rotate after (seconds)`. |
| **Kubeconfig expirations** (time-series) | Time-series | `max by (shootName)(infrastructure_manager_im_kubeconfig_expiration - time())` | Time until expiry per Shoot (seconds remaining). Lets operators see which kubeconfigs are closest to expiry at a glance. |

### 8.6 Dashboard Limitations and Gaps

The dashboard covers the most important signals but has the following gaps relevant to operators and L3 support (see §9 for full recommendations):

- **No per-step or per-reason failure breakdown.** The `unexpected_stops_total` panel shows an aggregate rate with no breakdown by condition reason or FSM step. It is not possible to distinguish `SeedNotFound` from `AuditLogErr` in the dashboard without querying logs.
- **"Cumulative Failed Runtimes" is misleading.** The panel title says "Cumulative" but queries the gauge `infrastructure_manager_im_runtime_state{state="Failed"}`, which is not cumulative — it shows how many Runtime CRs are *currently* in `Failed` state. Clusters that were fixed and moved to `Ready` are no longer counted. The panel should be renamed to "Currently Failed Runtimes" or supplemented with a true cumulative count from `unexpected_stops_total`.
- **Reconcile Duration is per queue item, not end-to-end.** `controller_runtime_reconcile_time_seconds` measures a single dequeue-process cycle. Because a full provisioning workflow spans many requeue cycles over several minutes, the per-cycle duration (typically seconds) does not reflect total time-to-Ready.
- **No alert thresholds defined in the dashboard.** The `unexpected_stops` panel has no alert rule attached. Operators rely on manual inspection.
- **`message` label content in the Runtime CR states table.** The table shows the free-text `message` field from the last status condition. This is human-readable but hard to filter or group in the dashboard. Adding a `reason` column would allow grouping by error category.

---

## 9. Gaps and Recommended Improvements

The following gaps exist against the requirements for full operator and L3 support observability.

### 9.1 Missing: End-to-End Provisioning Duration Metric

**Current state:** No metric tracks how long a full reconciliation cycle takes (time from `sFnTakeSnapshot` to `sFnUpdateStatus` completion).

**Impact:** Operators cannot detect hung or slow reconciliations without manually correlating log timestamps. There is no SLA metric for "time to Ready".

**Recommendation:** Add a `Histogram` or `Summary` metric:
```
infrastructure_manager_reconciliation_duration_seconds{runtimeId, provider, result}
```
where `result` is `success`, `failure`, or `requeue`. Record the value in `sFnUpdateStatus`.

### 9.2 Missing: Per-Step Error Counters

**Current state:** `unexpected_stops_total` is a single global counter with no labels. It is impossible to know which FSM step or error reason caused it without checking logs.

**Impact:** L3 support cannot extract statistics on failure distribution across steps (e.g., "how many failures are `SeedNotFound` vs `AuditLogErr`?").

**Recommendation:** Add labels to `unexpected_stops_total` or introduce a separate counter:
```
infrastructure_manager_reconciliation_errors_total{step, reason, provider}
```

### 9.3 Missing: Provisioning State Duration Metric

**Current state:** There is no gauge or histogram tracking how long a Runtime stays in `Pending` state.

**Impact:** It is difficult to distinguish a normally long provisioning from a stuck one without alerting on `state="Failed"`.

**Recommendation:** Track the `Pending` duration:
```
infrastructure_manager_runtime_pending_duration_seconds{runtimeId, provider}
```
Set on each requeue in `sFnWaitForShootCreation` / `sFnWaitForShootReconcile`, clear on `Ready` or `Failed`.

### 9.4 Limited: `message` Label on `im_runtime_state`

**Current state:** The `message` label on `infrastructure_manager_im_runtime_state` contains the last condition message. Prometheus labels with high cardinality and long free-text strings cause metric bloat and are difficult to query.

**Impact:** Structured error classification is not possible via PromQL. The `reason` field (which is a bounded enum) is not included.

**Recommendation:** Replace or supplement `message` with the last condition `reason` (bounded string):
```
infrastructure_manager_im_runtime_state{..., reason="SeedNotFound"}
```
This allows `count by (reason)` queries.

### 9.5 Missing: Structured Log Field for Error Category

**Current state:** Error log messages are human-readable strings. There is no machine-readable `errorReason` field in structured logs.

**Impact:** Log aggregation tools (e.g., Loki, Elasticsearch) cannot reliably extract error statistics without fragile regex parsing.

**Recommendation:** Add a structured `errorReason` field to all error log entries in the FSM, matching the condition reason enum values:
```go
m.log.Error(err, "Shoot creation failed", "runtimeID", s.instance.Name, "errorReason", string(imv1.ConditionReasonCreationError))
```

### 9.6 Missing: Runbook per Condition Reason

**Current state:** The dashboard and this document describe *what* the signals mean but there is no per-reason remediation guide.

**Impact:** L3 support must reason from first principles on each incident.

**Recommendation:** Add a `docs/operator/runbooks/` directory with one file per error reason (`seed-not-found.md`, `audit-log-err.md`, etc.), each containing: symptoms, probable causes, investigation steps, and fix actions.

### 9.7 Dashboard Fix: Rename "Cumulative Failed Runtimes"

**Current state:** The panel "Cumulative Failed Runtimes" queries the gauge `infrastructure_manager_im_runtime_state{state="Failed"}`, which reflects the *current* count, not a cumulative total. Runtimes that recovered are no longer counted.

**Impact:** The misleading name may cause operators to under-estimate the historical failure rate.

**Recommendation:** Rename to "Currently Failed Runtimes". Add a separate stat panel using `unexpected_stops_total` for a true cumulative failure count.

### 9.8 Dashboard Fix: Add `reason` Column to Runtime CR States Table

**Current state:** The "Runtime CR states" table shows the free-text `message` field. This is hard to filter or aggregate in the dashboard.

**Impact:** Operators cannot group failures by error category in the dashboard.

**Recommendation:** Extend the `infrastructure_manager_im_runtime_state` metric with a `reason` label (bounded enum, matches `ConditionReason` constants) and add a `Reason` column to the table. This enables `count by (reason)` aggregations.

### 9.9 Improvement: Gardener Last Errors in Conditions

**Current state:** Gardener `status.lastErrors` are logged as an Info message in `sFnSelectShootProcessing` but are **not propagated** into the Runtime CR's `status.conditions` message field.

**Impact:** Operators looking at `kubectl describe runtime` or the Plutono table panel do not see the root Gardener error. They must check KIM logs separately.

**Recommendation:** Propagate the Gardener `lastErrors` description into the `Provisioned` condition message when the Shoot is in a `Failed` or `Error` state. This is already partially done in `exposeShootStatusInfo` — verify it is consistently applied before all `updateStateFailed` calls. This also improves the `message` column in the dashboard table.

### 9.10 Improvement: Transition History

**Current state:** Kubernetes conditions only store the latest transition per type (`LastTransitionTime`). The full history of transitions is not retained.

**Impact:** For long-lived clusters that have experienced multiple failures and recoveries, the history is not visible in the CR or the dashboard.

**Recommendation:** Emit structured log events at `Info` level with a `conditionTransition` field for each condition change. This gives log aggregation tools a complete timeline without adding noise to the CR.

---

## 10. Summary — Current Coverage vs. Requirements

| Requirement | Coverage | Gap |
|-------------|----------|-----|
| Reconciliation process documented | Partial — architecture docs exist; no step-by-step operator runbook | §9.6: Add runbook per condition reason |
| Exceptions accessible to operators | Partial — `status.conditions` and K8s Events exist; Gardener errors not in conditions | §9.9: Propagate Gardener `lastErrors` to conditions |
| Structured logs | Partial — JSON structured with key fields; no per-step `errorReason` field | §9.5: Add `errorReason` label to error logs |
| Metrics for statistics extraction | Partial — state gauge and stop counter exist; no duration, no per-step counters | §9.1–9.4: Add duration histogram and labeled error counters |
| Dashboard exists | Yes — Plutono dashboard `R0wzpHTSk` covers Runtime states, GardenerCluster states, kubeconfig expiry, controller-runtime internals, and pod health | §9.7–9.8: Fix misleading panel title; add `reason` label |
| Dashboard: end-to-end provisioning duration | Partial — per-cycle `Reconcile Duration` panel exists; no full time-to-Ready metric | §9.1: Add end-to-end duration metric |
| Dashboard: failure breakdown by reason | Missing — `unexpected_stops_total` has no labels | §9.2: Add labeled error counter |
| Operator runbook | Missing | §9.6: Create runbook directory |

---

## 11. References

- [Plutono Dashboard: Infrastructure Manager](https://plutono.cp.kyma.cloud.sap/d/R0wzpHTSk/infrastructure-manager?orgId=1) (internal, requires SAP access)
- [KIM Configuration Parameters](./kim-configuration.md)
- [Cross-Cutting Concepts (Architecture)](../contributor/architecture/01-80-crosscutting-concepts.md)
- [Runtime View — State Sequences](../contributor/architecture/01-60-runtime-view.md)
- Source: `internal/controller/metrics/metrics.go`
- Source: `internal/controller/runtime/fsm/`
- Source: `api/v1/runtime_types.go`
