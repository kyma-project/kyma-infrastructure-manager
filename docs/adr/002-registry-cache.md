# Context
This document defines the architecture and API for enabling registry cache in the Kyma runtime. 

# Status
Proposed

# Requirements

- The Kyma runtime should be able to pull images from a local cache instead of directly from the registry.
- The user should be able to configure the cache for each image registry.
- Credentials for the image registries should be stored in a secret on the SKR. 
- The Gardener's `registry-cache` extension will be used to implement the registry cache functionality.
- In the first phase it is acceptable to periodically pull the cache configuration from the SKR.
- Finally, we will use the Runtime Watcher to trigger events to notify KIM that configuration changed.

# Options

## Option 1: enhance Runtime Controller to configure `registry-cache` extension

It seems to be obvious to implement the configuration of the `registry-cache` extension in the Runtime Controller. However, there are the following questions:
- If we introduce the property on Runtime CR that will enable the cache who will set the property? There could be a checkbox in the BTP, but it would require two steps (creating CR on the SKR and enabling cache in the BTP).
- If we don't add the property to the Runtime CR, how we will implement time based reconciliation? Currently, reconciliation loop for the Runtime CR is triggered by the KEB.

Pros:
- Seems like the firsts right approach.
- Adding Runtime Watcher integration seems to be easy.

Cons:
- If we pick an option without adding the property to the Runtime CR, the controller will do some steps that are not explicitly defined in the CR.
- Having control loop triggerred by both CR modifications, and time based reconciliation seems to be difficult to implement, and maintain.

## Option 2: implement periodic checks with go routines and directly modify the shoot to configure `registry-cache` extension

The second approach would be to implement a go routine that periodically checks all the SKRs for the cache configuration.

Pros:
- Seems to be the simplest solution. Go primitives such as go routines and channels are handy for such tasks.

Cons:
- KIM uses controller pattern for performing operations, so it would be a bit of a hack to implement this in the current architecture.
- Some implementation effort needed to synchronise access to multiple runtimes, and implement proper error handling.
- Directly modifying the shoot can generate conflicts with the Runtime controller.
- Responsibilities are not clearly defined. KIM sets everything up  but `registry-cache` extension
- The implementation is not scalable out of the box.
- Adding Runtime Watcher will require to slightly modify the implementation (e.g. some new channel will be needed to notify there is a configuration change)

## Options 3: implement a new controller for the `registry-cache` extension

The third approach would be to implement a new controller that will be responsible for configuring the `registry-cache` extension. 
Assumptions:
- New property for the Runtime CR will be added to enable the cache.
- Runtime Controller's FSM will be enhanced to handle the new property, and configure the `registry-cache` extension.
- The new controller will be listening on the Runtime CR events, and will be responsible for setting up the enable cache property on the Runtime CR.
- The new controller will be triggerred in time based manner to periodically check the SKRs for the cache configuration.

Pros:
- Responsibilities are clearly defined. The new controller reads the SKR to find the configuration and sets up the enable cache property.
- It is easy to implement time based reconciliation in a separate controller.
- Adding Runtime Watcher integration seems to be easy.

Cons:
- It seems to be a bit unclear why we have two controllers on Runtime resource. We could try to listen on another resource (e.g. Kyma) but it could be also not clear why Kyma Infrastructure Manager needs to do that.

# Decision

The following diagram shows the proposed architecture:

![](./assets/caching-in-kim.drawio.svg)

The following assumptions were taken:

- The user will be responsible for creating the custom resource with the configuration .
- In the first implementation phase KIM will periodically pull registry cache configuration from the SKR.
- In the second phase the Runtime Watcher will be used to trigger events to notify KIM that configuration changed.

## API proposal

### Runtime CR modification

```go
type RuntimeSpec struct {
	Shoot    RuntimeShoot        `json:"shoot"`
	Security Security            `json:"security"`
	Caching  *ImageRegistryCache `json:"imageRegistryCache,omitempty"`
}

type ImageRegistryCache struct {
	Enabled bool `json:"enabled"`
}
```

### Custom Resource for the cache configuration

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
	RegistryCache []registrycache.RegistryCache `json:"cache"`
}
```
