# Enable NVIDIA OpenShell Extension

## Overview

The `shoot-nvidia-openshell` extension is a Gardener extension that provides NVIDIA GPU support with OpenShell capabilities for Kubernetes clusters.
By default, the NVIDIA OpenShell extension is disabled. When enabled, it configures the necessary components to run GPU workloads on your Shoot cluster.

## Prerequisites

1. Your Gardener landscape has the `shoot-nvidia-openshell` extension registered and available
2. Your worker nodes support NVIDIA GPUs (appropriate machine types selected)
3. The necessary NVIDIA GPU drivers are available in your chosen machine image

## Enabling the Extension

To enable the NVIDIA OpenShell extension, add the **enableNvidiaOpenshell** field to your Runtime Shoot specification and set it to `true`.

```yaml
apiVersion: infrastructuremanager.kyma-project.io/v1
kind: Runtime
metadata:
  name: my-runtime
  namespace: kcp-system
spec:
  shoot:
    name: my-shoot
    # ... other shoot fields ...
    enableNvidiaOpenshell: true
  # ... other spec fields ...
```

You can find an example showing the **enableNvidiaOpenshell** field in the main Runtime sample at [`config/samples/infrastructuremanager_v1_runtime.yaml`](../../samples/infrastructuremanager_v1_runtime.yaml). By default, the field is commented out. To enable the extension, uncomment the field and set it to `true`.
To disable the extension, use one of the following options:
- Set `enableNvidiaOpenshell: false` in the Shoot specification
- Remove the **enableNvidiaOpenshell** field entirely

## Implementation Details

The extension is implemented in the following files:
- `api/v1/runtime_types.go` - API types for the extension configuration
- `pkg/gardener/shoot/extender/extensions/nvidia_openshell.go` - Extension implementation
- `pkg/gardener/shoot/extender/extensions/extender.go` - Integration with the extension extender framework

When enabled, the extension is added to the Gardener Shoot specification with:
- `Type: shoot-nvidia-openshell`
- `Disabled: false`
