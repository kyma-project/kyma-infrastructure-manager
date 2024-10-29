# Runtime Migrator
The `runtime-migrator` application
1. connects to a Gardener project
2. retrieves all existing shoot specifications
3. migrates the shoot specs to the new Runtime custom resource (Runtime CRs created with this migrator have the `operator.kyma-project.io/created-by-migrator=true` label)
4. saves the new Runtime custom resources to files
5. applies the new Runtime custom resources to the designated KCP cluster

## Build

In order to build the app, run the `go build` in `/hack/runtime-migrator` directory.

## Usage

```bash
cat input/runtimeIds.json | ./runtime-migrator \
  -gardener-kubeconfig-path=/Users/myuser/gardener-kubeconfig.yml \
  -gardener-project-name=kyma-dev  \
  -kcp-kubeconfig-path=/Users/myuser/kcp-kubeconfig.yml \
  -output-path=/tmp/ \
  -dry-run=true \
  -input-type=json 1> /tmp/stdout.txt 2> /tmp/stderr.txt
```

The above **execution example** will: 
1. take the stdin input (json with runtimeIds array)
1. proceed only with Runtime CRs creation for clusters listed in the input 
1. save generated Runtime CRs yamls in `/tmp/` directory. They will not be applied on the KCP cluster (`dry-run` mode)
1. send logs and errors to `/tmp/stderr.txt`
1. send json output to `/tmp/stdout.txt`

### Json output example

```json
[
    {
        "runtimeId": "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
        "shootName": "shoot-name1",
        "status": "Success",
        "pathToCRYaml": "/tmp/shoot-shoot-name1.yaml"
    },
    {
        "runtimeId": "",
        "shootName": "shoot-name2",
        "status": "Error",
        "errorMessage": "Shoot networking pods is nil"
    }
]
```

## Configurable Parameters

This table lists the configurable parameters, their descriptions, and default values:

| Parameter | Description                                                                                                                                                                                                                | Default value                  |
|-----------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|--------------------------------|
| **kcp-kubeconfig-path** | Path to the Kubeconfig file of KCP cluster.                                                                                                                                                                                | `/path/to/kcp/kubeconfig`      |
| **gardener-kubeconfig-path** | Path to the Kubeconfig file of Gardener cluster.                                                                                                                                                                           | `/path/to/gardener/kubeconfig` |
| **gardener-project-name** | Name of the Gardener project.                                                                                                                                                                                              | `gardener-project-name`        |
| **output-path** | Path where generated yamls will be saved. Directory has to exist.                                                                                                                                                          | `/tmp/`                        |
| **dry-run** | Dry-run flag. Has to be set to **false**, otherwise migrator will not apply the CRs on the KCP cluster.                                                                                                       | `true`                         |
| **input-type** | Type of input to be used. Possible values: **all** (will migrate all Gardener shoots), and **json** (will migrate only clusters whose runtimeIds were passed as an input, [see the example](input/runtimeids_sample.json)). | `json`                         |

