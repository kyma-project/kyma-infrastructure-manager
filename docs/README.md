# Docs

## Overview

This folder contains documents that relate to the project.

## Development

Run `make test` to see if all tests are passing. 

## Configuration

You can configure the Infrastructure Manager deployment with the following arguments:
1. `gardener-kubeconfig-path` - defines the path to the Gardener project kubeconfig used during API calls
2. `gardener-project-name` - the name of the Gardener project where the infrastructure operations are performed
3. `minimal-rotation-time` - the ratio determines what is the minimal time that needs to pass to rotate the certificate
4. `kubeconfig-expiration-time` - maximum time after which kubeconfig is rotated. The rotation happens between (`minimal-rotation-time` * `kubeconfig-expiration-time`) and `kubeconfig-expiration-time`.
5. `gardener-request-timeout` - specifies the timeout for requests to Gardener. Default value is `3s`.
6. `gardener-ctrl-reconcilation-timeout` - timeout for duration of the reconlication for Gardener Cluster Controller. Default value is `60s`.
7. `gardener-ratelimiter-qps` - Gardener client rate limiter QPS parameter for Runtime Controller.  Default value is `5`.
8. `gardener-ratelimiter-burst` - Gardener client rate limiter Burst parameter for Runtime Controller.  Default value is `5`.
9. `audit-log-mandatory` - feature flag responsible for enabling the Audit Log strict config. Default value is `true`.
10. `runtime-ctrl-workers-cnt` - number of workers running in parallel for Runtime Controller. Default value is `25`.
11. `gardener-cluster-ctrl-workers-cnt` - number of workers running in parallel for GardenerCluster Controller. Default value is `25`.


See [manager_gardener_secret_patch.yaml](../config/default/manager_gardener_secret_patch.yaml) for default values.
