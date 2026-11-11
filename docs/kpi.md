# KIM Key Performance Indicators (KPIs)

This document defines the Key Performance Indicators (KPIs) for the Kyma Infrastructure Manager (KIM). It specifies what each KPI measures, why it was chosen, how it is measured technically, and what actions must be taken when a threshold is breached.

---

## Purpose and Principles

These KPIs are designed to:

- Reflect **technical performance, reliability, scalability, and product health** in a way that maps directly to observable signals in production.
- Provide a **shared understanding** across the KIM team, stakeholders (Varban, Artur), and dependent teams (KEB, SRE, L3 support) so that every person interprets a KPI the same way.
- Define **operational action plans** — not just thresholds, but concrete steps owners must take when a KPI is breached.
- Support **continuous improvement and proactive issue identification** by making degradation visible before it becomes a user-facing incident.

KPIs are reviewed quarterly and updated whenever architectural changes affect what is measurable or what matters to the business.

---

## KPI Overview

| ID | KPI Name | Category | Primary Metric |
|----|----------|----------|----------------|
| KPI-1 | Provisioning Success Rate | Reliability | `im_runtime_state` |
| KPI-2 | Time to Ready (Provisioning Duration) | Performance | `infrastructure_manager_provisioning_duration_seconds` ¹ |
| KPI-3 | Failed Runtime Count | Reliability | `im_runtime_state{state="Failed"}` |
| KPI-4 | Unrecoverable FSM Stop Rate | Reliability | `unexpected_stops_total` |
| KPI-5 | Kubeconfig Rotation Success Rate | Security | `im_kubeconfig_expiration` + `im_gardener_clusters_state` |
| KPI-6 | Controller Reconcile Error Rate | Reliability | `controller_runtime_reconcile_errors_total` |
| KPI-7 | Work Queue Depth | Scalability | `workqueue_depth` |
| KPI-8 | KIM Pod Availability | Availability | Kubernetes Pod readiness |
| KPI-9 | Gardener API Error Rate | Reliability | `rest_client_requests_total` |
| KPI-10 | Pending Runtime Duration | Performance | `im_runtime_state{state="Pending"}` age |

¹ This metric does not yet exist. See the [gap in reconciliation-observability.md §9.1](./reconciliation-observability.md) for the implementation recommendation. Until it is implemented, Time to Ready is approximated from log timestamps.

---

## KPI Definitions

### KPI-1 — Provisioning Success Rate

**Definition:** The percentage of cluster provisioning requests that reach `status.state = Ready` within the agreed time window, out of all provisioning attempts that have reached a terminal state (`Ready` or `Failed`).

**Why this KPI:** This is the primary end-user-visible outcome of KIM. Every time a customer provisions a Kyma runtime via BTP, KIM is responsible for delivering a functional cluster. A declining success rate directly translates to failed customer orders, SLA violations, and escalations to management. It is the single most important indicator of product health.

**Formula:**
```
Provisioning Success Rate =
  count(Runtimes that transitioned to Ready in the window)
  ─────────────────────────────────────────────────────── × 100
  count(Runtimes that reached Ready or Failed in the window)
```

**How to measure:**

