# Runtime Restorer

The `runtime-restore` application
1. connects to a Gardener project
2. retrieves all existing shoot specifications
3. for each runtime on input list:
   a) gets shoot, Cluster Role Bindings and OpenIDConnect resources from the backup
   b) applies shoot if the current one is no more than one generation newer
   c) applies Cluster Role Bindings from backup provided the objects doesn't exist on the runtime
   d) applies OpenIDConnect from backup provided the objects doesn't exist on the runtime

## Build

In order to build the app, run the following command:

```bash
go build -o ./bin/runtime-restore ./cmd/restore
``` 

## Usage

### Dry run
```bash
cat ./bin/runtime-restore \
  -gardener-kubeconfig-path=/Users/myuser/gardener-kubeconfig.yml \
  -gardener-project-name=kyma-dev  \
  -kcp-kubeconfig-path=/Users/myuser/kcp-kubeconfig.yml \
  -output-path=/tmp/ \
  -dry-run=true \
  -input-file-path=input/runtimeIds.txt \
  -input-type=txt \
  -backup-path=/Users/myuser/backup/results/backup-2025-01-10T09:27:49+01:00
```

The above **execution example** will:
1. take the input from the `input/runtimeIds.txt` file (each raw contains single `RuntimeID`)
1. proceed only with fetching `Shoot`, `Cluster Role Bindings` and `OpenIDConnect` resources from the backup directory
1. save output files in the `/tmp/<generated name>` directory. The output directory contains the following:
   - `restore-results.json` - the output file with the backup results


### Restore runtime
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

The above **execution example** will:
1. take the input from the `input/runtimeIds.txt` file (each row contains single `RuntimeID`)
1. proceed only with fetching `Shoot`, `Cluster Role Bindings` and `OpenIDConnect` resources from the backup directory
1. save output files in the `/tmp/<generated name>` directory. The output directory contains the following:
   - `restore-results.json` - the output file with the backup results

### Backup and switch Runtime to be controlled by KIM

```bash
./bin/runtime-restore \
  -gardener-kubeconfig-path=/Users/myuser/gardener-kubeconfig.yml \
  -gardener-project-name=kyma-dev  \
  -kcp-kubeconfig-path=/Users/myuser/kcp-kubeconfig.yml \
  -output-path=/tmp/ \
  -dry-run=false \
  -input-file-path=input/runtimeIds.txt \
  -input-type=txt \
  -backup-path=/Users/myuser/backup/results/backup-2025-01-10T09:27:49+01:00
```

The above **execution example** will:
1. take the input from the `input/runtimeIds.txt` file (each raw contains single `RuntimeID`)
1. proceed with fetching `Shoot`, `Cluster Role Bindings` and `OpenIDConnect` resources from the backup directory
1. save output files in the `/tmp/<generated name>` directory. The output directory contains the following:
   - `restore-results.json` - the output file with the backup results
1. patch shoot with file from backup
1. create Cluster Role Bindings that doesn't exist on the runtime
1. create `OpenIDConnect` resources that doesn't exist on runtime. 

### Output example
```
2025/01/10 14:04:14 INFO Starting runtime-restorer
2025/01/10 14:04:14 gardener-kubeconfig-path: /Users/myuser/kubeconfig-garden-kyma-stage.yaml
2025/01/10 14:04:14 kcp-kubeconfig-path: /Users/myuser/dev/config/sap
2025/01/10 14:04:14 gardener-project-name: kyma-stage
2025/01/10 14:04:14 output-path: /tmp
2025/01/10 14:04:14 dry-run: false
2025/01/10 14:04:14 input-type: txt
2025/01/10 14:04:14 input-file-path: /Users/myuser/input/runtime-ids-oidc.txt
2025/01/10 14:04:14 backup-path: /Users/myuser/backup/results/backup-2025-01-10T09:27:49+01:00 
2025/01/10 14:04:14 restore-crb: true
2025/01/10 14:04:14 restore-oidc: true
2025/01/10 14:04:14
2025/01/10 14:04:14 INFO Reading runtimeIds from input file
2025/01/10 14:04:17 INFO Runtime restore performed successfully runtimeID=a774bae2-ed8b-464e-85cc-ab8acd4461d5
2025/01/10 14:04:17 ERROR Failed to fetch shoot: shoot was deleted or the runtimeID is incorrect runtimeID=exxe4b14-7bc2-4947-931c-f8673793b02f
2025/01/10 14:04:17 INFO Restore completed. Successfully restored backups: 1, Failed operations: 1
2025/01/10 14:04:17 INFO Restore results saved in: /tmp/restore-2025-01-10T14:04:14+01:00/restore-results.json
```

The restore results are saved in the `/tmp/restore-2025-01-10T14:04:14+01:00/restore-results.json` file.

The `restore-results.json` file contains the following content:
```
[
  {
    "runtimeId": "a774bae2-ed8b-464e-85cc-ab8acd4461d5",
    "shootName": "c-35a9898",
    "status": "Success",
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