# Context
This document defines the architecture and API for enabling registry cache in the Kyma runtime. 

# Status
Proposed

# Requirements

- The user should be able to configure the cache for specific image registries.
- Credentials for the image registries should be stored in secrets on SKR. 
- The Gardener extension `registry-cache` will be used to implement the registry cache functionality.
- In the first phase KIM, will periodically pull the registry cache configuration from SKR.
- At some point, the Runtime Watcher will be used to trigger events to notify KIM that the configuration changed.

# Options

The implementation of the registry cache in the KIM can be done in several ways. We considered following options:
- Option 1: enhance Runtime Controller to configure `registry-cache` extension
- Option 2: implement periodic checks with go routines and directly modify the shoot to configure `registry-cache` extension
- Option 3: implement a new controller for the `registry-cache` extension

## Option 1: Enhance Runtime Controller to Configure the `registry-cache` Extension

It seems to be obvious to implement the configuration of the `registry-cache` extension in the Runtime Controller. However, there are the following questions:
- Should the Runtime CR have a property to enable the cache? Who will set this property?
- How do we implement time-based reconciliation? What will be the impact on the existing implementation?

Conclusions:
- Introducing a new property to the Runtime CR does not seem to be a good idea. It would require two steps from the user: preparing configuration on SKR, and enabling caching in the BTP.
- Currently, in some states of the state machine we don't requeue the event. As a result of introducing time-based reconciliation, we would need to change the implementation of the state machine.

### Summary

Pros:
- Seems like the right approach because Runtime Controller is fully responsible for configuring the shoot.
- Integrating the Runtime Watcher will be relatively easy. 

Cons:
- The controller will apply some changes that are not explicitly defined in the Runtime CR. So the Runtime CR is no longer a full description of the desired state.
- Because the control loop is triggered by both CR modifications, and time-based reconciliation changes to the state machine code are required, it increases the complexity of the Runtime Controller implementation. 

## Option 2: Implement Periodic Checks with Go Routines and Directly Modify the Shoot to Configure `registry-cache` Extension

The second approach would be to implement a Go routine that periodically checks all the SKRs for the cache configuration.

There are the following questions:
- Should we directly modify the shoot to configure the `registry-cache` extension?
- What will be the impact on the existing implementation?

Conclusions:
- Direct shoot modification violates the architecture. The Runtime Controller should be fully responsible for configuring the shoot.
- The implementation will require introducing a new property to the Runtime CR to enable the cache.

### Summary

Pros:
- Go primitives such as Go routines and channels are handy for such tasks.

Cons:
- KIM uses a controller pattern for performing operations, so it would be a bit inconsistent to implement such a feature in a different way.
- Some implementation effort is needed to synchronise access to multiple runtimes, and implement proper error handling.
- Adding Runtime Watcher will require to slightly modify the implementation: A new channel will be needed to notify about the new configuration.

## Option 3: Implement a New Controller for The `registry-cache` Extension

The third approach would be to implement a new controller that will be responsible for configuring the `registry-cache` extension. 

Questions:
- What resource should the new controller listen on?
- Should we directly modify the shoot to configure the `registry-cache` extension?
- What will be the impact on the existing implementation?

Conclusions:
- Direct shoot modification violates the architecture. The Runtime Controller should be fully responsible for configuring the shoot.
- The implementation will require introducing a new property to the Runtime CR to enable the cache.
- The new controller will be listening on the secret events, and will be responsible for setting up the enable cache property on the Runtime CR.
- The new controller will be triggered in time-based manner to periodically check the SKRs for the cache configuration.

### Summary
Pros:
- Responsibilities are clearly defined. The new controller reads the SKR to find the configuration and sets up the enable cache property.
- It is easy to implement time-based reconciliation in a separate controller.
- Integrating the Runtime Watcher will be relatively easy.

Cons:
- At first glance, it seems like overkill to introduce a new controller for such a simple feature.

# Decision


  The third option was selected for the following reasons:
- A clear responsibility separation between the Runtime Controller and the new controller will improve the maintainability of the code.
- The impact on the existing implementation is minimal.
- Introducing a new property to the Runtime CR to enable caching aligns with the controller pattern.

The following diagram shows the proposed architecture:

![](./assets/caching-in-kim.drawio.svg)

The operation flow is as follows:
1. The user creates a CustomConfig CR with the cache configuration.
2. The Runtime Watcher triggers an event that notifies the KIM about the new configuration.
3. CustomConfig Controller reads the cache configuration from the CustomConfig CR and sets up the enable cache property on the Runtime CR.
4. The Runtime Controller reads CustomConfig CR and sets up `registry-cache` extension.

## API proposal

### Runtime CR modification

The Runtime CR will be modified to include the following:
```go
type RuntimeSpec struct {
	Shoot    RuntimeShoot        `json:"shoot"`
	Security Security            `json:"security"`
	ImageRegistryCache  *ImageRegistryCache `json:"imageRegistryCache,omitempty"`
}

type ImageRegistryCache struct {
	Enabled bool `json:"enabled"`
}
```

### Custom Resource for the Cache Configuration

The following assumptions were taken:
- The CR will be deployed as a part of the `kim-snatch` project.
- The Go types for the CR will not directly use the `registry-cache` extension types. Instead, we will define our own types that will be used to configure the cache. 

```go
type CustomConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CustomConfigSpec      `json:"spec,omitempty"`
	Status GardenerClusterStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

type CustomConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CustomConfig `json:"items"`
}

type CustomConfigSpec struct {
	RegistryCache []RegistryCache `json:"cache"`
}

type RegistryCache struct {
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
}

// Volume contains settings for the registry cache volume.
type Volume struct {
    // Size is the size of the registry cache volume.
    // Defaults to 10Gi.
    // This field is immutable.
    // +optional
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
```
