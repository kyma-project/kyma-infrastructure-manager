# Runtime Migrator
The `runtime-migrator` application
1. connects to a Gardener project
2. retrieves all existing shoot specifications
3. migrates the shoot specs to the new Runtime custom resource (Runtime CRs created with this migrator have the `operator.kyma-project.io/created-by-migrator=true` label)
4. saves the new Runtime custom resources to files
5. checks if the new Runtime custom resource will not cause update on the Gardener
6. saves the results of the comparison between the original shoot and the shoot KIM produces based on the new Runtime CR
7. applies the new Runtime CRs to the designated KCP cluster
8. saves the migration results in the output json file

## Build

In order to build the app, run the following command:

```bash
go build -o ./bin/runtime-migrator ./cmd
``` 

## Usage

```bash
cat ./runtime-migrator \
  -gardener-kubeconfig-path=/Users/myuser/gardener-kubeconfig.yml \
  -gardener-project-name=kyma-dev  \
  -kcp-kubeconfig-path=/Users/myuser/kcp-kubeconfig.yml \
  -output-path=/tmp/ \
  -dry-run=true \
  -input-file-path=input/runtimeIds.json \
  -input-type=json
```

The above **execution example** will: 
1. take the input from `input/runtimeIds.json` file (json with runtimeIds array)
1. proceed only with Runtime CRs creation for clusters listed in the input 
1. save output files in the `/tmp/` directory. The output directory contains the following:
    - `migration.json` - the output file with the migration results
    - `runtimes` - the directory with the Runtime CRs files
    - `comparison-results` - the directory with the files generated during the comparison process
1. They will not be applied on the KCP cluster (`dry-run` mode)

The input can be also provided in a form of test file:
```bash
cat ./runtime-migrator \
  -gardener-kubeconfig-path=/Users/myuser/gardener-kubeconfig.yml \
  -gardener-project-name=kyma-dev  \
  -kcp-kubeconfig-path=/Users/myuser/kcp-kubeconfig.yml \
  -output-path=/tmp/ \
  -dry-run=true \
  -input-file-path=input/runtimeIds.txt \
  -input-type=txt
```

### Output example

```
2024/11/20 19:49:41 INFO Starting runtime-migrator
2024/11/20 19:49:41 gardener-kubeconfig-path: /Users/i326211/Downloads/kubeconfig-garden-kyma-stage.yaml
2024/11/20 19:49:41 kcp-kubeconfig-path: /Users/i326211/dev/config/sap
2024/11/20 19:49:41 gardener-project-name: kyma-stage
2024/11/20 19:49:41 output-path: /tmp/
2024/11/20 19:49:41 dry-run: true
2024/11/20 19:49:41 input-type: txt
2024/11/20 19:49:41
2024/11/20 19:49:42 INFO Migrating runtimes
2024/11/20 19:49:42 INFO Reading runtimeIds from input file
2024/11/20 19:49:45 INFO Runtime processed successfully runtimeID=1df09b5b-0347-459d-aa0a-715db8fcaad7
2024/11/20 19:49:46 INFO Runtime processed successfully runtimeID=ea439a5e-aa59-4e3e-8bfb-9bab1b31371e
2024/11/20 19:49:50 INFO Runtime processed successfully runtimeID=d6eeafee-ffd5-4f23-97dc-a1df197b3b30
2024/11/20 19:49:52 INFO Runtime CR can cause unwanted update in Gardener. Please verify the runtime CR. runtimeID=99a38a99-e8d7-4b98-a6f2-5a54ed389c4d
2024/11/20 19:49:52 INFO Migration completed. Successfully migrated runtimes: 3, Failed migrations: 0, Differences detected: 1
2024/11/20 19:49:52 INFO Migration results saved in: /tmp/migration-2024-11-20T19:49:42+01:00/migration-results.json
```

The above example shows that the migration process detected a potential problem with Runtime CR. In such a case, Runtime CR that may cause unwanted updates on Gardener will not be applied to the cluster and will require manual intervention.
The migration results are saved in the `/tmp/migration-2024-10-28T13:38:49+01:00/migration-results.json` file.

