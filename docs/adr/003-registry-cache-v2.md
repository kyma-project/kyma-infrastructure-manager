# Context
This document defines the refined architecture and API for enabling registry cache in the Kyma runtime.

# Status
Proposed

# Requirements

- The user should be able to configure the cache for specific image registries.
- Credentials for the image registries should be stored in secrets on SKR, and transferred to Garden cluster.
- The user should be able to view the status of the registry cache configuration.
- The Gardener extension `registry-cache` will be used to implement the registry cache functionality.
- In the first phase KIM, will periodically pull the registry cache configuration from SKR.
- At some point, the Runtime Watcher is used to trigger events to notify KIM that the configuration has changed.

# Decision

The overall architecture is based on the following proposal: https://github.com/kyma-project/community/issues/992. The following diagram shows the proposed architecture:

![](./assets/caching-in-kim-v2.drawio.svg)

Highlights:
- The registry cache configuration is stored in the `RegistryCacheConfig` CRD in the SKR cluster. The `RegistryCacheConfig` CRD is a part of the Kyma module.
- The Registry Cache Controller synchronizes the registry cache configuration between SKR and the Runtime CR.
- The Runtime Controller:
  - Configures the `registry-cache` extension 
  - Synchronizes credential secrets between SKR and the Garden cluster 
  - Updates the status of the registry cache configuration in SKR
  - Removes credentials secrets from the Garden cluster when the registry cache configuration is removed from the SKR.
- The Runtime Watcher triggers events to notify KIM that the configuration changed.
- The Registry Cache Webhook validates the registry cache configuration in the Registry Cache CR.

The operation flow is as follows:
1. The user creates a `RegistryCacheConfig` CR with the cache configuration in the SKR.
2. The Registry Cache CR validating webhook verifies the configuration.
3. The Runtime Watcher triggers an event notifying KIM of the new configuration.
4. The Registry Cache Config Controller (if the Registry Cache module is enabled) reads the configuration from the `RegistryCacheConfig` CR and applies it to the Runtime CR.
5. The Runtime Controller synchronizes credential secrets between the SKR and the Garden cluster, and sets the registry cache configuration status in the SKR to `Pending`.
6. The Runtime Controller configures the `registry-cache` extension.
7. If the registry cache configuration is removed from the SKR, the Runtime Controller cleans up secrets from the Garden cluster.
8. The Runtime Controller updates the registry cache configuration status in the SKR to Ready.

## API

The `RegistryCacheConfig` CRD is defined as follows.

```go
type RegistryCacheConfigSpec struct {
	// Upstream is the remote registry host to cache.
	Upstream string `json:"upstream"`
	// RemoteURL is the remote registry URL. The format must be `<scheme><host>[:<port>]` where
	// `<scheme>` is `https://` or `http://` and `<host>[:<port>]` corresponds to the Upstream
	//
	// If defined, the value is set as `proxy.remoteurl` in the registry [configuration](https://github.com/distribution/distribution/blob/main/docs/content/recipes/mirror.md#configure-the-cache)
	// and in containerd configuration as `server` field in [hosts.toml](https://github.com/containerd/containerd/blob/main/docs/hosts.md#server-field) file.
	// +optional
	RemoteURL *string `json:"remoteURL,omitempty"`
	// Volume contains settings for the registry cache volume.
	// +optional
	Volume *Volume `json:"volume,omitempty"`
	// GarbageCollection contains settings for the garbage collection of content from the cache.
	// Defaults to enabled garbage collection.
	// +optional
	GarbageCollection *GarbageCollection `json:"garbageCollection,omitempty"`
	// SecretReferenceName is the name of the reference for the Secret containing the upstream registry credentials.
	// +optional
	SecretReferenceName *string `json:"secretReferenceName,omitempty"`
	// Proxy contains settings for a proxy used in the registry cache.
	// +optional
	Proxy *Proxy `json:"proxy,omitempty"`

	// HTTP contains settings for the HTTP server that hosts the registry cache.
	HTTP *HTTP `json:"http,omitempty"`
}

// Volume contains settings for the registry cache volume.
type Volume struct {
	// Size is the size of the registry cache volume.
	// Defaults to 10Gi.
	// This field is immutable.
	// +optional
	// +default="10Gi"
	Size *resource.Quantity `json:"size,omitempty"`
	// StorageClassName is the name of the StorageClass used by the registry cache volume.
	// This field is immutable.
	// +optional
	StorageClassName *string `json:"storageClassName,omitempty"`
}

// GarbageCollection contains settings for the garbage collection of content from the cache.
type GarbageCollection struct {
	// TTL is the time to live of a blob in the cache.
	// Set to 0s to disable the garbage collection.
	// Defaults to 168h (7 days).
	// +default="168h"
	TTL metav1.Duration `json:"ttl"`
}

// Proxy contains settings for a proxy used in the registry cache.
type Proxy struct {
	// HTTPProxy field represents the proxy server for HTTP connections which is used by the registry cache.
	// +optional
	HTTPProxy *string `json:"httpProxy,omitempty"`
	// HTTPSProxy field represents the proxy server for HTTPS connections which is used by the registry cache.
	// +optional
	HTTPSProxy *string `json:"httpsProxy,omitempty"`
}

// HTTP contains settings for the HTTP server that hosts the registry cache.
type HTTP struct {
	// TLS indicates whether TLS is enabled for the HTTP server of the registry cache.
	// Defaults to true.
	TLS bool `json:"tls,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// RegistryCacheConfig is the Schema for the registrycacheconfigs API.
type RegistryCacheConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RegistryCacheConfigSpec   `json:"spec,omitempty"`
	Status RegistryCacheConfigStatus `json:"status,omitempty"`
}
```

The `Runtime` CRD contains the following properties:

```go
// RuntimeSpec defines the desired state of Runtime
type RuntimeSpec struct {
	Shoot    RuntimeShoot         `json:"shoot"`
	Security Security             `json:"security"`
	Caching  []ImageRegistryCache `json:"imageRegistryCache,omitempty"`
}

type ImageRegistryCache struct {
	Name      string                                `json:"name"`
	Namespace string                                `json:"namespace"`
	UID       string                                `json:"uid"`
	Config    registrycache.RegistryCacheConfigSpec `json:"config"`
}

```