Using Prometheus (once KPI-2's metric is available):
```promql
# Success count (result="success")
sum(increase(infrastructure_manager_provisioning_duration_seconds_count{result="success"}[24h]))
/
# Total terminal outcomes
sum(increase(infrastructure_manager_provisioning_duration_seconds_count[24h]))
* 100
```

Until the duration histogram is available, this KPI cannot be derived from `Reconciliation done` logs: that entry is emitted at debug level (unavailable in production) and carries no `state` field. Instead, derive terminal transitions from Kubernetes Events on the Runtime CR (`Normal ConfigurationCompleted` → Ready, `Warning` with a permanent reason → Failed), or add a dedicated terminal-transition log/metric before operationalizing this KPI.

**Thresholds:**

| Level | Value | Meaning |
|-------|-------|---------|
| Target | ≥ 99.5% | Normal operations |
| Warning | 95.0% – 99.4% | Degraded; investigation required |
| Critical | < 95.0% | Incident; immediate action required |

**Actions when Warning threshold is breached:**
1. Check `infrastructure_manager_im_runtime_state{state="Failed"}` to identify which Runtimes are failing.
2. Use `kubectl describe runtime <name>` to read `status.conditions` and identify the failing condition reason.
3. Check whether failures are clustered by provider, region, or tenant — this distinguishes infrastructure problems from configuration problems.
4. Review KIM logs for recurring error patterns: `kubectl logs -n kcp-system -l control-plane=infrastructure-manager | grep -E "(error|AuditLogErr|SeedNotFound|ConversionErr)"`.
5. Assign an owner and open a tracking issue within 4 hours.

**Actions when Critical threshold is breached:**
1. Trigger a P1 incident in the on-call rotation immediately.
2. Check whether the issue is systemic (all regions/providers) or isolated (single seed/tenant). If systemic, escalate to Gardener team.
3. Evaluate whether to temporarily suspend new provisioning by coordinating with KEB team.
4. Prepare a post-incident review within 48 hours of resolution.

---

### KPI-2 — Time to Ready (Provisioning Duration)

**Definition:** The p50 and p95 end-to-end provisioning duration — the time from when a Runtime CR first enters `Pending` state to when it reaches `Ready` state.

**Why this KPI:** Cluster provisioning involves multiple sequential steps across KIM, Gardener, and the Kyma runtime. Slow provisioning erodes customer experience even when provisioning eventually succeeds. The p95 value captures outliers — long tail provisioning is often a signal of seed resource contention, misconfiguration, or stuck bootstrapper installations.

**How to measure:**

Once `infrastructure_manager_provisioning_duration_seconds{result, provider}` is implemented:
```promql
# p50 provisioning time (seconds)
histogram_quantile(0.50, sum by (le) (rate(infrastructure_manager_provisioning_duration_seconds_bucket{result="success"}[1h])))

# p95 provisioning time (seconds)
histogram_quantile(0.95, sum by (le) (rate(infrastructure_manager_provisioning_duration_seconds_bucket{result="success"}[1h])))
```

Until the metric is available, derive terminal transitions from Kubernetes Events on the Runtime CR (see KPI-1 note on log fallback).

**Reference provisioning duration breakdown (expected):**

| Phase | Expected duration |
|-------|------------------|
| Shoot creation to Pending | ~2 min |
| Gardener Shoot reconciliation | 10–20 min |
| Bootstrapper installation | 2–5 min |
| Total (expected) | 15–30 min |

**Thresholds:**

| Level | p50 | p95 |
|-------|-----|-----|
| Target | ≤ 20 min | ≤ 40 min |
| Warning | 20–30 min | 40–60 min |
| Critical | > 30 min | > 60 min |

**Actions when Warning threshold is breached:**
1. Check `workqueue_depth{name="runtime"}` — sustained high depth indicates worker saturation; consider increasing `--runtime-ctrl-workers-cnt`.
2. Check `rate(rest_client_requests_total{code=~"4.."}[5m])` — rising 4xx rates suggest Gardener rate limiting; review `--gardener-ratelimiter-qps`.
3. Identify which provider is contributing most to the slowdown via the Runtime CR states table in the Plutono dashboard.
4. Check Gardener seed status for the affected regions.

**Actions when Critical threshold is breached:**
1. Trigger a P2 incident.
2. Check for seed saturation in Gardener.
3. Identify if the bootstrapper is stalling: check `infrastructure_manager_im_runtime_state` for runtimes with `RuntimeBootstrapperInstallationInProgress` remaining > 30 minutes.
4. If worker saturation is the cause, update `--runtime-ctrl-workers-cnt` and restart the KIM pods as a temporary mitigation.

---

### KPI-3 — Failed Runtime Count

**Definition:** The number of Runtime CRs currently in `status.state = Failed` (i.e., non-recoverable error, no automatic retry).

**Why this KPI:** Every Runtime in `Failed` state represents a cluster that a customer cannot use and that requires explicit human intervention to recover. Unlike a transient error (which self-heals), `Failed` is a terminal state — it does not decrease on its own. A non-zero count is always an actionable signal.

**How to measure:**
```promql
sum(infrastructure_manager_im_runtime_state{state="Failed"})
```

This is the "Cumulative Failed Runtimes" panel in the Plutono dashboard (note: despite the panel name, this shows the *current* count, not a cumulative total — see [reconciliation-observability.md §9.7](./reconciliation-observability.md)).

**Thresholds:**

| Level | Value | Meaning |
|-------|-------|---------|
| Target | 0 | No failures in terminal state |
| Warning | 1–3 | Isolated failures; remediation within SLA |
| Critical | > 3 | Systemic issue; escalate immediately |

**Actions when Warning threshold is breached:**
1. For each failed Runtime, run `kubectl describe runtime <name> -n kcp-system` and read `status.conditions`.
2. Determine the condition reason:
   - `SeedNotFound` → check seed availability; update Runtime spec region; coordinate with Gardener team.
   - `AuditLogErr` → add tenant mapping to converter config; restart KIM.
   - `ConversionErr` / `CreationErr` → check converter config and Runtime spec; fix and re-trigger by patching the CR.
   - `OidcConfigurationErr` / `ConfigMapErr` → check OIDC config; verify kubeconfig Secret is valid.
3. Notify the owning team of the failed Runtime (typically KEB creates them; track by `kyma-project.io/runtime-id` label).
4. Record resolution in the tracking issue.

**Actions when Critical threshold is breached:**
1. Trigger P1 incident.
2. Determine if failures share a common reason (systemic) or are isolated.
3. If systemic (`SeedNotFound` in a specific region, or all `AuditLogErr`), escalate to the responsible sub-team and block new provisioning in that region/configuration via KEB until resolved.

---

### KPI-4 — Unrecoverable FSM Stop Rate

**Definition:** The per-second rate of `unexpected_stops_total` counter increments, measured over a 5-minute sliding window.

**Why this KPI:** `unexpected_stops_total` is incremented on instrumented FSM stop and error paths. Note that not every increment represents a permanent, non-retriable failure — some retrying error paths (such as a namespace creation failure followed by a requeue) also increment this counter. Nevertheless, a sustained increasing rate is the earliest systemic signal that something is wrong at the controller level — before it shows up as individual failed Runtimes. This KPI is the primary early-warning indicator; correlate with KPI-3 to distinguish permanent failures from elevated transient retries.

**How to measure:**
```promql
rate(unexpected_stops_total{job="infrastructure-manager-metrics"}[5m])
```

This is the "Unexpected Runtime processing stops" panel in the Plutono dashboard. The annotation on the panel states: "should be always 0".

**Thresholds:**

| Level | Rate | Meaning |
|-------|------|---------|
| Target | 0 / s | No unrecoverable stops |
| Warning | > 0 / s sustained for > 5 min | Ongoing failures; investigate |
| Critical | > 0.1 / s sustained for > 5 min | High volume failure rate; incident |

**Actions when Warning threshold is breached:**
1. Check KIM logs for the error paths being hit: `kubectl logs -n kcp-system -l control-plane=infrastructure-manager | jq 'select(.level=="error")'`.
2. Correlate with KPI-3: are Failed Runtime counts also increasing?
3. Check whether the rate is concentrated in a specific pod (pod-local issue) or all pods (systemic).
4. If the rate is new and coincides with a recent KIM deployment, consider rolling back.

**Actions when Critical threshold is breached:**
1. Trigger P1 incident immediately.
2. Check if a recent deployment introduced a regression.
3. Evaluate rollback of the last KIM deployment.

---

### KPI-5 — Kubeconfig Rotation Success Rate

**Definition:** The percentage of kubeconfigs that are successfully rotated before their expiry deadline, measured as the proportion of kubeconfigs that never reach an expiration state.

**Why this KPI:** Kubeconfig rotation is a core security requirement (QS-S1). An expired kubeconfig means the corresponding Kyma cluster is inaccessible from the control plane. This is a direct security and availability violation. The GardenerCluster controller must rotate kubeconfigs before expiry; failure to do so within the rotation window is a KIM defect.

**How to measure:**

Monitor kubeconfigs approaching expiry without having been rotated:
```promql
# Kubeconfigs expiring within the next 2 hours (warning)
count(infrastructure_manager_im_kubeconfig_expiration < (time() + 7200))

# Kubeconfigs already expired
count(infrastructure_manager_im_kubeconfig_expiration < time())
```

Rotation health is also visible via `infrastructure_manager_im_gardener_clusters_state`:
```promql
# GardenerClusters not in a healthy state
sum(infrastructure_manager_im_gardener_clusters_state) by (state)
```

**Thresholds:**

| Level | Condition | Meaning |
|-------|-----------|---------|
| Target | 0 kubeconfigs expiring within rotation window | All rotations on schedule |
| Warning | ≥ 1 kubeconfig expiring within 2 hours | Rotation delayed; investigate |
| Critical | ≥ 1 kubeconfig expired (`expiration < time()`) | Security violation; immediate action |

**Actions when Warning threshold is breached:**
1. Identify the affected runtimes: `infrastructure_manager_im_kubeconfig_expiration < (time() + 7200)` — note the `runtimeId` and `shoot` labels.
2. Check the GardenerCluster CR for the affected runtime: `kubectl describe gardenercluster <name> -n kcp-system`.
3. Check KIM logs for GardenerCluster controller errors: filter by shoot name.
4. Check whether the GardenerCluster controller workers are saturated: `workqueue_depth{name="gardenercluster"}`.
5. If a specific GardenerCluster CR is stuck, trigger manual re-reconciliation by patching an annotation.

**Actions when Critical threshold is breached:**
1. Trigger P1 security incident immediately.
2. Manually trigger kubeconfig rotation by adding the `operator.kyma-project.io/force-kubeconfig-rotation: "true"` annotation to the affected GardenerCluster CR. This uses the controller's normal reconciliation path to fetch and publish the replacement, avoiding credential gaps that could occur from deleting the Secret directly.
3. Conduct a post-incident review to understand why the rotation window was missed.

---

### KPI-6 — Controller Reconcile Error Rate

**Definition:** The per-second rate of reconciliation loops (across all controllers) that returned an error, causing a requeue.

**Why this KPI:** A sustained non-zero reconcile error rate means controllers are retrying work repeatedly without making progress. This can indicate a persistent external problem (Gardener API unavailable, permission issue) or a KIM bug. Unlike `unexpected_stops_total` (which counts permanent failures), reconcile errors also include transient failures. A baseline non-zero rate is expected during normal operations (e.g., Shoot still reconciling); the KPI targets a sustained elevated rate.

**How to measure:**
```promql
# Error rate per controller
sum by (controller) (
  rate(controller_runtime_reconcile_errors_total{job="infrastructure-manager-metrics"}[5m])
)
```

**Thresholds:**

| Level | Rate per controller | Meaning |
|-------|---------------------|---------|
| Target | < 0.05 / s | Low baseline (transient retries) |
| Warning | 0.05–0.2 / s sustained > 10 min | Elevated retry rate; investigate |
| Critical | > 0.2 / s sustained > 10 min | Persistent failure loop; incident |

**Actions when Warning threshold is breached:**
1. Identify which controller (`runtime` or `gardenercluster`) has elevated error rate.
2. Check `rest_client_requests_total` grouped by HTTP code — rising 4xx/5xx rates indicate API connectivity issues.
3. Check Gardener API availability from within the KCP cluster.
4. Review rate limiter settings: `--gardener-ratelimiter-qps` and `--gardener-ratelimiter-burst`.

**Actions when Critical threshold is breached:**
1. Trigger P2 incident.
2. If Gardener API is unreachable, coordinate with the Gardener team.
3. If the error rate is on the `gardenercluster` controller only, check whether kubeconfig rotation is being blocked by a specific API permission issue.

---

### KPI-7 — Work Queue Depth

**Definition:** The number of items waiting in the reconciliation work queue for the `runtime` and `gardenercluster` controllers at any given moment.

**Why this KPI:** Queue depth is the primary scalability signal. When the queue grows faster than workers can drain it, new provisioning requests are delayed and customer-visible latency increases. A persistently non-zero queue depth means KIM is operating above its steady-state processing capacity. This KPI directly informs scaling decisions.

**How to measure:**
```promql
workqueue_depth{name="runtime"}
workqueue_depth{name="gardenercluster"}
```

Also consider queue wait latency:
```promql
rate(workqueue_queue_duration_seconds_sum{name="runtime"}[5m])
/ rate(workqueue_queue_duration_seconds_count{name="runtime"}[5m])
```

**Thresholds:**

| Level | Depth | Meaning |
|-------|-------|---------|
| Target | 0–5 | Normal; small transient bursts acceptable |
| Warning | 6–25 | Growing backlog; monitor closely |
| Critical | > 25 | Workers saturated; provisioning is delayed |

**Actions when Warning threshold is breached:**
1. Monitor whether the queue is draining (transient burst) or growing (sustained saturation).
2. Check `controller_runtime_max_concurrent_reconciles` — verify worker count matches expectations.
3. Check CPU and memory usage of KIM pods. If resource-constrained, adjust Pod resource limits.
4. Check whether a specific provider or region is contributing a disproportionate number of items (burst provisioning event).

**Actions when Critical threshold is breached:**
1. Trigger P2 incident if the queue does not drain within 30 minutes of mitigation.
2. Increase `--runtime-ctrl-workers-cnt` and restart KIM pods.
3. If a batch provisioning event is the cause, coordinate with KEB to stagger requests.

---

### KPI-8 — KIM Pod Availability

**Definition:** The percentage of the rolling 24-hour window during which at least one KIM Pod passes its readiness probe.

**Why this KPI:** KIM is a single-point-of-automation for all Kyma cluster provisioning and kubeconfig rotation. If all KIM pods are unavailable, no new clusters can be provisioned, no kubeconfigs are rotated (introducing a security risk over time), and no cluster updates are applied. Pod availability is the foundation on which all other KPIs depend.

**How to measure:**

Compute availability from a recording rule that tracks whether any KIM pod is ready:
```promql
# Rolling 24-hour availability (requires a recording rule: kim_any_pod_ready)
avg_over_time(kim_any_pod_ready[24h:]) * 100
```

Use the instantaneous readiness query separately for outage alerting:
```promql
# Instantaneous: 1 = at least one pod ready, 0 = full outage
max(kube_pod_status_ready{namespace="kcp-system", pod=~"infrastructure-manager-.*", condition="true"})
```

The recording rule `kim_any_pod_ready` should evaluate `max(kube_pod_status_ready{namespace="kcp-system", pod=~"infrastructure-manager-.*", condition="true"})` at a 1-minute interval.

**Thresholds:**

| Level | Condition | Meaning |
|-------|-----------|---------|
| Target | ≥ 1 pod ready at all times | Full availability |
| Warning | Brief gap < 5 min (e.g., rolling update) | Expected; monitor |
| Critical | No ready pod for > 5 min | Full outage; immediate action |

**Actions when Warning threshold is breached:**
1. Check pod status: `kubectl get pods -n kcp-system -l control-plane=infrastructure-manager`.
2. Check pod logs for crash reasons: `kubectl logs -n kcp-system <pod-name> --previous`.
3. Check whether a rolling update is in progress — expected disruption is acceptable.
4. Verify liveness probe endpoint `:8081/healthz` responds.

**Actions when Critical threshold is breached:**
1. Trigger P1 incident immediately.
2. Check node health for the nodes hosting KIM pods.
3. Check for OOMKill events: `kubectl describe pod -n kcp-system <pod-name>`.
4. If the pod cannot start, roll back to the last known-good image.
5. Notify all dependent teams (KEB, SRE) that provisioning is suspended until recovery.

---

### KPI-9 — Gardener API Error Rate

**Definition:** The rate of 4xx and 5xx HTTP responses received by KIM from the Gardener API server, as a percentage of total outbound requests to Gardener.

**Why this KPI:** KIM's entire function depends on a healthy connection to the Gardener API. 4xx errors typically indicate permission, quota, or request-formation problems (usually KIM bugs or misconfiguration). 5xx errors indicate Gardener-side issues (outages, overload). Both have different remediation paths. Monitoring this rate provides early warning before Gardener issues cascade into provisioning failures.

**How to measure:**
```promql
# 4xx error rate to Gardener
rate(rest_client_requests_total{job="infrastructure-manager-metrics", host=~".*gardener.*", code=~"4.."}[5m])
/
rate(rest_client_requests_total{job="infrastructure-manager-metrics", host=~".*gardener.*"}[5m])

# 5xx error rate to Gardener
rate(rest_client_requests_total{job="infrastructure-manager-metrics", host=~".*gardener.*", code=~"5.."}[5m])
/
rate(rest_client_requests_total{job="infrastructure-manager-metrics", host=~".*gardener.*"}[5m])
```

**Thresholds:**

| Level | 4xx rate | 5xx rate | Meaning |
|-------|----------|----------|---------|
| Target | < 1% | < 0.5% | Normal |
| Warning | 1–5% | 0.5–2% | Elevated; investigate |
| Critical | > 5% | > 2% | Systemic; incident |

**Actions when Warning threshold is breached (4xx dominant):**
1. Check for recent KIM deployments that changed API call patterns.
2. Check whether the Gardener service account credentials (`--gardener-kubeconfig-path`) are still valid.
3. Review specific request paths causing 4xx via KIM logs: `kubectl logs | grep "status.*4[0-9][0-9]"`.
4. Check for Gardener quota exhaustion in the project namespace.

**Actions when Warning threshold is breached (5xx dominant):**
1. Check Gardener cluster health via the Gardener dashboard or SRE contacts.
2. Review `--gardener-request-timeout` — if Gardener is slow, consider increasing it temporarily.
3. The KIM rate limiter (`--gardener-ratelimiter-qps`) will back off automatically; no immediate action needed unless the rate persists.

**Actions when Critical threshold is breached:**
1. Trigger P1 incident and contact the Gardener team.
2. If Gardener is confirmed unavailable, suspend new provisioning requests in coordination with KEB.

---

### KPI-10 — Pending Runtime Duration

**Definition:** The number of Runtime CRs that have been in `status.state = Pending` for longer than the expected provisioning window (> 40 minutes).

**Why this KPI:** A Runtime stuck in `Pending` for an abnormally long time is either waiting on a slow Gardener operation or has encountered a state from which it cannot progress but has not yet entered `Failed`. This is the "silent failure" scenario — KIM is not reporting an error, but the customer's cluster is not ready. This KPI catches these cases proactively.

**How to measure:**

Currently approximated by correlating log timestamps (when `state=Pending` was first set) against current time. Once the end-to-end duration metric (KPI-2) is available:

```promql
# Runtimes stuck in Pending > 40 minutes (derived once duration metric exists)
infrastructure_manager_runtime_pending_duration_seconds > 2400
```

Until then: alert on Runtimes in `Pending` state combined with absence of recent log activity for that runtimeID over a 40-minute window.

**Thresholds:**

| Level | Stuck Runtimes | Meaning |
|-------|----------------|---------|
| Target | 0 | All provisioning within expected duration |
| Warning | ≥ 1 Runtime Pending > 40 min | Potentially stuck; investigate |
| Critical | ≥ 1 Runtime Pending > 90 min | Definitely stuck; incident |

**Actions when Warning threshold is breached:**
1. Identify the stuck Runtime from the Plutono dashboard "Runtime CR states" table.
2. Run `kubectl describe runtime <name> -n kcp-system` and check the last condition transition time and reason.
3. Check the corresponding Gardener Shoot: `kubectl get shoot <shoot-name> -n <project-namespace> -o jsonpath='{.status.lastOperation}'`.
4. Check whether the reconciliation suspension annotation is present: `operator.kyma-project.io/suspend-patch-reconciliation`.
5. If the Shoot is stuck in Gardener, escalate to the Gardener team.
6. If the bootstrapper is stuck (`RuntimeBootstrapperInstallationInProgress` > 30 min), check bootstrapper logs on the Kyma runtime.

**Actions when Critical threshold is breached:**
1. Trigger P2 incident.
2. Follow the investigation steps above.
3. If Gardener has no information about the Shoot or the Shoot is in `Error` state, consider deleting the Runtime CR and re-provisioning (coordinate with KEB team).

---

## Measurement Cadence and Review

| Activity | Frequency | Owner |
|----------|-----------|-------|
| KPI dashboard review | Weekly (team sync) | KIM team lead |
| Threshold review | Quarterly | KIM team + Varban + Artur |
| Alert rule validation | Each KIM release | On-call engineer |
| KPI document update | As architecture changes | PR author + reviewer |

---

## Missing Metrics — Implementation Priority

Several KPIs above reference metrics that do not yet exist. Their implementation is tracked in [reconciliation-observability.md §9](./reconciliation-observability.md).

| Missing Metric | Affects KPIs | Priority |
|----------------|-------------|----------|
| `infrastructure_manager_provisioning_duration_seconds` | KPI-1, KPI-2 | High |
| `infrastructure_manager_runtime_pending_duration_seconds` | KPI-10 | Medium |
| `infrastructure_manager_reconciliation_errors_total{step, reason, provider}` | KPI-4 | Medium |
| `reason` label on `infrastructure_manager_im_runtime_state` | KPI-3, KPI-4 | Low |

Until these metrics are implemented, the affected KPIs are measured via log analysis or approximated using available signals as described in each KPI section.

---

## Stakeholder Alignment

This document reflects input from the following stakeholders:

| Stakeholder | Role | Interest |
|-------------|------|---------|
| Varban | Management | Business impact, escalation thresholds, customer SLA alignment |
| Artur | Management | Product health, roadmap prioritization, incident response expectations |
| KIM team | Engineering | Technical accuracy, measurability, operational feasibility |
| KEB team | Dependent service | Provisioning success rate and duration directly affect KEB's own SLA |
| SRE / Platform Ops | Operations | Alert fatigue, actionability, on-call response clarity |
| L3 Support | Incident response | Failure breakdown, runbook availability, signal-to-noise ratio |

KPI thresholds marked as "Target" represent the agreed operating standard. Breaching a "Warning" threshold does not require escalation to management unless the condition persists beyond the stated action window. Breaching a "Critical" threshold triggers the standard incident process; stakeholder communication is handled by the on-call engineer on a case-by-case basis.

---

## References

- [Reconciliation Observability Reference](./reconciliation-observability.md)
- [KIM Configuration Parameters](./kim-configuration.md)
- [Architecture: Quality Requirements](../contributor/architecture/01-100-quality-requirements.md)
- [Architecture: Introduction and Goals](../contributor/architecture/01-10-introduction-and-goals.md)
- [Plutono Dashboard: Infrastructure Manager](https://plutono.cp.kyma.cloud.sap/d/R0wzpHTSk/infrastructure-manager) (internal, requires SAP access)
- Metrics source: `internal/controller/metrics/metrics.go`
