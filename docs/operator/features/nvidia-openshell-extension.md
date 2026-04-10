# Enable NVIDIA OpenShell Extension

This document describes how to enable the `shoot-nvidia-openshell` Gardener extension in Kyma Infrastructure Manager (KIM).

## Overview

The `shoot-nvidia-openshell` extension is a Gardener extension that provides NVIDIA GPU support with OpenShell capabilities for Kubernetes clusters. When enabled, it configures the necessary components to run GPU workloads on your Shoot cluster.

## Enabling the Extension

To enable the NVIDIA OpenShell extension, add the **enableNvidiaOpenshell** field to your Runtime Shoot specification.

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

## Example

You can find an example showing the **enableNvidiaOpenshell** field in the main Runtime sample at `config/samples/infrastructuremanager_v1_runtime.yaml`.

By default, the field is commented out. To enable the extension, uncomment the field and set it to `true`.

## Implementation Details

The extension is implemented in the following files:
- `api/v1/runtime_types.go` - API types for the extension configuration
- `pkg/gardener/shoot/extender/extensions/nvidia_openshell.go` - Extension implementation
- `pkg/gardener/shoot/extender/extensions/extender.go` - Integration with the extension extender framework

When enabled, the extension is added to the Gardener Shoot specification with:
- `Type: shoot-nvidia-openshell`
- `Disabled: false`

## Default Behavior

By default, the NVIDIA OpenShell extension is disabled. To activate it, you must explicitly set it to `true` in the Runtime Shoot specification.

To disable the extension after it has been enabled, use one of the following options:
- Set `enableNvidiaOpenshell: false` in the Shoot specification
- Remove the **enableNvidiaOpenshell** field entirely

## Prerequisites

1. Your Gardener landscape has the `shoot-nvidia-openshell` extension registered and available
2. Your worker nodes support NVIDIA GPUs (appropriate machine types selected)
3. The necessary NVIDIA GPU drivers are available in your chosen machine image
