# Docs

## Overview

This folder contains documents that relate to the project.

## Development

Run `make test` to see if all tests are passing. 

## Configuration

You can configure the Infrastructure Manager deployment with the following arguments:
1. `gardener-kubeconfig-path` - defines the path to the Gardener project kubeconfig used during API calls
2. `gardener-project` - the name of the Gardener project where the infrastructure operations are performed
3. `minimal-rotation-time` - the ratio determines what is the minimal time that needs to pass to rotate the certificate
4. `kubeconfig-expiration-time` - maximum time after which kubeconfig is rotated. The rotation happens between (`minimal-rotation-time` * `kubeconfig-expiration-time`) and `kubeconfig-expiration-time`.
4. `gardener-request-timeout` - specifies the timeout for requests to Gardener. Default value is `60s`.
5. `shoot-spec-dump-enabled` - feature flag responsible for enabling the shoot spec dump. Default value is `false`.
6. `audit-log-mandatory` - feature flag responsible for enabling the Audit Log strict config. Default value is `true`.


See [manager_gardener_secret_patch.yaml](../config/default/manager_gardener_secret_patch.yaml) for default values.

## Troubleshooting

1. Switching between the `provisioner` and `kim`.

The `kyma-project.io/controlled-by-provisioner` label provides fine-grained control over the `Runtime` CR. Only if the label value is set to `false`, the resource is considered managed and will be controlled by `kyma-application-manager`.

> TBD: List potential issues and provide tips on how to avoid or solve them. To structure the content, use the following sections:
>
> - **Symptom**
> - **Cause**
> - **Remedy**
