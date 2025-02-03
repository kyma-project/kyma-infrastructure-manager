# Docs

## Overview

This folder contains documents that relate to the project.

## Development

Run `make test` to see if all tests are passing. 

## Configuration

### Deployment Arguments Configuration
1. `gardener-kubeconfig-path` - defines the path to the Gardener project kubeconfig used during API calls
2. `gardener-project-name` - the name of the Gardener project where the infrastructure operations are performed
3. `minimal-rotation-time` - the ratio determines what is the minimal time that needs to pass to rotate the certificate
4. `kubeconfig-expiration-time` - maximum time after which kubeconfig is rotated. The rotation happens between (`minimal-rotation-time` * `kubeconfig-expiration-time`) and `kubeconfig-expiration-time`.
5. `gardener-request-timeout` - specifies the timeout for requests to Gardener. Default value is `3s`.
6. `gardener-ctrl-reconcilation-timeout` - timeout for duration of the reconlication for Gardener Cluster Controller. Default value is `60s`.
7. `gardener-ratelimiter-qps` - Gardener client rate limiter QPS parameter for Runtime Controller.  Default value is `5`.
8. `gardener-ratelimiter-burst` - Gardener client rate limiter Burst parameter for Runtime Controller.  Default value is `5`.
9. `shoot-spec-dump-enabled` - feature flag responsible for enabling the shoot spec dump. Default value is `false`.
10. `audit-log-mandatory` - feature flag responsible for enabling the Audit Log strict config. Default value is `true`.
11. `runtime-ctrl-workers-cnt` - number of workers running in parallel for Runtime Controller. Default value is `25`.
12. `gardener-cluster-ctrl-workers-cnt` - number of workers running in parallel for GardenerCluster Controller. Default value is `25`.

See [manager_gardener_secret_patch.yaml](../config/default/manager_gardener_secret_patch.yaml) for default values.

## Troubleshooting

### Runtime Custom Resources Configuration
The following annotations can control runtime behavior:

| Annotation  | Description                                                                                                                                                                                                                                                                                                                         |
| ------------- |-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| operator.kyma-project.io/force-patch-reconciliation  | If set to `true`, the next reconciliation loop enters the patch state regardless of the `runtime-generation` number. This annotation is removed automatically after attempting the patch operation. Might produce the `object has been modified` error in the RuntimeController logs until the state is reconciled. |
| operator.kyma-project.io/suspend-patch-reconciliation  | If set to`true`, the controller does not patch the shoot. It has to be manually removed to resume normal operation.                                                                                                                                                                                                    |


### Switching Between `provisioner` and `kim`.

The `kyma-project.io/controlled-by-provisioner` label provides fine-grained control over the `Runtime` CR. Only if the label value is set to `false`, the resource is considered managed and will be controlled by `kyma-application-manager`.

> TBD: List potential issues and provide tips on how to avoid or solve them. To structure the content, use the following sections:
>
> - **Symptom**
> - **Cause**
> - **Remedy**
