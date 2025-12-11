
# CredentialsBindings Migration Tool

This tool migrates Gardener `SecretBinding` resources to `CredentialsBinding` resources in a specified Gardener project namespace.

## Usage

### Prerequisites

- Go installed
- Access to the Gardener cluster (kubeconfig)
- The Gardener API and Security API must be available

### Build

```sh
go build -o credentialbindings main.go
```

### Run

```sh
./credentialbindings \
  -gardener-kubeconfig-path=/path/to/gardener/kubeconfig \
  -gardener-project-name=my-project \
  -dry-run=true
```

#### Arguments

- `-gardener-kubeconfig-path` - Path to the kubeconfig file for accessing the Gardener cluster. **Default:** `/gardener/kubeconfig/kubeconfig`
- `-gardener-project-name` - Name of the Gardener project (without the `garden-` prefix). **Default:** `gardener-project`
- `-dry-run` - If set to `true`, the tool will only print the `CredentialsBinding` resources that would be created, without actually creating them. **Default:** `true`

### Example

To perform a dry run for project `foo`:

```sh
./credentialbindings -gardener-project-name=foo -dry-run=true
```

To actually create the `CredentialsBinding` resources:

```sh
./credentialbindings -gardener-project-name=foo -dry-run=false
```

## What It Does

- Lists all `SecretBinding` resources in the specified Gardener project namespace.
- For each `SecretBinding`, creates a corresponding `CredentialsBinding` resource.
- In dry-run mode, prints the resources that would be created.
- In non-dry-run mode, creates the resources in the cluster.

## Notes

- The tool assumes the Gardener project namespace is named `garden-<project-name>`.
- Make sure you have the necessary permissions to create resources in the Gardener cluster.

