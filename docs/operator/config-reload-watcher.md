# ConfigReloadWatcher

## Overview

The `ConfigReloadWatcher` is a controller that monitors changes to Kubernetes configuration resources (ConfigMaps, Secrets, and ClusterTrustBundles) on the Kyma Control Plane (KCP).When a watched resource is modified, the controller triggers re-reconciliation of affected Runtime CRs by annotating them with a force-reconcile annotation.

This mechanism ensures that changes to shared configuration, such as API server ACL rules or runtime bootstrapper settings, are propagated to all relevant Kyma runtimes without requiring manual intervention.

## Activation

The controller is registered only when at least one of the following feature flags is enabled:

- `--api-server-acl-enabled` - enables the shoot API server ACL extender
- `--runtime-bootstrapper-enabled` - enables the runtime bootstrapper

If neither flag is set, the controller is not created and no watches are registered.

## Watched Resources

Depending on which features are enabled, the controller watches different sets of resources:

### When `--runtime-bootstrapper-enabled` is set

| Resource Type | Name Source | Namespace |
|:---|:---|:---|
| ConfigMap | `--runtime-bootstrapper-kcp-config-name` | `kcp-system` |
| ConfigMap | `--runtime-bootstrapper-manifests-config-map-name` | `kcp-system` |
| ClusterTrustBundle | `--runtime-bootstrapper-kcp-cluster-trust-bundle` (optional) | cluster-scoped |
| Secret | `--runtime-bootstrapper-kcp-pull-secret-name` (optional) | `kcp-system` |

The ClusterTrustBundle and pull Secret watches are only registered when the corresponding flag value is non-empty.

### When `--api-server-acl-enabled` is set

| Resource Type | Name Source | Namespace |
|:---|:---|:---|
| ConfigMap | `converterConfig.kubernetes.kubeApiServer.acl.configMapName` | `kcp-system` |

## Reconciliation Flow

When a watched resource is updated, the following steps occur:

1. A watched resource (ConfigMap, Secret, or ClusterTrustBundle) is updated.
2. The `ObjectUpdatedPredicate` filters the event - only Update events with a matching name and namespace pass through.
3. The `ConfigReloadWatcher.Reconcile` method is called. It lists all Runtime CRs in the `kcp-system` namespace.
4. For each Runtime, two checks are applied:
   - The `RuntimePredicate` determines if this Runtime should react to the specific config change ([see below](#runtime-predicate)).
   - If the Runtime already has the `force-patch-reconciliation` annotation, it is skipped.
5. Eligible Runtimes are patched via server-side apply with the annotation `operator.kyma-project.io/force-patch-reconciliation=true`.
6. The Runtime Controller detects the annotated CR and triggers a full re-reconciliation.

## Runtime Predicate

Not all Runtimes need to react to every configuration change. The `RuntimePredicate` function determines whether a specific Runtime should be re-reconciled for a given configuration change:

- If the triggering resource is the ACL ConfigMap, the predicate calls `AclNeedsToBeEnabled`, which returns `true` only when:
  - `--api-server-acl-enabled` is `true`
  - The Runtime's provider type is AWS or Azure
  - The Runtime has a non-empty ACL AllowedCIDRs list

- For all other resources (runtime bootstrapper ConfigMaps, Secrets, ClusterTrustBundles), the predicate returns `true` - all Runtimes are re-reconciled.

## Event Filtering

The controller uses `ObjectUpdatedPredicate` to filter Kubernetes watch events. Only **Update** events are processed Create, Delete, and Generic events are ignored.