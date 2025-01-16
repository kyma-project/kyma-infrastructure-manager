# Runtime Restore

The `runtime-restore` application has the following tasks:
1. Connect to a Gardener project, and a KCP cluster.
2. Retrieve all existing shoot specifications.
3. For each runtime on the input list:
   1. Gets the Shoot, Cluster Role Bindings and OpenIDConnect resources from the backup.
   2.  To prevent KIM from modifying the shoot and the runtime resources, update the Runtime CR to label the runtime as controlled by Provisioner.
   3. Patch the shoot with the Shoot spec read from the backup.
   4. Apply the ClusterRoleBindings from the backup if the objects don't exist on the runtime.
   5. Apply OpenIDConnect from the backup if the objects don't exist on the runtime.

## Build

To build the `runtime-restore` application, run:

```bash
go build -o ./bin/runtime-restore ./cmd/restore
``` 

## Usage

### Dry Run
```bash
./bin/runtime-restore \
  -gardener-kubeconfig-path=/Users/myuser/gardener-kubeconfig.yml \
  -gardener-project-name=kyma-dev  \
  -kcp-kubeconfig-path=/Users/myuser/kcp-kubeconfig.yml \
  -output-path=/tmp/ \
  -dry-run=true \
  -input-file-path=input/runtimeIds.txt \
  -input-type=txt \
  -backup-path=/Users/myuser/backup/results/backup-2025-01-10T09:27:49+01:00
```

This execution example does the following:
1. Take the input from the `input/runtimeIds.txt` file (each row contains single `RuntimeID`).
1. Proceed only with fetching the `Shoot`, `Cluster Role Binding`, and `OpenIDConnect` resources from the backup directory.
1. Save the output files in the `/tmp/<generated name>` directory. The output directory contains the following:
   - `restore-results.json` - the output file with the restore results

### Backup and Switch Runtime to Be Controlled by KIM

```bash
./bin/runtime-restore \
  -gardener-kubeconfig-path=/Users/myuser/gardener-kubeconfig.yml \
  -gardener-project-name=kyma-stage  \
  -kcp-kubeconfig-path=/Users/myuser/kcp-kubeconfig.yml \
  -output-path=/tmp/ \
  -dry-run=false \
  -input-file-path=input/runtimeIds.txt \
  -input-type=txt \
  -backup-path=/Users/myuser/backup/results/backup-2025-01-10T09:27:49+01:00
```

This execution example does the following:
1. Take the input from the `input/runtimeIds.txt` file (each row contains single `RuntimeID`).
1. Proceed with fetching the `Shoot`, `Cluster Role Binding`, and `OpenIDConnect` resources from the backup directory.
1. Patch shoot with the file from backup.
1. Create ClusterRoleBindings that don't exist on the runtime.
1. Create the `OpenIDConnect` resources that don't exist on runtime. 
1. Save the output files in the `/tmp/<generated name>` directory. The output directory contains the following:
   - `restore-results.json` - the output file with the backup results

### Output example

#### Runtime restored successfully

```
2025/01/13 08:50:23 INFO Starting runtime-restorer
2025/01/13 08:50:23 gardener-kubeconfig-path: /Users/myuser/Downloads/kubeconfig-garden-kyma-stage.yaml
2025/01/13 08:50:23 kcp-kubeconfig-path: /Users/myuser/dev/config/sap
2025/01/13 08:50:23 gardener-project-name: kyma-stage
2025/01/13 08:50:23 output-path: /tmp
2025/01/13 08:50:23 dry-run: false
2025/01/13 08:50:23 input-type: txt
2025/01/13 08:50:23 input-file-path: /Users/myuser/dev/input/runtime-ids-oidc.txt
2025/01/13 08:50:23 backup-path: /Users/myuser/dev/backup/results/backup-2025-01-13T07:49:17+01:00
2025/01/13 08:50:23 restore-crb: true
2025/01/13 08:50:23 restore-oidc: true
2025/01/13 08:50:23
2025/01/13 08:50:23 INFO Reading runtimeIds from input file
2025/01/13 08:50:34 INFO Runtime restore performed successfully runtimeID=a774bae2-ed8b-464e-85cc-ab8acd4461d5
2025/01/13 08:50:34 ERROR Failed to fetch shoot: shoot was deleted or the runtimeID is incorrect runtimeID=exxe4b14-7bc2-4947-931c-f8673793b02f
2025/01/13 08:50:34 INFO Restore completed. Successfully restored backups: 1, Failed operations: 1
2025/01/13 08:50:34 INFO Restore results saved in: /tmp/restore-2025-01-13T08:50:23+01:00/restore-results.json
```

The restore results are saved in the `/tmp/restore-2025-01-10T14:04:14+01:00/restore-results.json` file.

The `restore-results.json` file contains the following content:
```
[
  {
    "runtimeId": "a774bae2-ed8b-464e-85cc-ab8acd4461d5",
    "shootName": "c-35a9898",
    "status": "Success",
    "restoredCRBs": [
      "admin-cw4mz"
    ],
    "restoredOIDCs": [
      "kyma-oidc-0"
    ]
  },
  {
    "runtimeId": "exxe4b14-7bc2-4947-931c-f8673793b02f",
    "shootName": "",
    "status": "Error",
    "errorMessage": "Failed to fetch shoot: shoot was deleted or the runtimeID is incorrect"
  }
]
```

