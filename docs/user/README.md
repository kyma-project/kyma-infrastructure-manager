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

1. DEV: Used for development and integration purposes of KCP components.
2. STAGE: Runs the current or next release candidates of KCP component and is used for stabilization of these components.
3. PROD: The productive environment includes the stabilized components and manages productive Kyma runtimes.



## Solution Strategy

![architecture](../adr/assets/keb-kim-target-arch.drawio.svg)

### Building Blocks

|Component|Acronym|Purpose|
|--|--|--|
|Business Technology Platform|BTP|This is the entry point for customers to manage their Kyma runtime.|
|Kyma Environment Broker|KEB|The environment broker receives requests from BTP. Cluster related configuration parameters are converted into a `RuntimeCR`|
|Runtime Custom Resource|RuntimeCR|KIM exposes a Customer Resource Defintion to describe the infrastructure of a Kubernetes Cluster. A `RuntimeCR` represents an instance of this CRD.|
|||


**Process Flow:**

1. The customer requests  a Kyma runtime via BTP. The request is received by KEB which creates/updates an `RuntimeCR` instance. this `RuntimeCR` describes how the infrastructure of the Kubernetes cluster has to look like.
2. KIM is detecting the change of this `RuntimeCR` 