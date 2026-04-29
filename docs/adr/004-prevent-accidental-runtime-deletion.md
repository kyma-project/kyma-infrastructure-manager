---
name: Decision record
about: Decision record
---
<!-- Follow the decision making process (https://kyma-project.io/community/governance) -->

Created on 2026-04-22 by Tobias Schuhmacher (@tobiscr).

## Decision log

| Name | Description |
|-----------------------|------------------------------------------------------------------------------------|
| Title | Prevent accidental deletion of Kyma Runtime CRs via a validating admission webhook |
| Due date | 2026-05-15 |
| Status | Proposed on 2026-04-22 |
| Decision type | Choice |
| Affected decisions | None |

## Context

Kyma landscapes have been accidentally deleted in the past. Incidents were limited to the DEV landscape, but the same failure modes could affect customer-critical STAGE and PROD landscapes. Two distinct root causes have been observed:

**Human error:** SRE or on-call engineers execute an incorrect `kubectl delete` command, or a broad `kubectl delete` targeting the wrong resource type or namespace removes Runtime CRs as collateral damage.

**Software failure:** A cleanup or maintenance job contains a bug that removes all Runtime CRs in a landscape, or a job intended only for DEV is mistakenly deployed to STAGE or PROD and deletes all clusters there. An additional trigger is a CRD schema change that causes Kubernetes to cascade-delete existing Custom Resources.

In both cases the deletion reaches Kubernetes before any human or automated check can intervene, and KIM immediately begins deprovisioning the Gardener Shoot clusters. By the time the mistake is noticed, cluster deletion may already be irreversible.

### Requirements for a protection mechanism

1. **Two-step confirmation:** At least one explicit preparatory action (separate from the `kubectl delete` call itself) must be completed before a `Runtime` CR deletion is accepted. This prevents a single erroneous command from triggering deprovisioning.
2. **Rejection at the API level:** The deletion request must be refused by the Kubernetes API server before it reaches KIM. A controller-side finalizer alone is insufficient because a misconfigured or compromised controller could still process the deletion.
3. **Auditability:** Every rejection and every accepted deletion must produce an audit trail entry so incidents can be reconstructed.
4. **Minimal operational burden:** The confirmation step must be simple enough for a human to perform correctly under time pressure, and must be automatable by KEB for programmatic deletions.

### Options considered

#### Option 1: Controller-side finalizer only

The Runtime Controller already manages a finalizer (`runtime-controller.infrastructure-manager.kyma-project.io/deletion-hook`) that prevents the CR from disappearing until the Shoot is deleted. Adding a second, operator-controlled finalizer would mean the CR stays in a `Terminating` state until a human removes the second finalizer.

**Pros:**
- No additional Kubernetes components required; implemented entirely within KIM.
- Uses an established Kubernetes pattern.

**Cons:**
- A CR in `Terminating` state still triggers reconciliation; the Runtime Controller must be changed to distinguish between "intentional deletion that has been confirmed" and "accidental deletion that must be blocked".
- Does not satisfy requirement 2: the deletion event reaches KIM before it can be blocked. A bug or misconfiguration in KIM could still process the deletion.
- Does not satisfy requirement 1 cleanly: the only gate is removing the second finalizer, which is a single action.

#### Option 2: Validating admission webhook

A Kubernetes `ValidatingWebhookConfiguration` intercepts every `DELETE` request for `Runtime` objects before it is persisted. The webhook rejects the request with HTTP 403 and a human-readable message unless the Runtime CR carries a specific annotation added as a separate, prior action.

Proposed two-step protocol:
1. **Step 1 — annotate:** The caller (human or KEB) sets the annotation `operator.kyma-project.io/deletion-confirmed` on the Runtime CR to the current UTC timestamp in RFC 3339 format (e.g. `2026-04-29T14:00:00Z`) using a `kubectl annotate` or PATCH request.
2. **Step 2 — delete:** The caller issues the `kubectl delete runtime <name>` command within a short time window (default: 2 minutes) after setting the annotation. The webhook validates that the timestamp is not in the future, and that the current time falls within the configured acceptance window. Requests outside the window are rejected.

The webhook is the enforcement point; it runs in a separate process (or as a sub-handler in the KIM manager process) and has no dependency on the reconciliation loop.

**Pros:**
- Satisfies requirement 2: the API server rejects the deletion before it reaches etcd or any controller.
- Satisfies requirement 1: annotation and deletion are two distinct API calls; a single erroneous command cannot satisfy both.
- The time-window constraint makes the annotation self-expiring: once the window elapses the annotation is stale and a fresh annotation is required, which eliminates the retry-gap problem (a transient rejection does not leave a permanently valid annotation on the object).
- Future timestamps are rejected, preventing pre-staging of the annotation.
- Works regardless of the state of the KIM reconciler (e.g., even if the controller is temporarily down, the webhook still rejects unannotated deletions provided the webhook is reachable).
- Audit trail: every rejected deletion appears as a `403 Forbidden` response in the Kubernetes audit log; every accepted deletion carries the timestamp annotation in the audit record, making the confirmation time recoverable during post-mortems.
- Automatable: KEB can perform both steps programmatically with no user interaction.

**Cons:**
- Adds an operational dependency: if the webhook pod is unavailable and the `failurePolicy` is `Fail`, all Runtime deletions (including legitimate ones) are blocked. If `failurePolicy` is `Ignore`, the protection is bypassed during outages.
- Requires a TLS-secured HTTPS server, a `ValidatingWebhookConfiguration` resource, and a CA bundle rotation mechanism.
- Needs careful RBAC design to prevent the annotation from being added by any service account that also has delete permission (which would reduce the two-step requirement to a single automated step).
- The caller and the webhook server must have sufficiently synchronised clocks. A clock skew larger than the acceptance window would either block valid deletions or extend the window unintentionally. Mitigation: rely on NTP synchronisation, which is standard for Kubernetes nodes.

#### Option 3: OPA / Kyverno policy

An external policy engine (Open Policy Agent Gatekeeper or Kyverno) enforces the same annotation-before-delete rule as Option 2.

**Pros:**
- Policy logic is expressed in a high-level language (Rego or Kyverno YAML) rather than Go code, making it easier to audit and modify.
- Policy violations produce standardised reports.

**Cons:**
- Introduces a hard dependency on a third-party component (Gatekeeper or Kyverno) that must be deployed and maintained on every KCP cluster.
- Higher operational complexity: the policy engine itself becomes a critical infrastructure component that must be kept available.
- KIM should not require external policy engines for its own safety guarantees; the protection should be self-contained.

## Decision

The decision is to implement **Option 2: a Kubernetes validating admission webhook** embedded in the KIM manager process.

The webhook intercepts `DELETE` operations on `Runtime` objects (group `infrastructuremanager.kyma-project.io`, version `v1`, resource `runtimes`) and rejects them unless the Runtime CR carries the annotation:

```
operator.kyma-project.io/deletion-confirmed: "<RFC 3339 UTC timestamp>"
```

The annotation value must be a valid RFC 3339 UTC timestamp that:
1. Is **not in the future** (prevents pre-staging of the confirmation).
2. Falls **within the acceptance window** counted from the annotation timestamp to the moment the DELETE request arrives at the webhook (default: 2 minutes, configurable via a KIM manager flag).

If either condition is not met the webhook rejects the request with HTTP 403 and a message stating the reason (missing annotation, future timestamp, or expired window). The caller must re-annotate with the current time and retry.

This design inherently resolves the annotation-persistence-after-retry problem: an annotation set at time T is only valid until T + window. A transient rejection (e.g. webhook timeout) does not leave a permanently valid annotation; once the window elapses the annotation is stale and a fresh annotation is required.

### Implementation sketch

**Webhook handler** (`internal/webhook/runtime_deletion_webhook.go`):

```go
const DeletionConfirmedWindow = 2 * time.Minute

func (h *RuntimeDeletionWebhook) Handle(ctx context.Context, req admission.Request) admission.Response {
    if req.Operation != admissionv1.Delete {
        return admission.Allowed("")
    }

    var runtime imv1.Runtime
    if err := h.decoder.DecodeRaw(req.OldObject, &runtime); err != nil {
        return admission.Errored(http.StatusBadRequest, err)
    }

    raw, ok := runtime.Annotations[AnnotationRuntimeDeletionConfirmed]
    if !ok || raw == "" {
        return admission.Denied(
            "Runtime deletion requires the annotation " +
            AnnotationRuntimeDeletionConfirmed + " to be set to a UTC RFC 3339 timestamp " +
            "no more than " + DeletionConfirmedWindow.String() + " in the past.",
        )
    }

    ts, err := time.Parse(time.RFC3339, raw)
    if err != nil {
        return admission.Denied("annotation " + AnnotationRuntimeDeletionConfirmed +
            " is not a valid RFC 3339 timestamp: " + err.Error())
    }

    now := time.Now().UTC()
    if ts.After(now) {
        return admission.Denied("annotation " + AnnotationRuntimeDeletionConfirmed +
            " must not be a future timestamp")
    }

    if now.Sub(ts) > h.window {
        return admission.Denied("annotation " + AnnotationRuntimeDeletionConfirmed +
            " has expired (set at " + raw + ", window is " + h.window.String() + ")" +
            " — re-annotate with the current timestamp and retry")
    }

    return admission.Allowed("")
}
```

**Annotation constant** (`api/v1/runtime_types.go`):

```go
// AnnotationRuntimeDeletionConfirmed gates Runtime CR deletion at the KIM webhook layer.
// Its value must be a UTC RFC 3339 timestamp set immediately before the DELETE request.
// The webhook rejects the deletion if the timestamp is in the future or older than the
// configured acceptance window (default 2 minutes).
//
// This constant is distinct from AnnotationGardenerCloudDelConfirmation
// ("confirmation.gardener.cloud/deletion"), which gates Gardener Shoot deletion.
const AnnotationRuntimeDeletionConfirmed = "operator.kyma-project.io/deletion-confirmed"
```

**Registration** in `cmd/main.go`:

```go
mgr.GetWebhookServer().Register(
    "/validate-infrastructuremanager-kyma-project-io-v1-runtime",
    &webhook.Admission{Handler: &webhookhandler.RuntimeDeletionWebhook{...}},
)
```

**ValidatingWebhookConfiguration** (`config/webhook/`):

```yaml
apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingWebhookConfiguration
metadata:
  name: infrastructure-manager-validating-webhook
  annotations:
    cert-manager.io/inject-ca-from: kcp-system/infrastructure-manager-webhook-cert
webhooks:
  - name: vruntime.kb.io
    rules:
      - apiGroups: ["infrastructuremanager.kyma-project.io"]
        apiVersions: ["v1"]
        resources: ["runtimes"]
        operations: ["DELETE"]
        scope: Namespaced
    clientConfig:
      service:
        name: infrastructure-manager-webhook-service
        namespace: kcp-system
        path: /validate-infrastructuremanager-kyma-project-io-v1-runtime
      caBundle: "" # populated automatically by cert-manager ca-injector
    admissionReviewVersions: ["v1"]
    sideEffects: None
    failurePolicy: Fail
```

`failurePolicy: Fail` is the required setting for a safety mechanism: if the webhook is unreachable, deletions are blocked rather than allowed. The KIM webhook server must therefore be included in the KCP availability SLO.

### RBAC considerations

The `deletion-confirmed` annotation must not be freely settable by any service account that also holds the `delete` verb on `runtimes`. Otherwise, a single compromised or buggy service account could annotate and delete in one automated flow, reducing the two-step protocol to a single step.

- **KEB's service account** holds `patch`/`update` on `runtimes` (to apply the annotation) **and** `delete` permission. This is unavoidable for programmatic deletions. **Known limitation:** for the KEB path the two-step requirement is enforced procedurally (annotation must precede delete in KEB's workflow), not technically. The timestamp window mitigates the risk of annotation pre-staging (an annotation set long in advance will have expired by the time the delete is issued) but a bug or compromise in KEB that sets the annotation and immediately deletes within the window would bypass the intent of the two-step protocol. This limitation is accepted for the KEB path; future work may introduce a separate, annotation-only service account for KEB distinct from its delete credential.
- **SRE break-glass roles** should have `delete` permission but not `patch`/`update` on `runtimes` in STAGE/PROD landscapes, so that a human deletion always requires a second person with annotation rights to act first. This enforces the two-step protocol technically for the human access path.

