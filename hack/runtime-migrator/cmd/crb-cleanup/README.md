# CRB Cleanup script

This script is used to clean up old ClusterRoleBindings (CRBs) after migration.
It looks for old and new CRBs, compares them,
and if all old ones have a new equivalent - it removes the old ones.

The cleanup script is run locally, with kubeconfig pointing to the cluster.

## Configuration

| flag          | description                                         | default                                                                                            |
| ------------- | --------------------------------------------------- | -------------------------------------------------------------------------------------------------- |
| `-kubeconfig` | Path to the kubeconfig file                         | in-cluster config                                                                                  |
| `-oldLabel`   | Label selector for old CRBs                         | `kyma-project.io/deprecation=to-be-removed-soon,reconciler.kyma-project.io/managed-by=provisioner` |
| `-newLabel`   | Label selector for new CRBs                         | `reconciler.kyma-project.io/managed-by=infrastructure-manager`                                     |
| `-output`     | Output dir for created logs.                        | _empty_ (acts like `./ `)                                                                          |
| `-dry-run`    | Don't perform any destructive actions               | `true`                                                                                             |
| `-verbose`    | Print detailed logs                                 | `false`                                                                                            |
| `-force`      | Delete old CRBs even if they have no new equivalent | `false`                                                                                            |

> [!NOTE]
> if `-output` doesn't end with `/`, the name of the files will be prefixed with last segment.
> eg. `-output=./dev/log/cluster_a-` will create files like `./dev/log/cluster_a-missing.json`, `./dev/log/cluster_a-removed.json`, etc.

> [!WARNING]
> without `-dry-run=false` the script won't delete anything, even with a `-force` flag

## Usage

To run cleanup script, execute:

```bash
go run ./cmd/crb-cleanup -output=./dev/logs/my-cluster/ -kubeconfig=./dev/kubeconfig -dry-run=false
```

If there are missing CRBs, the script will print a list of them and exit with a zero status code.
Missing CRBs can be inspected as JSON in `./dev/logs/my-cluster/missing.json`. No CRBs will be removed.

After inspecting the missing CRBs, you can re-run the script with the `-force` flag to delete them.

If no CRBs are missing, the script will remove old CRBs.
Removed CRBs can be inspected as JSON in `./dev/logs/my-cluster/removed.json`.

If any errors occured during deletion (eg. permission error), the CRBs that failed will be listed in `./dev/logs/my-cluster/failures.json`.

All of the log files will be created either way.

### `-dry-run` mode

When running the script without `-dry-run=false` flag, CRBs that _would_ be removed will be listed as JSON in `./dev/logs/my-cluster/removed.json`.
No destructive actions will be performed.
