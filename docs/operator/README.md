# Kyma Infrastructure Manager

## Introduction

Kyma Infrastructure Manager (KIM) manages the lifecycle of SAP BTP, Kyma runtime, using SAP Gardener to create and manage Kubernetes cluster infrastructure components. KIM translates Kubernetes cluster-related requirements for Kyma runtime into Gardener's Shoot definitions, managing their creation, update, and deletion. While Gardener reconciles a Kubernetes cluster, KIM monitors its progress, reacting to failures and ensuring the cluster is prepared to run the Kyma software.

## Stakeholder Expectations

You have multiple configuration options for Kyma runtime, such as defining cluster sizes, configuring OIDC authentication providers and administrators, introducing worker pools, and more. KIM aligns the Kubernetes infrastructure created from Gardener with your latest Kyma configuration with minimal delay.

As a Kubernetes Operator, KIM provides a Custom Resource Definition (CRD) that exposes all configurable options of a Kubernetes cluster. It continuously watches the instances (custom resources (CRs)) of this CRD, triggering Shoot definition reconciliation upon any change to match the description provided by the CR.

In addition to the Kubernetes infrastructure alignment, KIM provides cluster access using Kyma Control Plane (KCP) and kubeconfig exposure and rotation. Each Kyma runtime has its kubeconfig stored in a Secret on KCP. To address security requirements, KIM also regularly rotates these kubeconfigs.

## Context and Scope

With Kyma Environment Broker (KEB) and Lifecycle Manager, KIM builds the foundation for the Kyma runtime, with all backend services running within KCP.

## Architecture

### Components

![architecture](./assets/keb-kim-arch.drawio.svg)

- Business Technology Platform (BTP) - An entry point for managing Kyma runtime.
- Kyma Environment Broker - Receives requests from BTP and converts cluster-related configuration parameters into Runtime CR.
- Runtime custom resource - Represents an instance of KIM's CRD, describing Kubernetes cluster infrastructure.
- RuntimeKubeconfig CR - Administrates and rotates Kyma runtime's kubeconfig; created and managed by KIM.

### Process Flow

1. When you request a Kyma runtime using BTP, KEB processes the request, creating or updating a Runtime CR instance that describes the Kubernetes cluster's infrastructure.
2. KIM detects the change in the Runtime CR and triggers the reconciliation process.
3. KIM converts the information into a Shoot definition for Gardener.
4. After the cluster is created, KIM generates a RuntimeKubeconfig CR, which includes data required for fetching and rotating the cluster kubeconfig.
5. KIM fetches the kubeconfig from Gardener and stores it in a Secret in the KCP cluster.

### Architectural Decisions

#### Kubebuilder

KIM follows the Kubernetes [operator pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/). Technically, it's built on the [Kubebuilder framework](https://github.com/kubernetes-sigs/kubebuilder) and benefits from its scaffolding features for controllers and models (CRDs).

#### Domain Model

All business data is stored in etcd, using [custom resources](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/).

The data model consists of two CRDs:

* Runtime CRD - Stores cluster data within instances of the CRD.
* RuntimeKubeconfig CRD - Stores kubeconfig-related metadata in entities of the CRD.

For more information on the structure and purpose of various fields, see the CRD files. The URL pattern for these files is:

`https://github.com/kyma-project/kyma-infrastructure-manager/tree/{KIM_VERSION}/config/crd/bases`

Example:

https://github.com/kyma-project/kyma-infrastructure-manager/tree/1.24.0/config/crd/bases

#### State Machine

The reconciliation process is implemented using a [state machine pattern](https://en.wikipedia.org/wiki/Finite-state_machine). Each state represents a reconciliation step. Transitions to subsequent steps occur based on the return value of a step.

#### Sub-Components

To address the requirements for implementing KIM features, the following sub-components are introduced:

* KIM Snatch: Automatically installed on all Kyma runtimes. To prevent assigning Kyma workloads to worker pools created by customers, KIM Snatch assigns Kyma workloads to the Kyma worker pool. For more information, see the [KIM Snatch documentation](https://github.com/kyma-project/kim-snatch/tree/main/docs/user).
* Gardener Syncer: A Kubernetes CronJob synchronizing Seed cluster data from the Gardener cluster to KCP. KEB uses this data for customer input validation (primarily for the "Colocate Control Plane" feature to detect if a Seed cluster exists in a particular Shoot region). The Seed data is stored in a ConfigMap in KCP. For more information, see the [Gardener Syncer documentation](https://github.com/kyma-project/gardener-syncer/blob/main/README.md).

## Operations

### Installation

KIM, a component of the KCP, is delivered as a containerized application and deployed within a Kubernetes cluster. The HELM-based deployment process is self-contained and fully automated, requiring no manual pre- or post-deployment actions. The SRE team facilitates deployment through a common delivery process using Argo CD.

### Updates

Updates don't usually require manual action. In rare cases where a new feature requires a migration, a rollout guide will be provided.

### Observability

KIM exposes multiple data over a metrics REST endpoint (`/metrics/*`), which provides insights about the application health state.

In addition to the standard Pod resource indicators such as CPU and memory, KIM exposes cluster reconciliation-specific information:

- `im_gardener_clusters_state` - Indicates the Status.state for GardenerCluster CRs
- `im_runtime_state` - Exposes current Status.state for Runtime CRs
- `unexpected_stops_total` - Exposes the number of unexpected state machine stop events
- `im_kubeconfig_expiration` - Exposes the current kubeconfig expiration value in epoch timestamp value format

### Configuration Parameters

KIM can be configured using command-line parameters. Supported parameters are described in [Kyma Infrastructure Manager Configuration](https://github.com/kyma-project/kyma-infrastructure-manager/blob/main/docs/operator/kim-configuration.md).

## Quality Requirements

* Performance: Processes about 5.000 Runtime instances.
* Extensibility: Ensures easy extensibility by employing the following software patterns:  
    * State machine pattern for process flow control.
    * Chain of responsibility pattern for rendering the Shoot definitions.
* Reliability: Monitoring tools detect throughput degradation and long-running reconciliation processes, triggering alerts for any SLA violations.
* Security: Adheres to SAP product standards during development; threat modeling workshops are executed annually.
