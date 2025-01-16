# Runtime Backup and Switch

The `runtime-backup-and-switch` application has the following tasks:
1. Connect to a Gardener project and KCP cluster.
2. Retrieve all existing shoot specifications.
3. For each runtime on the input list that was created by the provisioner (shoot is labelled with `kcp.provisioner.kyma-project.io/runtime-id`):
   1. Get the `Shoot`, `ClusterRoleBinding`, and `OpenIDConnect` resources.
   2. Save the backup on a disk.
   3. Mark the ClusterRoleBindings that were created by the Provisioner with the `kyma-project.io/deprecation` label.
   4. To make sure KIM controls the runtime, set the `kyma-project.io/controlled-by-provisioner` label to `false`.

## Build

To build the `runtime-backup-and-switch` application, run:

```bash
go build -o ./bin/runtime-backup-and-switch ./cmd/backup-and-switch
``` 

## Usage

### Dry Run
This execution does the following:
1. Take the input from the `input/runtimeIds.txt` file (each raw contains a single `RuntimeID`).
1. proceed only with fetching Shoot, Cluster Role Bindings and OpenIDConnect resources
1. Save the output files in the `/tmp/<generated name>` directory. The output directory contains the following:
   - `backup-and-switch-results.json` - the output file with the backup results
   - `backup` - the directory with the backup files

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


   
### Backup and Switch Runtime to Be Controlled by KIM

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

This execution example does the following:
1. Take the input from the `input/runtimeIds.txt` file (each raw contains single RuntimeID).
1. Proceed with fetching the `Shoot`, `ClusterRoleBinding`, and `OpenIDConnect` resources.
1. Save the output files in the `/tmp/<generated name>` directory. The output directory contains the following:
    - `backup-and-switch-results.json` - the output file with the backup results
    - `backup` - the directory with the backup files
1. Label the ClusterRoleBindings that were created by the Provisioner.
1. Switch the runtime to be controlled by KIM.

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
In the above example, the runtime with the `exxe4b14-7bc2-4947-931c-f8673793b02f` identifier was not found. In such a case, verify the following:
- Is the identifier correct?
- Does the corresponding shoot exist, and does it have the `kcp.provisioner.kyma-project.io/runtime-id` label set?

The runtime with the `a774bae2-ed8b-464e-85cc-ab8acd4461d5` was successfully processed and the backup was stored in the `backup/results/backup-2025-01-10T09:27:49+01:00/backup/a774bae2-ed8b-464e-85cc-ab8acd4461d5` folder. The `admin-cw4mz` ClusterRoleBinding was marked as deprecated, and will be cleaned up at some point.

The `backup/results/backup-2025-01-10T09:27:49+01:00/backup/a774bae2-ed8b-464e-85cc-ab8acd4461d5` directory contains the following:
- `c-35a9898-original.yaml` file
- `c-35a9898-to-restore.yaml` file
- `crb` folder
- `oidc` folder

The `c-35a9898-original.yaml` file contains the shoot fetched from Gardener. The `c-35a9898-to-restore.yaml` file contains the shoot that will be used by the restore operation for patching. 
The `crb` directory contains the yaml files with ClusterRoleBindings that refer to the `cluster-admin` role. The `oidc` folder contains yaml files with OpenIDConnect resources.

## Configurable Parameters

The following table lists the configurable parameters, their descriptions, and default values:

| Parameter | Description                                                                                                                                                                                                                                                                         | Default value                  |
|------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|--------------------------------|
| **kcp-kubeconfig-path** | Path to the Kubeconfig file of the KCP cluster.                                                                                                                                                                                                                                         | `/path/to/kcp/kubeconfig`      |
| **gardener-kubeconfig-path** | Path to the Kubeconfig file of the Gardener cluster.                                                                                                                                                                                                                                    | `/path/to/gardener/kubeconfig` |
| **gardener-project-name** | Name of the Gardener project.                                                                                                                                                                                                                                                       | `gardener-project-name`        |
| **output-path** | Path where the generated report, and the yaml files are saved. This directory must exist.                                                                                                                                                                                                       | `/tmp/`                        |
| **dry-run** | Dry-run flag. Must be set to **false**, otherwise the migrator does not apply the CRs on the KCP cluster.                                                                                                                                                                             | `true`                         |
| **input-type** | Type of input to be used. Possible values: **txt** (expects a text file with one runtime identifier per line, [see the example](input/runtimeids_sample.txt)), and a **json** (will expect `json` array with runtime identifiers, [see the example](input/runtimeids_sample.json)). | `json`                         |
| **input-file-path** | Path to the file containing the runtimes to be migrated.                                                                                                                                                                                                                                | `/path/to/input/file`          |
| **set-controlled-by-kim** | Flag determining whether the runtime CR is modified to be controlled by KIM.                                                                                                                                                                                                      | `false`                        |

