# Runtime backup and switch

The `runtime-backup-and-switch` application
1. connects to a Gardener project, and KCP cluster
2. retrieves all existing shoot specifications
3. for each runtime on input list:
  - a) gets shoot, Cluster Role Bindings and OpenIDConnect resources 
  - b) saves the backup on a disk
  - c) marks Cluster Role Bindings that were created by the Provisioner with `kyma-project.io/deprecation` label
  - d) sets `kyma-project.io/controlled-by-provisioner` label with `false` value in order to make sure KIM will control the runtime

## Build

In order to build the app, run the following command:

```bash
go build -o ./bin/runtime-backup-and-switch ./cmd/backup-and-switch
``` 

## Usage

### Dry run
```bash
./bin/runtime-backup-and-switch \
  -gardener-kubeconfig-path=/Users/myuser/gardener-kubeconfig.yml \
  -gardener-project-name=kyma-dev  \
  -kcp-kubeconfig-path=/Users/myuser/kcp-kubeconfig.yml \
  -output-path=/tmp/ \
  -dry-run=true \
  -input-file-path=input/runtimeIds.txt \
  -input-type=txt
```

The above **execution example** will:
1. take the input from the `input/runtimeIds.txt` file (each raw contains single `RuntimeID`)
1. proceed only with fetching Shoot, Cluster Role Bindings and OpenIDConnect resources
1. save output files in the `/tmp/<generated name>` directory. The output directory contains the following:
    - `backup-and-switch-results.json` - the output file with the backup results
    - `backup` - the directory with the backup files
   
### Backup and switch Runtime to be controlled by KIM

```bash
./bin/runtime-backup-and-switch \
  -gardener-kubeconfig-path=/Users/myuser/gardener-kubeconfig.yml \
  -gardener-project-name=kyma-dev  \
  -kcp-kubeconfig-path=/Users/myuser/kcp-kubeconfig.yml \
  -output-path=/tmp/ \
  -dry-run=false \
  -input-file-path=input/runtimeIds.txt \
  -set-controlled-by-kim=true \
  -input-type=txt
```

The above **execution example** will:
1. take the input from the `input/runtimeIds.txt` file (each raw contains single RuntimeID)
1. proceed with fetching Shoot, Cluster Role Bindings and OpenIDConnect resource
1. save output files in the `/tmp/<generated name>` directory. The output directory contains the following:
    - `backup-and-switch-results.json` - the output file with the backup results
    - `backup` - the directory with the backup files
1. label Cluster Role Bindings that were created by the Provisioner
1. switch Runtime to be controlled by KIM

### Output example

```
2025/01/10 09:27:49 INFO Starting runtime backup and switch
2025/01/10 09:27:49 gardener-kubeconfig-path: /Users/myuser/Downloads/kubeconfig-garden-kyma-stage.yaml
2025/01/10 09:27:49 kcp-kubeconfig-path: /Users/myuser/dev/config/sap
2025/01/10 09:27:49 gardener-project-name: kyma-stage
2025/01/10 09:27:49 output-path: /Users/myuser/backup/results 
2025/01/10 09:27:49 dry-run: false
2025/01/10 09:27:49 input-type: txt
2025/01/10 09:27:49 input-file-path: /Users/myuser/dev/runtime-ids.txt
2025/01/10 09:27:49 set-controlled-by-kim: true
2025/01/10 09:27:49
2025/01/10 09:27:49
2025/01/10 09:27:49 INFO Reading runtimeIds from input file
2025/01/10 09:27:54 INFO Runtime backup created successfully runtimeID=a774bae2-ed8b-464e-85cc-ab8acd4461d5
2025/01/10 09:27:54 ERROR Failed to fetch shoot: shoot was deleted or the runtimeID is incorrect runtimeID=exxe4b14-7bc2-4947-931c-f8673793b02f
2025/01/10 09:27:54 INFO Backup completed. Successfully stored backups: 1, Failed backups: 1
2025/01/10 09:27:54 INFO Backup results saved in: backup/results/backup-2025-01-10T09:27:49+01:00/backup-and-switch-results.json
```