## Consequences

**Advantages:**
- Accidental single-command deletions by humans or software are rejected before any deprovisioning begins.
- The protection is enforced at the Kubernetes API layer, independently of the KIM controller process; a bug in the KIM reconciler cannot bypass it.
- The mechanism is self-contained within KIM's deployment (no external policy engine dependency).
- Every rejected deletion is recorded in the Kubernetes API audit log, supporting incident post-mortems; the timestamp annotation makes the confirmation time recoverable.
- The time-window approach eliminates the annotation-persistence-after-retry problem: a stale annotation (outside the window) is rejected, forcing re-confirmation regardless of whether a previous deletion attempt succeeded or failed transiently.
- Future timestamps are rejected, preventing pre-staging of the annotation.

**Disadvantages and mitigations:**
- The KIM webhook server becomes a critical component on KCP: if it is unavailable (with `failurePolicy: Fail`), Runtime deletions are blocked. Mitigation: the webhook server runs inside the same manager process as the controllers, benefits from the same HA/leader-election setup, and should be included in KCP readiness checks.
- The time-window adds a deadline for the caller: the DELETE must follow the annotation within 2 minutes (configurable). Mitigation: the window is configurable; KEB automates both steps back-to-back; human operators follow a runbook that keeps the steps close together.
- Clock skew between the caller and the webhook server could cause spurious rejections if skew exceeds the window. Mitigation: rely on NTP synchronisation, which is standard for Kubernetes nodes; skew is expected to be well under one second.
- The two-step requirement for KEB is enforced only procedurally, not technically (KEB holds both annotation and delete permissions). Mitigation: the timestamp window limits the exposure period; future work may separate KEB's annotation credential from its delete credential.
