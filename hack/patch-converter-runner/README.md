# Patch Converter Runner 

## Overview

The `patch-converter-runner` program invokes converter library to generate shoot from runtime resources. The purpose of this application is to allow developers to:
- quickly reproduce problems with updating existing Gardener clusters
- debug converter without a need to run the whole KIM system

## Building the Binary

To build the `patch-converter-runner` binary navigate to the `hack/patch-converter-runner` folder, and execute the following command:

```sh
  go build -o ./bin/patch-converter-runner ./cmd/main.go
```

## Usage

### Options

- `--runtime-path`: The path to the runtime CR file
- `--shoot-path`: The path to the existing shoot CR file
- `--kcp-kubeconfig-path`: The path to the KCP kubeconfig file
- `--output-path`: The path to the output shoot CR file

### Example

To run the `patch-converter-runner` program, execute the following command in the `hack/patch-converter-runner` folder:
```sh
./bin/patch-converter-runner -runtime-path=<runtime CR fie path> -shoot-path=<shoot CR path> -kcp-kubeconfig-path=<KCP kubeconfig path> -output-path=<output shoot CR path>
```

To run the `patch-converter-runner` program with `go run`, execute the following command in the `hack/patch-converter-runner` folder:
```sh
go run ./cmd/main.go -runtime-path=<runtime CR fie path> -shoot-path=<shoot CR path> -kcp-kubeconfig-path=<KCP kubeconfig path> -output-path=<output shoot CR path>
```

### Helper script

You can also run `./script/run.sh` script to execute the application. Before running the script you need to setup the following environment variables:
- `KCP_KUBECONFIG` - the path to the KCP kubeconfig file
- `GARDENER_KUBECONFIG` - the path to the Gardener kubeconfig file
- `GARDENER_NAMESPACE` - the namespace where the shoot resources are stored

To run the script navigate to the `hack/patch-converter-runner` folder, and execute the following command:
```sh
./script/run.sh <output folder> <runtime ID> <shoot name>
```