The `migration-results.json` file contains the following content:
```json
[
   {
      "runtimeId": "1df09b5b-0347-459d-aa0a-715db8fcaad7",
      "shootName": "c-1228ddd",
      "status": "Success",
      "runtimeCRFilePath": "/tmp/migration-2024-11-20T19:49:42+01:00/runtimes/1df09b5b-0347-459d-aa0a-715db8fcaad7.yaml"
   },
   {
      "runtimeId": "ea439a5e-aa59-4e3e-8bfb-9bab1b31371e",
      "shootName": "c3a59d5",
      "status": "Success",
      "runtimeCRFilePath": "/tmp/migration-2024-11-20T19:49:42+01:00/runtimes/ea439a5e-aa59-4e3e-8bfb-9bab1b31371e.yaml"
   },
   {
      "runtimeId": "d6eeafee-ffd5-4f23-97dc-a1df197b3b30",
      "shootName": "c141da7",
      "status": "Success",
      "runtimeCRFilePath": "/tmp/migration-2024-11-20T19:49:42+01:00/runtimes/d6eeafee-ffd5-4f23-97dc-a1df197b3b30.yaml"
   },
   {
      "runtimeId": "99a38a99-e8d7-4b98-a6f2-5a54ed389c4d",
      "shootName": "c-71da0f2",
      "status": "ValidationDetectedUnwantedUpdate",
      "errorMessage": "Runtime may cause unwanted update in Gardener. Please verify the runtime CR.",
      "runtimeCRFilePath": "/tmp/migration-2024-11-20T19:49:42+01:00/runtimes/99a38a99-e8d7-4b98-a6f2-5a54ed389c4d.yaml",
      "comparisonResultDirPath": "/tmp/migration-2024-11-20T19:49:42+01:00/comparison-results/99a38a99-e8d7-4b98-a6f2-5a54ed389c4d"
   }
]
```
The runtime custom resource is saved in the `/tmp/migration-2024-11-20T19:49:42+01:00/comparison-results/99a38a99-e8d7-4b98-a6f2-5a54ed389c4d` file. 

The `comparison-results` directory contains the following files:
- `c-71da0f2.diff`
- `converted-shoot.yaml`
- `original-shoot.yaml` 

The `c-71da0f2.diff` file contains the differences between the original shoot and the shoot that will be created based on the new Runtime CR. The `converted-shoot.yaml` file contains the shoot that will be created based on the new Runtime CR. The `original-shoot.yaml` file contains the shoot fetched from the Gardener.

## Configurable Parameters

This table lists the configurable parameters, their descriptions, and default values:

| Parameter | Description                                                                                                                                                                                                                                                                         | Default value                  |
|-----------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|--------------------------------|
| **kcp-kubeconfig-path** | Path to the Kubeconfig file of KCP cluster.                                                                                                                                                                                                                                         | `/path/to/kcp/kubeconfig`      |
| **gardener-kubeconfig-path** | Path to the Kubeconfig file of Gardener cluster.                                                                                                                                                                                                                                    | `/path/to/gardener/kubeconfig` |
| **gardener-project-name** | Name of the Gardener project.                                                                                                                                                                                                                                                       | `gardener-project-name`        |
| **output-path** | Path where generated report, and yamls will be saved. Directory has to exist.                                                                                                                                                                                                       | `/tmp/`                        |
| **dry-run** | Dry-run flag. Has to be set to **false**, otherwise migrator will not apply the CRs on the KCP cluster.                                                                                                                                                                             | `true`                         |
| **input-type** | Type of input to be used. Possible values: **txt** (will expect text file with one runtime identifier per line, [see the example](input/runtimeids_sample.txt)), and **json** (will expect `json` array with runtime identifiers, [see the example](input/runtimeids_sample.json)). | `json`                         |
| **input-file-path** | Path to the file containing Runtimes to be migrated.                                                                                                                                                                                                                                | `json`                         |

