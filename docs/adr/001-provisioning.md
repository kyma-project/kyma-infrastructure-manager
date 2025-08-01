# Context
This document defines the architecture and API for the Gardener cluster provisioning functionality.

# Status
Accepted

# Decision

The following diagram shows the proposed architecture:
![](./assets/keb-kim-target-arch.drawio.svg)

> Note: At the time of writing, the GardenerCluster CR was used to generate kubeconfig. The [workplan](https://github.com/kyma-project/infrastructure-manager/issues/112) for delivering provisioning functionality includes renaming the CR to maintain consistency.

The following assumptions were taken:
- Kyma Environment Broker must not contain all the details of the cluster infrastructure.
- Kyma Infrastructure Manager's API must expose properties that:
  - can be set in the BTP cockpit by the user
  - are directly related to plans in the KEB
- Kyma Infrastructure Manager's API must not expose properties that are:
  - hardcoded in the Provisioner, or the KEB
  - statically configured in the management-plane-config

Kyma Environment Broker has the following responsibilities:  
- Create Runtime CR containing the following data:
    - Provider config (type, region, and secret with credentials for hyperscaler)
    - Worker pool specification
    - Cluster networking settings (nodes, pods, and services API ranges)
    - OIDC settings
    - Cluster administrators list
    - Egress network filter settings
    - Control Plane failure tolerance config
- Observe the status of the CR to determine whether provisioning succeeded

 Kyma Infrastructure Manager has the following responsibilities:
- Create shoots based on:
   - Corresponding `Runtime` CR properties
   - Corresponding `Runtime` CR labels:
     -  `kyma-project.io/platform-region` for determining if the cluster is located in EU 
   - Predefined defaults for the optional properties:
     - Kubernetes version
     - Machine image version
   - Predefined configuration for the following functionalities:
     - configuring DNS extension 
     - configuring Certificates extension
     - providing maintenance settings (Kubernetes, and image autoupdates)
     - creating provider specific config
 - Upgrade and delete shoots for the corresponding `Runtime` CRs
 - Apply the audit log configuration on the shoot resource
 - Create cluster role bindings for administrators
 - Generate the kubeconfig

## API proposal

### CR examples

Mind that the Runtime CR must be labeled to make searching for a particular instance easier. 
The proposed list of labels to be added to the Runtime CR:
```yaml
 kyma-project.io/instance-id: instance-id
 kyma-project.io/runtime-id: runtime-id
 kyma-project.io/broker-plan-id: plan-id
 kyma-project.io/broker-plan-name: plan-name
 kyma-project.io/global-account-id: global-account-id
 kyma-project.io/subaccount-id: subAccount-id
 kyma-project.io/shoot-name: shoot-name
 kyma-project.io/region: region
 operator.kyma-project.io/kyma-name: kymaName
```

The labels are skipped in the following examples due to clarity.

The example below shows the CR that must be created by the KEB to provision the AWS production cluster:
```yaml
apiVersion: infrastructuremanager.kyma-project.io/v1alpha1
kind: Runtime
metadata:
  name: runtime-id
  namespace: kcp-system
spec:
  shoot:
    # spec.shoot.name is required
    name: shoot-name
    # spec.shoot.purpose is required
    purpose: production
    # spec.shoot.region is required
    region: eu-central-1
    # spec.shoot.platformRegion is required
    platformRegion: "cf-eu11"
    # spec.shoot.secretBindingName is required
    secretBindingName: "hyperscaler secret"
    kubernetes:
      kubeAPIServer:
        # spec.shoot.kubernetes.kubeAPIServer.oidcConfig is required
        oidcConfig:
          clientID: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
          groupsClaim: groups
          issuerURL: https://my.cool.tokens.com
          signingAlgs:
            - RS256
          usernameClaim: sub
    provider:
      # spec.shoot.provider.type is required
      type: aws
      # spec.shoot.provider.workers is required
      workers:
        - machine:
          # spec.shoot.workers.machine.type is required
          type: m6i.large
          # spec.shoot.workers.zones is required
          zones:
            - eu-central-1a
            - eu-central-1b
            - eu-central-1c
          # spec.shoot.workers.minimum is required
          minimum: 3
          # spec.shoot.workers.maximum is required
          maximum: 20
          # spec.shoot.workers.maxSurge is required in the first release.
          # It can be optional in the future, as it equals to zone count
          maxSurge: 3
          # spec.shoot.workers.maxUnavailable is required in the first release.
          # It can be optional in the future, as it is always set to 0
          maxUnavailable: 0
    # spec.shoot.Networking is required
    networking:
      pods: 100.64.0.0/12
      nodes: 10.250.0.0/16
      services: 100.104.0.0/13
    # spec.shoot.controlPlane is required
    controlPlane:
      highAvailability:
        failureTolerance:
          type: node
  security:
    networking:
      filter:
        # spec.security.networking is required
        egress:
          enabled: false
    # spec.security.administrators is required
    administrators:
      - admin@myorg.com
```

There are some additional optional fields that could be specified:
- `spec.shoot.enforceSeedLocation` ; if not provided `false` value will be used
- `spec.shoot.licenceType` ; if not provided `nil` value will be used
- `spec.shoot.kubernetes.version` ; if not provided, the default value will be read by the KIM from the configuration
- `spec.shoot.kubernetes.kubeAPIServer.additionalOidcConfig` ; if not provided, no additional OIDC provider will be configured
- `spec.shoot.workers.name` ; if not provided, a Gardener default will be used
- `spec.shoot.workers.machine.image` ; if not provided, the default value will be read by the KIM from the configuration
- `spec.security.networking.filtering.ingress.enabled` ; if not provided, the `false` value will be used

The following example shows the Runtime CR that must be created to provision a cluster with an additional OIDC provider and to enable ingress network filtering:
```yaml
apiVersion: infrastructuremanager.kyma-project.io/v1alpha1
kind: Runtime
metadata:
  name: runtime-id
  namespace: kcp-system
spec:
  shoot:
    # spec.shoot.name is required
    name: shoot-name
    # spec.shoot.purpose is required
    purpose: production
    # spec.shoot.region is required
    region: eu-central-1
    # spec.shoot.platformRegion is required
    platformRegion: "cd-eu11"
    # spec.shoot.secretBindingName is required
    secretBindingName: "hyperscaler secret"
    # spec.shoot.enforceSeedLocation is optional ; it allows to make sure the seed cluster will be located in the same region as the shoot cluster
    enforceSeedLocation: "true"
    kubernetes:
      # spec.shoot.kubernetes.version is optional, when not provided default will be used
      # Will be modified by the SRE
      version: "1.28.7"
      kubeAPIServer:
        # spec.shoot.kubernetes.kubeAPIServer.oidcConfig is required
        oidcConfig:
          clientID: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
          groupsClaim: groups
          issuerURL: https://my.cool.tokens.com
          signingAlgs:
          - RS256
          usernameClaim: sub
        # spec.shoot.kubernetes.kubeAPIServer.additionalOidcConfig is optional, not implemented in the first KIM release
        additionalOidcConfig:
          - clientID: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
            groupsClaim: groups
            issuerURL: https://some.others.tokens.com
            signingAlgs:
              - RS256
            usernameClaim: sub
            usernamePrefix: 'someother'
    provider:
      # spec.shoot.provider.type is required
      type: aws
      # spec.shoot.provider.workers is required
      workers:
        - machine:
            # spec.shoot.workers.machine.type is required
            type: m6i.large
            # spec.shoot.workers.machine.image is optional, when not provider default will be used
            # Will be modified by the SRE
            image:
              name: gardenlinux
              version: 1312.3.0
          # spec.shoot.workers.volume is required for the first release
          # Probably can be moved into KIM, as it is hardcoded in KEB, and not dependent on plan
          volume:
            type: gp2
            size: 50Gi
          # spec.shoot.workers.zones is required
          zones:
            - eu-central-1a
            - eu-central-1b
            - eu-central-1c
          # spec.shoot.workers.name is optional, if not provided default will be used
          name: cpu-worker-0
          # spec.shoot.workers.minimum is required
          minimum: 3
          # spec.shoot.workers.maximum is required
          maximum: 20
          # spec.shoot.workers.maxSurge is required in the first release.
          # It can be optional in the future, as it equals to zone count
          maxSurge: 3
          # spec.shoot.workers.maxUnavailable is required in the first release.
          # It can be optional in the future, as it is always set to 0
          maxUnavailable: 0
    # spec.shoot.Networking is required
    networking:
      pods: 100.64.0.0/12
      nodes: 10.250.0.0/16
      services: 100.104.0.0/13
    # spec.shoot.controlPlane is required
    controlPlane:
      highAvailability:
        failureTolerance:
          type: zone
  security:
    networking:
      filter:
        # spec.security.networking.filter.egress.enabled is required
        egress:
          enabled: false
        # spec.security.networking.filter.ingress.enabled is optional (default=false), not implemented in the first KIM release
        ingress:
          enabled: true
    # spec.security.administrators is required
    administrators:
      - admin@myorg.com
```
> Note: please mind that the additional OIDC providers, and ingress network filtering will not be implemented in the first release.

Please see the following examples to understand what CRs must be created for particular KEB plans:
- [AWS trial plan](assets/runtime-examples/aws-trial.yaml)
- [Azure](assets/runtime-examples/azure.yaml)
- [Azure lite](assets/runtime-examples/azure-lite.yaml)
- [GCP](assets/runtime-examples/gcp.yaml)
- [SAP Converge Cloud](assets/runtime-examples/sap-converged-cloud.yaml)

## API structures

```go
package v1

import (
	gardener "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Runtime is the Schema for the runtimes API
type Runtime struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RuntimeSpec   `json:"spec,omitempty"`
	Status RuntimeStatus `json:"status,omitempty"`
}

// RuntimeSpec defines the desired state of Runtime
type RuntimeSpec struct {
	Shoot    RuntimeShoot `json:"shoot"`
	Security Security     `json:"security"`
}

// RuntimeStatus defines the observed state of Runtime
type RuntimeStatus struct {
	// State signifies current state of Runtime
	State State `json:"state,omitempty"`
	// List of status conditions to indicate the status of a ServiceInstance.
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

type RuntimeShoot struct {
	Name                string                `json:"name"`
	Purpose             gardener.ShootPurpose `json:"purpose"`
	PlatformRegion      string                `json:"platformRegion"` 
	Region              string                `json:"region"`
	LicenceType         *string               `json:"licenceType,omitempty"`
	SecretBindingName   string                `json:"secretBindingName"`
	EnforceSeedLocation *bool                 `json:"enforceSeedLocation,omitempty"`
	Kubernetes          Kubernetes            `json:"kubernetes"`
	Provider            Provider              `json:"provider"`
	Networking          Networking            `json:"networking"`
	ControlPlane        gardener.ControlPlane `json:"controlPlane"`
}

type Kubernetes struct {
	Version       *string   `json:"version,omitempty"`
	KubeAPIServer APIServer `json:"kubeAPIServer,omitempty"`
}

type APIServer struct {
	OidcConfig           gardener.OIDCConfig    `json:"oidcConfig"`
	AdditionalOidcConfig *[]gardener.OIDCConfig `json:"additionalOidcConfig,omitempty"`
}

type Provider struct {
	Type    string            `json:"type"`
	Workers []gardener.Worker `json:"workers"`
}

type Networking struct {
	Pods     string `json:"pods"`
	Nodes    string `json:"nodes"`
	Services string `json:"services"`
}

type Security struct {
	Administrators []string           `json:"administrators"`
	Networking     NetworkingSecurity `json:"networking"`
}

type NetworkingSecurity struct {
	Filter Filter `json:"filter"`
}

type Filter struct {
	Ingress *Ingress `json:"ingress,omitempty"`
	Egress  Egress   `json:"egress"`
}

type Ingress struct {
	Enabled bool `json:"enabled"`
}

type Egress struct {
	Enabled bool `json:"enabled"`
}

```
