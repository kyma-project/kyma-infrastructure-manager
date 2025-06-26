# Kyma Infrastructure Manager


## Introduction

The Kyma Infrastructure Manager (KIM) is responsible for the lifecycle management of SAP Kyma runtimes (SKR). It was replacing the previously used Kyma Provisioner.

KIM is using SAP Gardener for creating and administrating Kubernetes cluster related infrastructure components.


## Goal

KIM is translating Kubernetes cluster related requirements for Kyma runtimes into Gardener's Shoot definitions. It is creating, updating and deleting Shoot definition. While Gardener is reconciling a Kubernetes cluster, KIM is monitoring the progress, reacting on failure cases but also ensuring the cluster is prepared for running the Kyma software.


## Constraints

Kyma customers have multiple options to adjust their Kyma runtime  (e.g. define cluster sizes, configure OIDC authentication providers and administrators, introduce worker pools etc.) KIM has to ensure the created Kubernetes infrastructure from Gardener is aligned with the latest Kyma customer configuration. The alignment has to happen with minimal delay.

The maintainability of the previous Provisioner was a major drawback and new features required usually modification at multiple places. A simple and easy extendable architecture of KIM has to address these points. The reconciliation flow has to be easy adjust- and extendable.

KIM has to be implemented as Kubernetes Operator. It will provide a Custom Resource Definition (CRD) which exposes all configurable options of a Kubernetes cluster. This operator will continuously watch the instances (Custom Resources, CR) of this CRD. Any change will trigger a reconciliation of the Shoot definition to align the Shoot definition with the description provided by the CR.

In addition to the alignment of Kubernetes infrastructure. KIM is also providing access to these clusters via KCP by exposing and rotating their Kubeconfig: for each running Kyma runtime is the kubeconfig stored in a secret on KCP. For addressing security requirements, KIM is also regularly rotating this kubeconfig.

## Context and Scope

Together with the Kyma Environment Broker (KEB) and Kyma Lifecycle Manager (KLM), it builds the backbone of managed Kyma runtimes.

All backend service are running in the Kyma Control Plane (KCP).

Kyma operators three different control planes, each has a KIM instance deployed:

1. DEV

    Used for development and integration purposes of KCP components.

2. STAGE

    Runs the current or next release candidates of KCP component and is used for stabilization of these components.

3. PROD

    The productive environment includes the stabilized components and manages productive Kyma runtimes.



## Solution Strategy

![architecture](../adr/assets/keb-kim-target-arch.drawio.svg)

### Building Blocks

|Component|Acronym|Purpose|
|--|--|--|
|Business Technology Platform|BTP|This is the entry point for customers to manage their Kyma runtime.|
|Kyma Environment Broker|KEB|The environment broker receives requests from BTP. Cluster related configuration parameters are converted into a `RuntimeCR`|
|Runtime Custom Resource|RuntimeCR|KIM exposes a Customer Resource Defintion to describe the infrastructure of a Kubernetes Cluster. A `RuntimeCR` represents an instance of this CRD.|
|Runtime Kubeconfig Custom Resource|RuntimeKubeconfigCR|This CR is used to administrate and rotate the kubeconfig of a Kyma runtime. It is created and managed by KIM.|


### Process Flow

1. The customer requests  a Kyma runtime via BTP. The request is received by KEB which creates/updates an `RuntimeCR` instance. this `RuntimeCR` describes how the infrastructure of the Kubernetes cluster.
2. KIM is detecting the change of this `RuntimeCR` and triggers the reconciliation process.
3. KIM converts the information into a Shoot definition for Gardener.
4. After the cluster was created, KIM creates a `RuntimeKubeconfigCR` which include required data for fetching and rotating the cluster Kubeconfig.
5. It fetches the Kubeconfig from Gardener and stores it in a secret on the KCP cluster.


## Operation

### Installation

As runtime is a Kubernetes cluster expected.

KIM is bundled a containerized application. As component of the Kyma Control Plane, the common delivery process of our SRE team (Argo CD) is supported. Deployment is based on HELM.

The deployment of KIM is self-contained and can be fully automated (no manual pre- or post-actions are required).


### Updates

Updates are normally not requiring any manual action. In seldom cases where a new feature requires a migration, a rollout guide will be provided.


### Troubleshooting

Please see the trouble shooting guides for KCP components for further details.


### Observability

KIM exposes, beside the common Pod resource indicators (CPU, memory etc.), also cluster reconciliation specific information over his metrics REST-endpoint (`/metrics/*` ):

1. `im_gardener_clusters_state`

    Indicates the Status.state for GardenerCluster CRs.

2. `im_runtime_state`

    Exposes current Status.state for Runtime CRs.

3. `unexpected_stops_total`

    Exposes the number of unexpected state machine stop events.

4. `im_kubeconfig_expiration`

    Exposes current kubeconfig expiration value in epoch timestamp value format.


## Architectural decisions

### Kubebuilder

KIM is following the [operator pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/) of Kubernetes. 

Technically, its build on the [Kubebuilder framework](https://github.com/kubernetes-sigs/kubebuilder) and benefits from its scaffolding features for controller and models (CRDs).


### Domain model

All business data are completely stored in ETCD, using [Custom Resources](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/).

The data model consists of two CRDs:

* `Runtime`

    Cluster data are stored in instances of the  CRD. 

* `RuntimeKubeconfig`

    Kubeconfig related metadata are stored in entities of the  CRD.

The structure and the purpose of the different fields in these resources are documented in the CRD files:

https://github.com/kyma-project/kyma-infrastructure-manager/tree/main/config/crd/bases


### State Machine
The reconciliation process is implemented by using a [state machine pattern](https://en.wikipedia.org/wiki/Finite-state_machine). Each state represents a reconciliation step. Based on the return value of a step, the transition to a next steps happens.

### ADR
Please see the ADR (Architectural Decision Record) of KIM tp get more insights about its [software architecture](../adr/001-provisioning.md).


## Quality Requirements

Non-technical requirements for KIM are:

* Performance

    Goal was to be able to process +/- 5.000 Runtime instances with KIM. A load and performance test was implemented which confirms the achievement of this goal.

* Extensibility

    Selected software patterns are used to ensure an easy extensibility of KIM:
    
    *  state machine pattern is used for process flow control
    *  chain of responsibility pattern is used for rendering the Shoot definitions

*  Reliability

    Uptime is crucial for KIM. Monitoring is detecting throughput degradation and long-running reconciliation processes. Alerting rule are in place to notify the team about SLA violations.

* Security

    SAP product standards are used during development and a threat modeling workshop is executed annually.


