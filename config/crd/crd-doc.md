# API Reference

## Packages
- [infrastructuremanager.kyma-project.io/v1](#infrastructuremanagerkyma-projectiov1)


## infrastructuremanager.kyma-project.io/v1

Package v1 contains API Schema definitions for the infrastructuremanager v1 API group

### Resource Types
- [GardenerCluster](#gardenercluster)
- [Runtime](#runtime)



#### APIServer







_Appears in:_
- [Kubernetes](#kubernetes)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `oidcConfig` _[OIDCConfig](#oidcconfig)_ |  |  |  |
| `additionalOidcConfig` _[OIDCConfig](#oidcconfig)_ |  |  |  |






#### Egress



Egress filtering is a default filtering mode for `shoot-networking-fitler` extension.



_Appears in:_
- [Filter](#filter)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `enabled` _boolean_ |  |  |  |


#### Filter







_Appears in:_
- [NetworkingSecurity](#networkingsecurity)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `ingress` _[Ingress](#ingress)_ |  |  |  |
| `egress` _[Egress](#egress)_ |  |  |  |


#### GardenerCluster



GardenerCluster is the Schema for the clusters API





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `infrastructuremanager.kyma-project.io/v1` | | |
| `kind` _string_ | `GardenerCluster` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  |  |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  |  |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.25/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[GardenerClusterSpec](#gardenerclusterspec)_ |  |  |  |


#### GardenerClusterSpec



GardenerClusterSpec defines the desired state of GardenerCluster



_Appears in:_
- [GardenerCluster](#gardenercluster)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `kubeconfig` _[Kubeconfig](#kubeconfig)_ |  |  |  |
| `shoot` _[Shoot](#shoot)_ |  |  |  |


#### ImageRegistryCache







_Appears in:_
- [RuntimeSpec](#runtimespec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ |  |  |  |
| `namespace` _string_ |  |  |  |
| `uid` _string_ |  |  |  |
| `config` _[RegistryCacheConfigSpec](#registrycacheconfigspec)_ |  |  |  |


#### Ingress



Ingress filtering can be enabled for `shoot-networking-fitler` extension with
the blackholing feature, see https://github.com/gardener/gardener-extension-shoot-networking-filter/blob/master/docs/usage/shoot-networking-filter.md#ingress-filtering



_Appears in:_
- [Filter](#filter)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `enabled` _boolean_ | It means that the blackholing filtering is enabled on the per shoot level. |  |  |


#### Kubeconfig



Kubeconfig defines the desired kubeconfig location



_Appears in:_
- [GardenerClusterSpec](#gardenerclusterspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `secret` _[Secret](#secret)_ |  |  |  |


#### Kubernetes







_Appears in:_
- [RuntimeShoot](#runtimeshoot)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `version` _string_ |  |  |  |
| `kubeAPIServer` _[APIServer](#apiserver)_ |  |  |  |


#### Networking







_Appears in:_
- [RuntimeShoot](#runtimeshoot)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `type` _string_ |  |  |  |
| `pods` _string_ |  |  |  |
| `nodes` _string_ |  |  |  |
| `services` _string_ |  |  |  |


#### NetworkingSecurity







_Appears in:_
- [Security](#security)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `filter` _[Filter](#filter)_ |  |  |  |


#### OIDCConfig



OIDCConfig contains configuration settings for the OIDC provider.
Note: Descriptions were taken from the Kubernetes documentation.



_Appears in:_
- [APIServer](#apiserver)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `OIDCConfig` _[OIDCConfig](#oidcconfig)_ |  |  |  |
| `jwks` _integer array_ |  |  |  |


#### Provider







_Appears in:_
- [RuntimeShoot](#runtimeshoot)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `type` _string_ |  |  | Enum: [aws azure gcp openstack] <br /> |
| `workers` _Worker array_ |  |  |  |
| `additionalWorkers` _[Worker](#worker)_ |  |  |  |
| `controlPlaneConfig` _[RawExtension](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.25/#rawextension-runtime-pkg)_ |  |  |  |
| `infrastructureConfig` _[RawExtension](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.25/#rawextension-runtime-pkg)_ |  |  |  |


#### Runtime



Runtime is the Schema for the runtimes API





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `infrastructuremanager.kyma-project.io/v1` | | |
| `kind` _string_ | `Runtime` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  |  |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  |  |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.25/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[RuntimeSpec](#runtimespec)_ |  |  |  |






#### RuntimeShoot







_Appears in:_
- [RuntimeSpec](#runtimespec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ |  |  |  |
| `purpose` _[ShootPurpose](#shootpurpose)_ |  |  |  |
| `platformRegion` _string_ |  |  |  |
| `region` _string_ |  |  |  |
| `licenceType` _string_ |  |  |  |
| `secretBindingName` _string_ |  |  |  |
| `enforceSeedLocation` _boolean_ |  |  |  |
| `kubernetes` _[Kubernetes](#kubernetes)_ |  |  |  |
| `provider` _[Provider](#provider)_ |  |  |  |
| `networking` _[Networking](#networking)_ |  |  |  |
| `controlPlane` _[ControlPlane](#controlplane)_ |  |  |  |


#### RuntimeSpec



RuntimeSpec defines the desired state of Runtime



_Appears in:_
- [Runtime](#runtime)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `shoot` _[RuntimeShoot](#runtimeshoot)_ |  |  |  |
| `security` _[Security](#security)_ |  |  |  |
| `imageRegistryCache` _[ImageRegistryCache](#imageregistrycache) array_ |  |  |  |


#### Secret



SecretKeyRef defines the location, and structure of the secret containing kubeconfig



_Appears in:_
- [Kubeconfig](#kubeconfig)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ |  |  |  |
| `namespace` _string_ |  |  |  |
| `key` _string_ |  |  |  |


#### Security







_Appears in:_
- [RuntimeSpec](#runtimespec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `administrators` _string array_ |  |  |  |
| `networking` _[NetworkingSecurity](#networkingsecurity)_ |  |  |  |


#### Shoot



Shoot defines the name of the Shoot resource



_Appears in:_
- [GardenerClusterSpec](#gardenerclusterspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ |  |  |  |




