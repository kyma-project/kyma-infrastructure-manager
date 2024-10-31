# Runtime Migrator
The `runtime-migrator` application
1. connects to a Gardener project
2. retrieves all existing shoot specifications
3. migrates the shoot specs to the new Runtime custom resource (Runtime CRs created with this migrator have the `operator.kyma-project.io/created-by-migrator=true` label)
4. saves the new Runtime custom resources to files
5. checks if the new Runtime custom resource will not cause update on the Gardener
6. saves the results of the comparison between original shoot and the shoot KIM will produce based on new Runtime custom resource
7. applies the new Runtime custom resources to the designated KCP cluster
8. saves the results migration in the output json file

## Build

In order to build the app, run the following command:

```bash
go build -o ./bin/runtime-migrator ./cmd
``` 

## Usage

```bash
cat input/runtimeIds.json | ./runtime-migrator \
  -gardener-kubeconfig-path=/Users/myuser/gardener-kubeconfig.yml \
  -gardener-project-name=kyma-dev  \
  -kcp-kubeconfig-path=/Users/myuser/kcp-kubeconfig.yml \
  -output-path=/tmp/ \
  -dry-run=true \
  -input-type=json
```

The above **execution example** will: 
1. take the stdin input (json with runtimeIds array)
1. proceed only with Runtime CRs creation for clusters listed in the input 
1. save output files in `/tmp/` directory. The output directory will contain the following content:
    - `migration.json` - the output file with the migration results
    - `runtimes` - the directory with the Runtime CRs files
    - `comparison-results` - the directory with the files generated during the comparison process
1. They will not be applied on the KCP cluster (`dry-run` mode)


### Output example

```
2024/10/28 13:38:49 INFO Starting runtime-migrator
2024/10/28 13:38:49 gardener-kubeconfig-path: /Users/i326211/Downloads/kubeconfig-garden-kyma-dev.yaml
2024/10/28 13:38:49 kcp-kubeconfig-path: /Users/i326211/dev/config/sap
2024/10/28 13:38:49 gardener-project-name: kyma-dev
2024/10/28 13:38:49 output-path: /tmp/
2024/10/28 13:38:49 dry-run: true
2024/10/28 13:38:49 input-type: json
2024/10/28 13:38:49
2024/10/28 13:38:49 INFO Migrating runtimes
2024/10/28 13:38:49 INFO Reading runtimeIds from stdin
2024/10/28 13:38:49 INFO Migrating runtime with ID: 80dfc8d7-6687-41b4-982c-2292afce5ac9
2024/10/28 13:39:01 WARN Runtime CR can cause unwanted update in Gardener. Please verify the runtime CR. runtimeID=80dfc8d7-6687-41b4-982c-2292afce5ac9
2024/10/28 13:39:01 INFO Migration completed. Successfully migrated runtimes: 0, Failed migrations: 0, Differences detected: 1
2024/10/28 13:39:01 INFO Migration results saved in: /tmp/migration-2024-10-28T13:38:49+01:00/migration-results.json
```

The above example shows that the migration process detected potential problem with Runtime CR. In such a case the Runtime CR that may cause unwanted updates on Gardener will not be applied on the cluster and require manual intervention.
The migration results are saved in the `/tmp/migration-2024-10-28T13:38:49+01:00/migration-results.json` file.

The `migration-results.json` file contains the following content:
```json
[
  {
    "runtimeId": "80dfc8d7-6687-41b4-982c-2292afce5ac9",
    "shootName": "c6069ce",
    "status": "ValidationError",
    "errorMessage": "Runtime may cause unwanted update in Gardener. Please verify the runtime CR.",
    "runtimeCRFilePath": "/tmp/migration-2024-10-28T13:38:49+01:00/runtimes/80dfc8d7-6687-41b4-982c-2292afce5ac9.yaml",
    "comparisonResultDirPath": "/tmp/migration-2024-10-28T13:38:49+01:00/comparison-results/80dfc8d7-6687-41b4-982c-2292afce5ac9"
  }
]
```
The runtime custom resource is saved in the `/tmp/migration-2024-10-28T13:38:49+01:00/runtimes/80dfc8d7-6687-41b4-982c-2292afce5ac9.yaml` file. 

The `comparison-results` directory contains the following content:
```
drwxr-xr-x@ 5 i326211  wheel    160 28 paź 13:39 .
drwxr-xr-x@ 3 i326211  wheel     96 28 paź 13:39 ..
-rw-r--r--@ 1 i326211  wheel   1189 28 paź 13:39 c6069ce.diff
-rw-r--r--@ 1 i326211  wheel   3492 28 paź 13:39 converted-shoot.yaml
-rw-r--r--@ 1 i326211  wheel  24190 28 paź 13:39 original-shoot.yaml
```

The `c6069ce.diff` file contains the differences between the original shoot and the shoot that will be created based on the new Runtime CR. The `converted-shoot.yaml` file contains the shoot that will be created based on the new Runtime CR. The `original-shoot.yaml` file contains the shoot fetched from the Gardener.

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

