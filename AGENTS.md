# AGENTS.md — Kyma Infrastructure Manager

This file provides AI agents (Cursor, Copilot, Codex, etc.) with the context needed to contribute effectively to this repository.

---

## Project Overview

**Kyma Infrastructure Manager (KIM)** is a Kubernetes operator built with [kubebuilder](https://github.com/kubernetes-sigs/kubebuilder) that manages the Kyma cluster infrastructure on top of [Gardener](https://gardener.cloud/).

Core responsibilities:
- Provisioning and reconciling Gardener `Shoot` clusters via the `Runtime` CRD.
- Generating and rotating Secrets containing dynamic kubeconfigs via the `GardenerCluster` CRD.
- Bootstrapping runtime components (OIDC, ConfigMaps, registry cache, namespace creation, ClusterRoleBindings).

Module path: `github.com/kyma-project/infrastructure-manager`  
Go version: see `go.mod`  
FIPS build flag: `GOFIPS140=v1.0.0` is required for all builds.

---

## Repository Layout

```
api/v1/                          CRD types — Runtime, GardenerCluster
cmd/main.go                      Operator entry point
internal/
  controller/
    runtime/                     Runtime controller and FSM
      fsm/                       Finite-state machine states (one file per state)
    rtbootstrapperconfig/        RegistryCacheConfig controller
    metrics/                     Prometheus metrics
  log_level/                     Structured log-level helpers
  rtbootstrapper/                Runtime bootstrapper install logic
pkg/
  config/                        Operator configuration structs
  gardener/                      Gardener client, kubeconfig provider, shoot extenders
    shoot/extender/              Per-concern shoot spec extenders (DNS, OIDC, provider, networking…)
  provisioning/                  Provisioning helpers
  reconciler/                    Shared reconciler utilities
config/                          Kustomize manifests and CRD bases
docs/                            Architecture, ADRs, configuration reference
test/e2e/                        End-to-end tests (require a live cluster)
hack/                            Code generation helpers, boilerplate header
```

---

## Custom Resources

| CRD | Purpose |
|---|---|
| `Runtime` (`runtime_types.go`) | Desired state of a Gardener Shoot cluster. Drives the FSM-based reconciler. |
| `GardenerCluster` (`gardenercluster_types.go`) | Manages kubeconfig Secret generation and rotation for an existing Shoot. |

### Runtime States
`Pending` → `Ready` | `Failed` | `Terminating`

### Runtime Condition Types
`Provisioned`, `KubeconfigReady`, `OidcAndConfigMapConfigured`, `KymaSystemNSCreated`, `Configured`, `Deprovisioned`, `RegistryCacheConfigured`, `RuntimeBootstrapperReady`

---

## Architecture: FSM

The `Runtime` controller is driven by a **finite-state machine** (`internal/controller/runtime/fsm/`). Each state is a `stateFn` — a function with the signature:

```go
type stateFn func(context.Context, *fsm, *systemState) (stateFn, *ctrl.Result, error)
```

Each file in the `fsm/` package implements one or a small group of related states (e.g. `runtime_fsm_create_shoot.go`, `runtime_fsm_patch_shoot.go`). The FSM starts at `sFnTakeSnapshot` and transitions until a `nil` state is returned.

When adding a new state:
1. Create `runtime_fsm_<name>.go` with the state function and a corresponding `runtime_fsm_<name>_test.go`.
2. Wire the state into the FSM transition chain from an existing state.
3. Update condition types in `api/v1/runtime_types.go` if a new condition is introduced.

---

## Build & Run

```bash
# Build (requires FIPS flag)
make build

# Run locally against a cluster
make run

# Generate CRD manifests and DeepCopy methods
make manifests generate

# Format and vet
make fmt vet
```

The `make build` target automatically runs `manifests`, `generate`, `fmt`, and `vet` before compiling.

---

## Testing

```bash
# Unit + integration tests (uses envtest, no live cluster needed)
make test

# End-to-end tests (requires a live k3d/remote cluster)
KUBECONFIG_K3D=<path> go test ./test/e2e -v -timeout 30m
```

- Unit/integration tests live alongside source files (`*_test.go`) and use `envtest` for a local API server.
- Mocks are generated with `mockery` and live in `mocks/` subdirectories — do **not** edit them manually.
- E2E tests live in `test/e2e/` and run against a real cluster.
- Coverage is written to `coverage.txt`.

---

## Contributing Workflow

- All contributions must be submitted as pull requests from a **personal fork** of the repository — do not push feature branches directly to `kyma-project/kyma-infrastructure-manager`.
- Fork the repo, commit changes to a branch on your fork, then open a PR against `main` on the upstream repository.

---

## Code Conventions

- All new Go files must begin with the Apache-2.0 copyright header from `hack/boilerplate.go.txt`.
- Use `sigs.k8s.io/controller-runtime` patterns: reconcile via `ctrl.Result`, patch via `client.MergeFrom`, record events via `record.EventRecorder`.
- Errors: prefer `github.com/pkg/errors` for wrapping (`errors.Wrap`, `errors.Wrapf`).
- Logging: use `logr.Logger` (injected into FSM). Levels: `DEBUG` = 1, `TRACE` = 2 (see `internal/log_level`).
- Validation: struct-level validation uses `github.com/go-playground/validator/v10`.
- New shoot spec concerns go into `pkg/gardener/shoot/extender/` as a new extender file.
- Avoid adding direct `k8s.io/client-go` rest calls — use `controller-runtime` client abstractions.

---

## Dependencies

Key dependencies (see `go.mod` for exact versions):

| Package | Purpose |
|---|---|
| `github.com/gardener/gardener` | Gardener API types and client |
| `sigs.k8s.io/controller-runtime` | Operator framework |
| `github.com/kyma-project/lifecycle-manager/api` | Kyma module API types |
| `github.com/onsi/ginkgo/v2` + `gomega` | BDD-style test framework |
| `github.com/prometheus/client_golang` | Metrics |
| `github.com/go-playground/validator/v10` | Struct validation |

Dependency updates are managed by **Renovate** (`renovate.json`).

---

## Configuration Reference

Operator arguments (set in `config/default/manager_gardener_secret_patch.yaml`):

| Flag | Description | Default |
|---|---|---|
| `gardener-kubeconfig-path` | Path to Gardener project kubeconfig | — |
| `gardener-project-name` | Gardener project name | — |
| `minimal-rotation-time` | Ratio of `kubeconfig-expiration-time` before rotation starts | — |
| `kubeconfig-expiration-time` | Maximum time before kubeconfig rotation | — |
| `gardener-request-timeout` | Timeout for Gardener API requests | `3s` |
| `gardener-ctrl-reconcilation-timeout` | Timeout for GardenerCluster reconciliation | `60s` |
| `gardener-ratelimiter-qps` | Rate limiter QPS for Runtime controller | `5` |
| `gardener-ratelimiter-burst` | Rate limiter Burst for Runtime controller | `5` |
| `audit-log-mandatory` | Enforce audit log configuration | `true` |
| `runtime-ctrl-workers-cnt` | Parallel workers for Runtime controller | `25` |
| `gardener-cluster-ctrl-workers-cnt` | Parallel workers for GardenerCluster controller | `25` |
| `structured-auth-enabled` | Enable structured authentication | `false` |
| `registry-cache-config-controller-enabled` | Enable RegistryCacheConfig controller | `false` |

---

## Runtime Annotations (Troubleshooting)

| Annotation | Effect |
|---|---|
| `operator.kyma-project.io/force-patch-reconciliation: "true"` | Force next reconciliation into patch state regardless of generation number. Removed automatically after the attempt. |
| `operator.kyma-project.io/suspend-patch-reconciliation: "true"` | Prevents the controller from patching the Shoot. Must be removed manually to resume. |
| `operator.kyma-project.io/force-kubeconfig-rotation: "true"` | Forces kubeconfig Secret rotation before the scheduled time (on `GardenerCluster` CR). |

---

## Licensing

All source files must comply with [REUSE](https://reuse.software/) licensing. The project is licensed under **Apache-2.0** (see `LICENSES/Apache-2.0.txt`). The copyright header from `hack/boilerplate.go.txt` must appear at the top of every new `.go` file. The `REUSE.toml` file defines licensing for non-Go assets.
