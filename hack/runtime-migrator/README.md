# Benchmark
The `runtime-migrator` script
1. connects to a gardener project
2. retrieves all existing shoot specifications
3. migrates the shoot specs to the new Runtime custom resource
4. saves the new Runtime custom resources to files

## Usage
Run `go run main.go -gardener-kubeconfig-path=/path/to/garden/kubeconfig.yml -gardener-project-name=garden-project-name -output-path=/path/where/to/save/crs` 