In the above example, the runtime with the `exxe4b14-7bc2-4947-931c-f8673793b02f` identifier was not found. In such a case, verify the following:
- Is the identifier correct?
- Does the corresponding shoot exist, and does it has `kcp.provisioner.kyma-project.io/runtime-id` label set?

The runtime with the `a774bae2-ed8b-464e-85cc-ab8acd4461d5` was successfully restored. The shoot spec was patched, and the following resources recreated:
- `admin-cw4mz` of type ClusterRoleBinding
- `kyma-oidc-0` of type OpenIDConnect 

#### Runtime must be restored manually

```
2025/01/13 09:04:56 INFO Starting runtime-restorer
2025/01/13 09:04:56 gardener-kubeconfig-path: /Users/myuser/Downloads/kubeconfig-garden-kyma-stage.yaml
2025/01/13 09:04:56 kcp-kubeconfig-path: /Users/myuser/dev/config/sap
2025/01/13 09:04:56 gardener-project-name: kyma-stage
2025/01/13 09:04:56 output-path: /tmp
2025/01/13 09:04:56 dry-run: false
2025/01/13 09:04:56 input-type: txt
2025/01/13 09:04:56 input-file-path: /Users/myuser/dev/input/runtime-ids-oidc.txt
2025/01/13 09:04:56 backup-path: /Users/myuser/backup/results/backup-2025-01-10T13:50:55+01:00
2025/01/13 09:04:56 restore-crb: true
2025/01/13 09:04:56 restore-oidc: true
2025/01/13 09:04:56
2025/01/13 09:04:56 INFO Reading runtimeIds from input file
2025/01/13 09:05:01 WARN Verify the current state of the system. Restore should be performed manually, as the backup may overwrite user's changes. runtimeID=a774bae2-ed8b-464e-85cc-ab8acd4461d5
2025/01/13 09:05:01 ERROR Failed to fetch shoot: shoot was deleted or the runtimeID is incorrect runtimeID=exxe4b14-7bc2-4947-931c-f8673793b02f
2025/01/13 09:05:01 INFO Restore completed. Successfully restored backups: 0, Failed operations: 1
2025/01/13 09:05:01 INFO Restore results saved in: /tmp/restore-2025-01-13T09:04:56+01:00/restore-results.json
```

The `restore-results.json` file contains the following content:
```
[
  {
    "runtimeId": "a774bae2-ed8b-464e-85cc-ab8acd4461d5",
    "shootName": "c-35a9898",
    "status": "UpdateDetected"
  },
  {
    "runtimeId": "exxe4b14-7bc2-4947-931c-f8673793b02f",
    "shootName": "",
    "status": "Error",
    "errorMessage": "Failed to fetch shoot: shoot was deleted or the runtimeID is incorrect"
  }
]
```

The runtime with the `a774bae2-ed8b-464e-85cc-ab8acd4461d5` cannot be automatically restored, because there were several updates to the shoot. The runtime must be fixed manually, because there is a risk that some updates performed by the user will be overwritten.

## Configurable Parameters

The following table lists the configurable parameters, their descriptions, and default values:

| Parameter | Description                                                                                                                                                                                                                                                                         | Default Value          |
|------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|------------------------|
| **kcp-kubeconfig-path** | Path to the Kubeconfig file of the KCP cluster.                                                                                                                                                                                                                                         | `/path/to/kcp/kubeconfig` |
| **gardener-kubeconfig-path** | Path to the Kubeconfig file of the Gardener cluster.                                                                                                                                                                                                                                    | `/path/to/gardener/kubeconfig` |
| **gardener-project-name** | Name of the Gardener project.                                                                                                                                                                                                                                                       | `gardener-project-name` |
| **output-path** | Path where the generated report, and the yaml files are saved. This directory must exist.                                                                                                                                                                                                       | `/tmp/`                |
| **dry-run** | Dry-run flag. Must be set to **false**, otherwise the migrator does not apply the CRs on the KCP cluster.                                                                                                                                                                             | `true`                 |
| **input-type** | Type of input to be used. Possible values: **txt** (expects a text file with one runtime identifier per line, [see the example](input/runtimeids_sample.txt)), and **json** (expects a `json` array with runtime identifiers, [see the example](input/runtimeids_sample.json)). | `json`                 |
| **input-file-path** | Path to the file containing the runtimes to be migrated.                                                                                                                                                                                                                                | `/path/to/input/file`  |
| **backup-path** | Path to the directory containing backup files.                                                                                                                                                                                                                                       | `/path/to/input/file`  |
| **restore-crb** | Flag determining whether ClusterRoleBindings are restored.                                                                                                                                                                                                                   | `/path/to/backup/dir`                       |
| **restore-oidc** | Flag determining whether OPenIDConnect resources are restored.                                                                                                                                                                                                                 | `/path/to/backup/dir`                       |
