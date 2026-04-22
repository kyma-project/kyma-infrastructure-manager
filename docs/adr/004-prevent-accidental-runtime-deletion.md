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
1. **Step 1 — annotate:** The caller (human or KEB) adds the annotation `operator.kyma-project.io/deletion-confirmed: "true"` to the Runtime CR using a `kubectl annotate` or PATCH request.
2. **Step 2 — delete:** The caller issues the `kubectl delete runtime <name>` command. The webhook reads the annotation from the persisted object and allows the deletion.

The webhook is the enforcement point; it runs in a separate process (or as a sub-handler in the KIM manager process) and has no dependency on the reconciliation loop.

**Pros:**
- Satisfies requirement 2: the API server rejects the deletion before it reaches etcd or any controller.
- Satisfies requirement 1: annotation and deletion are two distinct API calls; a single erroneous command cannot satisfy both.
- Works regardless of the state of the KIM reconciler (e.g., even if the controller is temporarily down, the webhook still rejects unannotated deletions provided the webhook is reachable).
- Audit trail: every rejected deletion appears as a `403 Forbidden` response in the Kubernetes audit log; every accepted deletion carries the annotation in the audit record.
- Automatable: KEB can perform both steps programmatically with no user interaction.

**Cons:**
- Adds an operational dependency: if the webhook pod is unavailable and the `failurePolicy` is `Fail`, all Runtime deletions (including legitimate ones) are blocked. If `failurePolicy` is `Ignore`, the protection is bypassed during outages.
- Requires a TLS-secured HTTPS server, a `ValidatingWebhookConfiguration` resource, and a CA bundle rotation mechanism.
- Needs careful RBAC design to prevent the annotation from being added by any service account that also has delete permission (which would reduce the two-step requirement to a single automated step).

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
operator.kyma-project.io/deletion-confirmed: "true"
```

The annotation must have been applied in a separate API call prior to the delete request. After a successful deletion the annotation is irrelevant (the object is gone); if the deletion is retried after the object is restored, the annotation must be re-applied.

### Implementation sketch

**Webhook handler** (`internal/webhook/runtime_deletion_webhook.go`):

```go
func (h *RuntimeDeletionWebhook) Handle(ctx context.Context, req admission.Request) admission.Response {
    if req.Operation != admissionv1.Delete {
        return admission.Allowed("")
    }

    var runtime imv1.Runtime
    if err := h.decoder.DecodeRaw(req.OldObject, &runtime); err != nil {
        return admission.Errored(http.StatusBadRequest, err)
    }

    if runtime.Annotations[AnnotationDeletionConfirmed] != "true" {
        return admission.Denied(
            "Runtime deletion requires the annotation " +
            AnnotationDeletionConfirmed + "=true to be set before deleting the CR. " +
            "Apply the annotation first, then retry the deletion.",
        )
    }

    return admission.Allowed("")
}
```

**Annotation constant** (`api/v1/runtime_types.go`):

```go
const AnnotationDeletionConfirmed = "operator.kyma-project.io/deletion-confirmed"
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
    admissionReviewVersions: ["v1"]
    sideEffects: None
    failurePolicy: Fail
```

`failurePolicy: Fail` is the required setting for a safety mechanism: if the webhook is unreachable, deletions are blocked rather than allowed. The KIM webhook server must therefore be included in the KCP availability SLO.

### RBAC considerations

The `deletion-confirmed` annotation must not be settable by any service account that also holds the `delete` verb on `runtimes`. Otherwise, a single compromised or buggy service account could annotate and delete in one automated flow, bypassing the two-step intent. The following rule applies:

- KEB's service account has `patch`/`update` permission on `runtimes` (to apply the annotation) **and** `delete` permission. This is unavoidable for programmatic deletions. The two-step requirement is enforced procedurally for KEB (annotation must precede delete in its workflow) and technically for ad-hoc human access (SRE roles must not have both permissions simultaneously).
- SRE break-glass roles should have `delete` permission but not `patch`/`update` on `runtimes` in STAGE/PROD landscapes, so that a human deletion always requires a second person with annotation rights to act first.

## Consequences

**Advantages:**
- Accidental single-command deletions by humans or software are rejected before any deprovisioning begins.
- The protection is enforced at the Kubernetes API layer, independently of the KIM controller process; a bug in the KIM reconciler cannot bypass it.
- The mechanism is self-contained within KIM's deployment (no external policy engine dependency).
- Every rejected deletion is recorded in the Kubernetes API audit log, supporting incident post-mortems.

**Disadvantages and mitigations:**
- The KIM webhook server becomes a critical component on KCP: if it is unavailable (with `failurePolicy: Fail`), Runtime deletions are blocked. Mitigation: the webhook server runs inside the same manager process as the controllers, benefits from the same HA/leader-election setup, and should be included in KCP readiness checks.
- TLS certificate management is required for the webhook endpoint. Mitigation: use `cert-manager` (already used in the Kyma ecosystem) to issue and rotate the webhook server certificate automatically.
- The two-step requirement adds friction for legitimate deletions. Mitigation: KEB automates both steps; human operators follow a runbook that makes the annotation step explicit and deliberate.
- Protecting a restored Runtime CR after accidental re-creation requires re-applying the annotation. Mitigation: document this in the operator runbook.
