# Kyma Infrastructure Manager

## Introduction

Kyma Infrastructure Manager (KIM) manages the lifecycle of SAP BTP, Kyma runtime, using SAP Gardener to create and manage Kubernetes cluster infrastructure components.
KIM translates Kubernetes cluster-related requirements for Kyma runtime into Gardener's Shoot definitions, managing their creation, update, and deletion. 
While Gardener reconciles a Kubernetes cluster, KIM monitors its progress, reacting to failures and ensuring the cluster is prepared for running the Kyma software.

## Stakeholder Expectations

You have multiple options to adjust your Kyma runtime. For example, you can define cluster sizes, configure OIDC authentication providers and administrators, introduce worker pools, and more. KIM ensures that the Kubernetes infrastructure created from Gardener is aligned with your latest Kyma configuration with minimal delay.


KIM must be implemented as a Kubernetes Operator. It provides a Custom Resource Definition (CRD) that exposes all configurable options of a Kubernetes cluster. This operator continuously watches the instances (custom resources (CRs)) of this CRD. Any change triggers reconciliation of the Shoot definition to align the Shoot definition with the description provided by the CR.

In addition to aligning the Kubernetes infrastructure, KIM also provides access to these clusters through Kyma Control Plane (KCP) by exposing and rotating their kubeconfigs. Each Kyma runtime has the kubeconfig stored in a Secret on KCP. To address security requirements, KIM also regularly rotates these kubeconfigs.


## Context and Scope

With Kyma Environment Broker (KEB) and Lifecycle Manager, KIM builds the foundation of Kyma runtime.
All backend services run in KCP.

Kyma runs three different control planes, each has a KIM instance deployed:

1. DEV: Used for development and integration of KCP components.

2. STAGE: Runs the current or next release candidates of the KCP components and is used to stabilize these components.
3. PROD: Includes the stabilized components and manages productive Kyma runtimes.

## Solution Strategy

![architecture](../adr/assets/keb-kim-target-arch.drawio.svg)


### Building Blocks

|Component|Acronym|Purpose|
|--|--|--|
|Business Technology Platform|BTP|An entry point for managing Kyma runtime|
|Kyma Environment Broker|KEB|The environment broker that receives requests from BTP and converts cluster-related configuration parameters into `RuntimeCR`|
|Runtime custom resource|RuntimeCR|KIM exposes a CRD to describe the infrastructure of a Kubernetes cluster. `RuntimeCR` represents an instance of this CRD |
|Runtime kubeconfig custom resource|RuntimeKubeconfigCR|Used to administrate and rotate Kyma runtime's kubeconfig. It is created and managed by KIM |


### Process Flow

1. You request a Kyma runtime using BTP. KEB receives the request and creates or updates a `RuntimeCR` instance. This `RuntimeCR` describes the Kubernetes cluster's infrastructure.
2. KIM detects the change of this `RuntimeCR` and triggers the reconciliation process.
3. KIM converts the information into a Shoot definition for Gardener.
4. After the cluster is created, KIM generates `RuntimeKubeconfigCR`, which includes data required for fetching and rotating the cluster kubeconfig.
5. KIM fetches the kubeconfig from Gardener and stores it in a Secret in the KCP cluster.


## Operations


### Installation

As runtime is a Kubernetes cluster expected.

KIM is bundled a containerized application. As component of the Kyma Control Plane, the common delivery process of our SRE team (Argo CD) is supported. Deployment is based on HELM.

cluster. The HELM-based deployment process is self-contained and fully automated, requiring no manual pre- or post-deployment actions. The SRE team facilitates deployment through a common delivery process using Argo CD.


### Updates

Updates normally do not require manual action. In rare cases where a new feature requires a migration, a rollout guide will be provided.


### Troubleshooting

For details, see the troubleshooting guides for KCP components.


### Observability

In addition to the standard Pod resource indicators such as CPU and memory, KIM exposes cluster reconciliation-specific information through its metrics REST endpoint (/metrics/*).

- `im_gardener_clusters_state` - Indicates the Status.state for GardenerCluster CRs.
- `im_runtime_state` - Exposes current Status.state for Runtime CRs.
3. `unexpected_stops_total` - Exposes the number of unexpected state machine stop events.
4. `im_kubeconfig_expiration` - Exposes the current kubeconfig expiration value in epoch timestamp value format.

## Architectural Decisions


### Kubebuilder

KIM follows the Kubernetes [operator pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/). 
Technically, it's built on the [Kubebuilder framework](https://github.com/kubernetes-sigs/kubebuilder) and benefits from its scaffolding features for controllers and models (CRDs).


### Domain Model

All business data is stored in etcd, using [custom resources](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/).

The data model consists of two CRDs:

* `Runtime` - Cluster data is stored in instances of the  CRD. 

* `RuntimeKubeconfig` - Kubeconfig-related metadata is stored in entities of the  CRD.

For information on the structure and the purpose of the different fields in these resources, see the [CRD files](https://github.com/kyma-project/kyma-infrastructure-manager/tree/main/config/crd/bases).

### State Machine
The reconciliation process is implemented using a [state machine pattern](https://en.wikipedia.org/wiki/Finite-state_machine). Each state represents a reconciliation step, and transitions to subsequent steps occur based on a step's return value.


### Sub-Components

To address the requirements for implementing KIM features, the following sub-components have been introduced :

* KIM Snatch

    This mandatory Kyma module belongs to KIM and is automatically installed on all Kyma runtimes. To avoid Kyma workloads from being assigned to worker pools created by customers, KIM Snatch assigns Kyma workloads to the Kyma worker pool.

    For more information, see the [KIM Snatch repository](https://github.com/kyma-project/kim-snatch/tree/main/docs/user).

* Gardener Syncer

    This Kubernetes CronJob synchronizes Seed cluster data from the Gardener cluster to KCP. KEB uses this data for customer input validation (primarily for the "Shoot and Seed Same Region" feature to detect if a Seed cluster exists in a particular Shoot region). The Seed data is stored in a ConfigMap in KCP.

    For more information, see the [Gardener Syncer documentation](https://github.com/kyma-project/gardener-syncer/blob/main/README.md).




## Quality Requirements

Non-technical requirements for KIM are:

* Performance: KIM processes about 5.000 Runtime instances.
* Extensibility: To ensure easy extensibility of KIM, the following software patterns are used:  
    * The state machine pattern for process flow control
    * The chain of responsibility pattern for rendering the Shoot definitions
*  Reliability: Since uptime is crucial for KIM, monitoring tools are employed to detect throughput degradation and long-running reconciliation processes, with alerting rules to notify about any SLA violations.
* Security:  SAP product standards are used during development, and a threat modeling workshop is executed annually.

