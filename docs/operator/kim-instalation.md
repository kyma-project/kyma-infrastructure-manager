# Setting Up Kyma Infrastructure Manager

## Context

Kyma Infrastructure Manager (KIM) is a Kubernetes Operator that manages the lifecycle of an SAP BTP, Kyma runtime. It uses SAP Gardener to create, update, and delete the underlying Kubernetes cluster infrastructure components. KIM translates the Kyma runtime configuration into a Gardener `Shoot` definition and monitors the cluster's reconciliation process, ensuring the cluster is properly prepared to run the Kyma software.

Additionally, KIM manages secure cluster access from the Kyma Control Plane (KCP) by exposing and regularly rotating the `kubeconfig` for each Kyma runtime.

The KIM image consists of three controllers:

- Runtime controller
- GardenerCluster controller that manages and rotates the Shoot `kubeconfig`
- RegistryCache controller

**Those three controllers are the inseparable components of KIM.**

With Kyma Environment Broker (KEB) and Lifecycle Manager, KIM builds the foundation for the Kyma runtime, with all backend services running within KCP.

### Process Flow

1.  When you request a Kyma runtime using BTP, KEB processes the request, creating or updating a `Runtime` CR instance that describes the Kubernetes cluster's infrastructure.
2.  KIM detects the change in the `Runtime` CR and triggers its reconciliation process.
3.  KIM converts the information from the `Runtime` CR into a `Shoot` definition for Gardener.
4.  After the cluster is created, KIM generates a `GardenCluster` CR, which contains the metadata required for fetching and rotating the cluster's `kubeconfig`.
5.  KIM fetches the `kubeconfig` from Gardener and stores it in a Secret within the KCP cluster.

## Configuration Parameters

The SAP BTP, Kyma runtime product installation sets up KIM.

You can change many of the settings by providing corresponding values in the `infrastructure-manager` `DataObject` for the SAP BTP, Kyma runtime product installation. The following table outlines the main configuration possibilities in the `infrastructure-manager` `DataObject`. The configuration uses the YAMLPath notation in the `DataObject's` data section, where each dot `.` represents a level in the `DataObject's` YAML structure under the main data key.
### Helm Chart Values

The following table describes the configurable parameters of the Kyma Infrastructure Manager Helm chart and their default values.

| Attribute(s)                              | Type   | Description                                                                                                                                             | Default Value                                                           | Mandatory |
|:------------------------------------------|:-------|:--------------------------------------------------------------------------------------------------------------------------------------------------------|:------------------------------------------------------------------------|-----------|
| `global.auditLogMandatory`                | bool   | A feature flag that enables strict audit log configuration. If `true`, a Shoot cluster will only be created if a corresponding audit log tenant exists. | `true`                                                                  |           |
| `global.structuredAuthEnabled`            | bool   | A feature flag to enable structured authentication.                                                                                                     | `true`                                                                  |           |
| `global.customConfigControllerEnabled`    | bool   | A feature flag to enable the dedicated controller for the registry cache feature.                                                                       | `false`                                                                 |           |
| `manager.kubeconfigExpiration`            | string | The maximum age (in minutes) of a Shoot `kubeconfig` before it is considered invalid and requires rotation.                                             | `"100m"`                                                                |           |
| `manager.minimalRotationTime`             | float  | A ratio of `kubeconfigExpiration` that determines the minimal time that must pass before a `kubeconfig` is rotated.                                     | `0.6`                                                                   |           |
| `manager.runtimeCtrlWorkersCnt`           | int    | The number of workers running in parallel for the Runtime Controller.                                                                                   | `25`                                                                    |           |
| `manager.gardenerClusterCtrlWorkersCnt`   | int    | The number of workers running in parallel for the GardenerCluster Controller.                                                                           | `5`                                                                     |           |
| `gardener.projectName`                    | string | The name of the Gardener project where Shoot definitions are stored.                                                                                    | `null`                                                                  | Yes       |
| `gardener.clientTimeout`                  | string | The timeout for requests made by the Runtime Controller to the Gardener API server.                                                                     | `"5s"`                                                                  |           |
| `gardener.clientQPS`                      | int    | The queries per second (QPS) limit for the Gardener client rate limiter.                                                                                | `20`                                                                    |           |
| `gardener.clientBurst`                    | int    | The burst value for the Gardener client rate limiter, allowing a number of requests above the QPS limit for a short period.                             | `20`                                                                    |           |
| `gardener.auditLogExtensionConfigMapName` | string | The name of the `ConfigMap` containing the audit log extension configuration.                                                                           | `"audit-extension-config"`                                              |           |
| `converterConfig.name`                    | string | The name of the `ConfigMap` for the shoot converter configuration.                                                                                      | `"infrastructure-manager-converter-config"`                             |           |
| `converterConfig.mountPath`               | string | The path inside the manager container where the converter `ConfigMap` should be mounted.                                                                | `"/converter-config"`                                                   |           |
| `converterConfig.key`                     | string | The key in the `ConfigMap` containing the converter configuration.                                                                                      | `"converter_config.json"`                                               |           |
| `converterConfig.path`                    | string | The filename for the mounted `ConfigMap` key.                                                                                                           | `"converter_config.json"`                                               |           |
| `converterConfig.contents`                | JSON   | The JSON content for the converter configuration.                                                                                                       | See more details in the section `Converter Config configuration values` | Yes       |
| `maintenanceWindow.name`                  | string | The name of the `ConfigMap` for the Gardener maintenance window configuration.                                                                          | `"maintenance-window-config"`                                           |           |
| `maintenanceWindow.mountPath`             | string | The path inside the manager container where the Gardener maintenance window `ConfigMap` should be mounted.                                              | `"/maintenance-window-config"`                                          |           |
| `maintenanceWindow.key`                   | string | The key in the `ConfigMap` containing the Gardener maintenance window configuration.                                                                    | `"config"`                                                              |           |
| `maintenanceWindow.path`                  | string | The filename for the mounted `ConfigMap` key.                                                                                                           | `"config"`                                                              |           |
| `auditlog.extensionConfig`                | JSON   | The JSON content for the audit log configuration for the regions and hyperscalers.                                                                      | `null`                                                                  | Yes       |