The backup results are saved in the `backup/results/backup-2025-01-10T09:27:49+01:00/backup-and-switch-results.json` file.

The `backup-and-switch-results.json` file contains the following content:
```json
[
  {
    "runtimeId": "a774bae2-ed8b-464e-85cc-ab8acd4461d5",
    "shootName": "c-35a9898",
    "status": "Success",
    "backupDirPath": "backup/results/backup-2025-01-10T09:27:49+01:00/backup/a774bae2-ed8b-464e-85cc-ab8acd4461d5",
    "deprecatedCRBs": [
      "admin-cw4mz"
    ],
    "setControlledByKIM": true
  },
  {
    "runtimeId": "exxe4b14-7bc2-4947-931c-f8673793b02f",
    "shootName": "",
    "status": "Error",
    "errorMessage": "Failed to fetch shoot: shoot was deleted or the runtimeID is incorrect",
    "setControlledByKIM": false
  }
]

```
In the above example the runtime with the `exxe4b14-7bc2-4947-931c-f8673793b02f` identifier was not found ; it may be incorrect, or the corresponding shoot was deleted for some reason. 

The runtime with the `a774bae2-ed8b-464e-85cc-ab8acd4461d5` was successfully processed and the backup was stored in the `backup/results/backup-2025-01-10T09:27:49+01:00/backup/a774bae2-ed8b-464e-85cc-ab8acd4461d5` folder. The `admin-cw4mz` Cluster Role Binding was marked as deprecated, and will be cleaned up at some point.

The `backup/results/backup-2025-01-10T09:27:49+01:00/backup/a774bae2-ed8b-464e-85cc-ab8acd4461d5` directory contains the following:
- `c-35a9898-original.yaml` file
- `c-35a9898-to-restore.yaml` file
- `crb` folder
- `oidc` folder

The `c-35a9898-original.yaml` file contains the shoot fetched from the Gardener. The `c-35a9898-to-restore.yaml` file contains the shoot that will be used by restore operation for patching. 
The `crb` directory contains the yaml files with Cluster Role Bindings that refer to `cluster-admin` role. The `oidc` folder contains yaml files with OpenIDConnect resources.

## Configurable Parameters

This table lists the configurable parameters, their descriptions, and default values:

| Parameter | Description                                                                                                                                                                                                                                                                         | Default value                  |
|------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|--------------------------------|
| **kcp-kubeconfig-path** | Path to the Kubeconfig file of KCP cluster.                                                                                                                                                                                                                                         | `/path/to/kcp/kubeconfig`      |
| **gardener-kubeconfig-path** | Path to the Kubeconfig file of Gardener cluster.                                                                                                                                                                                                                                    | `/path/to/gardener/kubeconfig` |
| **gardener-project-name** | Name of the Gardener project.                                                                                                                                                                                                                                                       | `gardener-project-name`        |
| **output-path** | Path where generated report, and yamls will be saved. Directory has to exist.                                                                                                                                                                                                       | `/tmp/`                        |
| **dry-run** | Dry-run flag. Has to be set to **false**, otherwise migrator will not apply the CRs on the KCP cluster.                                                                                                                                                                             | `true`                         |
| **input-type** | Type of input to be used. Possible values: **txt** (will expect text file with one runtime identifier per line, [see the example](input/runtimeids_sample.txt)), and **json** (will expect `json` array with runtime identifiers, [see the example](input/runtimeids_sample.json)). | `json`                         |
| **input-file-path** | Path to the file containing Runtimes to be migrated.                                                                                                                                                                                                                                | `/path/to/input/file`          |
| **set-controlled-by-kim** | Flag determining whether Runtime CR should be modified to be controlled by KIM                                                                                                                                                                                                      | `false`                        |

