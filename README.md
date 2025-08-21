[![REUSE status](https://api.reuse.software/badge/github.com/kyma-project/infrastructure-manager)](https://api.reuse.software/info/github.com/kyma-project/infrastructure-manager)
[![Go Report Card](https://goreportcard.com/badge/github.com/kyma-project/infrastructure-manager)](https://goreportcard.com/report/github.com/kyma-project/infrastructure-manager)
[![unit tests](https://badgen.net/github/checks/kyma-project/kyma-infrastructure-manager/main/unit-tests)](https://github.com/kyma-project/kyma-infrastructure-manager/actions/workflows/run-tests.yaml)
[![Coverage Status](https://coveralls.io/repos/github/kyma-project/kyma-infrastructure-manager/badge.svg?branch=main)](https://coveralls.io/github/kyma-project/kyma-infrastructure-manager?branch=main)
[![golangci lint](https://badgen.net/github/checks/kyma-project/kyma-infrastructure-manager/main/golangci-lint)](https://github.com/kyma-project/kyma-infrastructure-manager/actions/workflows/golangci-lint.yaml)
[![latest release](https://badgen.net/github/release/kyma-project/kyma-infrastructure-manager)](https://github.com/kyma-project/kyma-infrastructure-manager/releases/latest)

# Kyma Infrastructure Manager

## Overview

Kyma Infrastructure Manager (KIM) manages the [Kyma](https://kyma-project.io/#/) cluster infrastructure. It's built using the [kubebuilder](https://github.com/kubernetes-sigs/kubebuilder) framework.

It's responsible for generating and rotating Secrets containing dynamic kubeconfigs.

## Prerequisites

- Access to a Kubernetes cluster. You can use [k3d](https://k3d.io) to get a local cluster for testing or run against a remote cluster.
- [kubectl](https://kubernetes.io/docs/tasks/tools/)

## Installation

1. Clone the project.

    ```bash
    git clone https://github.com/kyma-project/infrastructure-manager.git && cd infrastructure-manager/
    ```

2. Set the `infrastructure-manager` image name.

    ```bash
    export IMG=custom-infrastructure-manager:0.0.1
    export K3D_CLUSTER_NAME=infrastructure-manager-demo
    ```

3. Build the project.

    ```bash
    make build
    ```

4. Build the image.

    ```bash
    make docker-build
    ```

5. Push the image to the registry.

    <div tabs name="Push image" group="infrastructure-manager-installation">
      <details>
      <summary label="k3d">
      k3d
      </summary>


      ```bash
      k3d cluster create $K3D_CLUSTER_NAME
      k3d image import $IMG -c $K3D_CLUSTER_NAME
      ```

      </details>
      <details>
      <summary label="Docker registry">
      Globally available Docker registry
      </summary>

      ```bash
      make docker-push
      ```

      </details>
    </div>

6. Deploy.

    ```bash
    make deploy
    ```

7. Create a Secret with the Gardener credentials.

    ```bash
    export GARDENER_KUBECONFIG_PATH=<kubeconfig file for Gardener project> 
    make gardener-secret-deploy
    ```

## Usage

KIM is responsible for creating and rotating Secrets of clusters defined in the `GardenerCluster` custom resources (CRs). The sample CR is available in this [YAML file](config/samples/infrastructuremanager_v1_gardenercluster.yaml).

### Time-Based Rotation

Secrets are rotated based on `kubeconfig-expiration-time`. For more information, see [Configuration](docs/README.md#configuration).

### Force Rotation

It's possible to force the Secret rotation before the time-based rotation kicks in. To do that, add the `operator.kyma-project.io/force-kubeconfig-rotation: "true"` annotation to the `GardenCluster` CR.

## Contributing
<!--- mandatory section - do not change this! --->

See [CONTRIBUTING.md](CONTRIBUTING.md)

## Code of Conduct
<!--- mandatory section - do not change this! --->

See [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md)

## Licensing
<!--- mandatory section - do not change this! --->

See the [LICENSE file](./LICENSES/Apache-2.0.txt)
