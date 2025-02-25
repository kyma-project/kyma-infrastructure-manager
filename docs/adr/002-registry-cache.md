# Context
This document defines the architecture and API for enabling registry cache in the Kyma runtime. 

# Status
Proposed

# Decision

The following diagram shows the proposed architecture:

The following assumptions were taken:
- The Gardener's `registry-cache` extension will be used to implement the registry cache functionality.
- The credentials for the registry cache will be stored in the custom resource stored on the SKR.
- The user will be responsible for creating the custom resource with the credentials for the registry cache.
- In the first implementation phase the controller on the KCP will be triggered periodically.
- In the second phase the Runtime Watcher will be used to trigger the controller on the KCP.

## API proposal

