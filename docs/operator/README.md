# Kyma Infrastructure Manager

## Introduction

Kyma Infrastructure Manager (KIM) manages the lifecycle of SAP BTP, Kyma runtime, using SAP Gardener to create and manage Kubernetes cluster infrastructure components.
KIM translates Kubernetes cluster-related requirements for Kyma runtime into Gardener's Shoot definitions, managing their creation, update, and deletion. 
While Gardener reconciles a Kubernetes cluster, KIM monitors its progress, reacting to failures and ensuring the cluster is prepared for running the Kyma software.

## Stakeholder Expectations

You have multiple configuration options for Kyma runtime, such as defining cluster sizes, configuring OIDC authentication providers and administrators, introducing worker pools, and more. KIM aligns the Kubernetes infrastructure created from Gardener with your latest Kyma configuration with minimal delay.
As a Kubernetes Operator, KIM provides a Custom Resource Definition (CRD) that exposes all configurable options of a Kubernetes cluster. It continuously watches the instances (custom resources (CRs)) of this CRD, triggering Shoot definition reconciliation upon any change to match the description provided by the CR.

In addition to the Kubernetes infrastructure alignment, KIM provides cluster access through Kyma Control Plane (KCP) through kubeconfig exposure and rotation. Each Kyma runtime has its kubeconfig stored in a Secret on KCP. To address security requirements, KIM also regularly rotates these kubeconfigs.

## Context and Scope

With Kyma Environment Broker (KEB) and Lifecycle Manager, KIM builds the foundation for Kyma runtime, with all backend services running within KCP.

Kyma runs three different control planes, each deploying a KIM instance:

1. DEV: For development and integration of KCP components
2. STAGE: Running the current or next release candidates of the KCP components; used to stabilize these components
3. PROD: Including the stabilized components and managing productive Kyma runtimes

## Architecture

### Components

![architecture](../assets/keb-kim-arch.drawio.svg)
- Business Technology Platform (BTP) - Entry point for managing Kyma runtime
- Kyma Environment Broker (KEB) - Receives requests from BTP and converts cluster-related configuration parameters into `RuntimeCR`
- Runtime custom resource (`RuntimeCR`) - Represents an instance of KIM's CRD, describing Kubernetes cluster infrastructure
- Runtime kubeconfig custom resource (`RuntimeKubeconfigCR`) - Administrates and rotates Kyma runtime's kubeconfig; created and managed by KIM

### Process Flow

1. When you request a Kyma runtime using BTP, KEB processes the request, creating or updating a `RuntimeCR` instance that describes the Kubernetes cluster's infrastructure.
2. KIM detects the change in `RuntimeCR` and triggers the reconciliation process.
3. KIM converts the information into a Shoot definition for Gardener.
4. After the cluster is created, KIM generates `RuntimeKubeconfigCR`, which includes data required for fetching and rotating the cluster kubeconfig.
5. KIM fetches the kubeconfig from Gardener and stores it in a Secret in the KCP cluster.

### Architectural Decisions

#### Kubebuilder

KIM follows the Kubernetes [operator pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/). 
Technically, it's built on the [Kubebuilder framework](https://github.com/kubernetes-sigs/kubebuilder) and benefits from its scaffolding features for controllers and models (CRDs).


#### Domain Model

All business data is stored in etcd, using [custom resources](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/).

The data model consists of two CRDs:

* `Runtime CRD` - Stores cluster data within instances of the  CRD
* `RuntimeKubeconfig CRD` - Stores kubeconfig-related metadata in entities of the  CRD

For information on the structure and the purpose of the different fields in these resources, see the CRD files. URL-Pattern for these files is:

`https://github.com/kyma-project/kyma-infrastructure-manager/tree/{KIM_VERSION}/config/crd/bases`

Example:

https://github.com/kyma-project/kyma-infrastructure-manager/tree/1.24.0/config/crd/bases

#### State Machine

The reconciliation process is implemented using a [state machine pattern](https://en.wikipedia.org/wiki/Finite-state_machine). Each state represents a reconciliation step, and transitions to subsequent steps occur based on a step's return value.

#### Sub-Components

To address the requirements for implementing KIM features, the following sub-components have been introduced :

* KIM Snatch: Automatically installed on all Kyma runtimes to prevent assigning Kyma workloads to worker pools created by customers, KIM Snatch assigns Kyma workloads to the Kyma worker pool. For more information, see the [KIM Snatch documentation](https://github.com/kyma-project/kim-snatch/tree/main/docs/user).
* Gardener Syncer: A Kubernetes CronJob synchronizing Seed cluster data from the Gardener cluster to KCP. KEB uses this data for customer input validation (primarily for the "Shoot and Seed Same Region" feature to detect if a Seed cluster exists in a particular Shoot region). The Seed data is stored in a ConfigMap in KCP. For more information, see the [Gardener Syncer documentation](https://github.com/kyma-project/gardener-syncer/blob/main/README.md).


## Operations

### Installation

KIM, a component of the KCP, is delivered as a containerized application and deployed within a Kubernetes cluster. The HELM-based deployment process is self-contained and fully automated, requiring no manual pre- or post-deployment actions. The SRE team facilitates deployment through a common delivery process using Argo CD.

### Updates

Updates don't usually require manual action. In rare cases where a new feature requires a migration, a rollout guide will be provided.

### Troubleshooting

For details, see the troubleshooting guides for KCP components.


### Observability

KIM exposes multiple data over an metrics REST endpoint (`/metrics/*`) which provides insights about the application health state.

In addition to the standard Pod resource indicators such as CPU and memory, KIM exposes cluster reconciliation-specific information:

- `im_gardener_clusters_state` - Indicates the Status.state for GardenerCluster CRs
- `im_runtime_state` - Exposes current Status.state for Runtime CRs
- `unexpected_stops_total` - Exposes the number of unexpected state machine stop events
- `im_kubeconfig_expiration` - Exposes the current kubeconfig expiration value in epoch timestamp value format


### Configuration Parameters

KIM can be configured via command line parameters. The supported command line parameters are available and described in this file main.go file of KIM.

Please use this URL pattern to retrieve the supported parameters for your deployed KIM version:

`https://github.com/kyma-project/kyma-infrastructure-manager/blob/{KIM_VERSION}/cmd/main.go#L108`

Example-URL for KIM version 1.26.2:

https://github.com/kyma-project/kyma-infrastructure-manager/blob/1.26.2/cmd/main.go#L108

Alternatively, the support configuration parameters can be retrieved by executing:

```
# Please define your KIM version:
export KIM_VERSION=1.26.2

git clone -b "$KIM_VERSION" https://github.com/kyma-project/kyma-infrastructure-manager.git

cd ./kyma-infrastructure-manager/cmd

go run main.go --help
```

## Quality Requirements

Non-technical requirements for KIM are:

* Performance: Processes about 5.000 Runtime instances.
* Extensibility: Ensures easy extensibility by employing the following software patterns:  
    * State machine pattern for process flow control.
    * Chain of responsibility pattern for rendering the Shoot definitions.
*  Reliability: Monitoring tools detect throughput degradation and long-running reconciliation processes, triggering alerts for any SLA violations.
* Security:  Adheres to SAP product standards during development; threat modeling workshops are executed annually.

