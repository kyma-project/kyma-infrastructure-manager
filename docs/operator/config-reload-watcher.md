# ConfigReloadWatcher

## Overview

The `ConfigReloadWatcher` is a controller that monitors changes to Kubernetes configuration resources (ConfigMaps, Secrets, and ClusterTrustBundles) on Kyma Control Plane (KCP). When a watched resource is modified, the controller triggers re-reconciliation of affected Runtime custom resources (CRs) by annotating them with a force-reconcile annotation.

This mechanism ensures that changes to shared configuration, such as API server Access Control List (ACL) rules or Runtime Bootstrapper settings, are propagated to all relevant Kyma runtimes without requiring manual intervention.

## Activation

The controller is registered only when at least one of the following feature flags is enabled:

- `--api-server-acl-enabled` - enables the Shoot API server ACL extender
- `--runtime-bootstrapper-enabled` - enables Runtime Bootstrapper

If neither flag is set, the controller is not created and no watches are registered.

## Watched Resources

The resources watched depend on the enabled feature.

- For `--runtime-bootstrapper-enabled`, the controller watches the following resources:

  | Resource Type | Name Source | Namespace |
  |:---|:---|:---|
  | ConfigMap | `--runtime-bootstrapper-kcp-config-name` | `kcp-system` |
  | ConfigMap | `--runtime-bootstrapper-manifests-config-map-name` | `kcp-system` |
  | ClusterTrustBundle | `--runtime-bootstrapper-kcp-cluster-trust-bundle` (optional) | cluster-scoped |
  | Secret | `--runtime-bootstrapper-kcp-pull-secret-name` (optional) | `kcp-system` |

  The ClusterTrustBundle and pull Secret watches are only registered when the corresponding flag value is non-empty.

- For `--api-server-acl-enabled`, the controller watches the following resource:

  | Resource Type | Name Source | Namespace |
  |:---|:---|:---|
  | ConfigMap | `converterConfig.kubernetes.kubeApiServer.acl.configMapName` | `kcp-system` |

## Reconciliation Flow

When a watched resource is updated, the following steps occur:

1. A watched resource (ConfigMap, Secret, or ClusterTrustBundle) is updated.
2. The `ObjectUpdatedPredicate` filters the event - only **Update** events with a matching name and namespace pass through.
3. The `ConfigReloadWatcher.Reconcile` method is called. It lists all Runtime CRs in the `kcp-system` namespace.
4. For each Runtime CR, two checks are applied:
   - The `RuntimePredicate` determines if this Runtime CR needs to be patched (see [Runtime Predicate](#runtime-predicate)).
   - If the Runtime CR already has the `force-patch-reconciliation` annotation, it is skipped.
5. Eligible Runtime CRs are patched through server-side apply with the annotation `operator.kyma-project.io/force-patch-reconciliation=true`.
6. The Runtime Controller detects the annotated CR and triggers a full cluster reconciliation.

## Runtime Predicate

Not all Runtimes need to react to every configuration change. The `RuntimePredicate` function determines whether a specific Runtime should be re-reconciled for a given configuration change:

- If the triggering resource is the ACL ConfigMap, the predicate calls `AclNeedsToBeEnabled`, which returns `true` only when all the following conditions are met:
  - `--api-server-acl-enabled` is `true`
  - The Runtime's provider type is AWS or Azure
  - The Runtime has a non-empty ACL AllowedCIDRs list

- For all other resources (Runtime Bootstrapper ConfigMaps, Secrets, ClusterTrustBundles), the predicate returns `true` - all Runtimes are re-reconciled.

## Event Filtering

The controller uses `ObjectUpdatedPredicate` to filter Kubernetes watch events. Only **Update** events are processed. The **Create**, ** Delete **, and ** Generic ** events are ignored.