### Converter Config configuration values

The following table describes the parameters within the JSON object provided to the `converterConfig.contents` field. This configuration defines the default settings used by the Kyma Infrastructure Manager when converting a `Runtime` CR into a Gardener `Shoot` definition.

**All of below fields are mandatory**

| Attribute(s) | Type | Description |
| :--- | :--- | :--- |
| `cluster.defaultSharedIASTenant.ClientID` | string | The default OIDC client ID for the shared Identity and Access Service (IAS) tenant. |
| `cluster.defaultSharedIASTenant.GroupsClaim` | string | The claim in the OIDC token that contains the user's groups. |
| `cluster.defaultSharedIASTenant.IssuerURL` | string | The URL of the OIDC issuer. |
| `cluster.defaultSharedIASTenant.SigningAlgs` | list | A list of supported signing algorithms for the OIDC token. |
| `cluster.defaultSharedIASTenant.UsernameClaim` | string | The claim in the OIDC token to be used as the username. |
| `cluster.defaultSharedIASTenant.UsernamePrefix` | string | A prefix to be added to the username claim. |
| `converter.kubernetes.defaultVersion` | string | The default Kubernetes version for newly created Shoot clusters. |
| `converter.kubernetes.enableKubernetesVersionAutoUpdate` | bool | If `true`, the Kubernetes version of the Shoot cluster is automatically updated to newer patch versions. |
| `converter.kubernetes.enableMachineImageVersionAutoUpdate` | bool | If `true`, the machine image version of the Shoot cluster is automatically updated. |
| `converter.kubernetes.defaultOperatorOidc.ClientID` | string | The default OIDC client ID used by the Kubernetes operator. |
| `converter.kubernetes.defaultOperatorOidc.GroupsClaim` | string | The OIDC groups claim for the operator. |
| `converter.kubernetes.defaultOperatorOidc.IssuerURL` | string | The OIDC issuer URL for the operator. |
| `converter.kubernetes.defaultOperatorOidc.SigningAlgs` | list | The supported OIDC signing algorithms for the operator. |
| `converter.kubernetes.defaultOperatorOidc.UsernameClaim` | string | The OIDC username claim for the operator. |
| `converter.kubernetes.defaultOperatorOidc.UsernamePrefix` | string | The username prefix for the operator. |
| `converter.dns.secretName` | string | The name of the Kubernetes `Secret` containing credentials for the DNS provider. |
| `converter.dns.domainPrefix` | string | The domain prefix used for the cluster's DNS records (e.g., `example.com` results in `sub.example.com`). |
| `converter.dns.providerType` | string | The type of DNS provider to use for managing DNS records. |
| `converter.provider.aws.enableIMDSv2` | bool | If `true`, Instance Metadata Service Version 2 (IMDSv2) is enforced on all AWS nodes in the cluster. |
| `converter.gardener.projectName` | string | The name of the Gardener project where the Shoot cluster will be created. |
| `converter.machineImage.defaultName` | string | The default name of the machine image to use for worker nodes. |
| `converter.machineImage.defaultVersion` | string | The default version of the machine image to use. |
| `converter.auditLogging.policyConfigMapName` | string | The name of the `ConfigMap` containing the audit logging policy. |
| `converter.auditLogging.tenantConfigPath` | string | The file path inside the manager container where the audit log tenant configuration is located. |
| `converter.maintenanceWindow.windowMapPath` | string | The file path inside the manager container where the maintenance window configuration `ConfigMap` is mounted. |