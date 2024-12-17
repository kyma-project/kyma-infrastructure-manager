# Shoot comparator scripts

## Overview
The scripts are designed to make shoot comparison easier. 

## Running comparison scripts

In order to run the comparison scripts, execute the following command:
```bash
./run-comparison.sh
```

The script will perform the following steps:
- Invoke ./cleanup-comparison-resources.sh to clean up the resources created during the previous comparison
- Prepare volume snapshots and PVCs containing files stored by Provisioner and KIM
- Run the comparison job that mounts the PVCs and compares the files. Results will be stored in the separate volume.
- Prepare volume snapshot and PVC containing the comparison results
- Run pod that mounts the PVC
- Copy the comparison results to the local directory

The following is an example of the script output:
```
Cleaning up the resources created for shoot comparison

Removing resources needed for fetching results
pod "fetch-test-comparison-results" deleted
persistentvolumeclaim "shoot-comparator-pvc-read-only" deleted
volumesnapshot.snapshot.storage.k8s.io "shoot-comparator-pvc" deleted

Removing resources needed for performing comparison
job.batch "compare-shoots" deleted
persistentvolumeclaim "shoot-comparator-pvc" deleted
persistentvolumeclaim "test-prov-shoot-read-only" deleted
persistentvolumeclaim "test-kim-shoot-read-only" deleted
volumesnapshot.snapshot.storage.k8s.io "test-kim-shoot-spec-storage" deleted
volumesnapshot.snapshot.storage.k8s.io "test-prov-shoot-spec-storage" deleted

Preparing data for comparison
volumesnapshot.snapshot.storage.k8s.io/test-prov-shoot-spec-storage created
volumesnapshot.snapshot.storage.k8s.io/test-kim-shoot-spec-storage created
persistentvolumeclaim/test-prov-shoot-read-only created
persistentvolumeclaim/test-kim-shoot-read-only created

Running comparison job
persistentvolumeclaim/shoot-comparator-pvc created
job.batch/compare-shoots created

Waiting for the job to complete. It may take couple of minutes. Please, be patient!
job.batch/compare-shoots condition met
Job completed

Fetching logs for the job
2024/08/01 11:10:56 INFO Comparing directories: /testdata/provisioner and /testdata/kim
2024/08/01 11:10:56 INFO Saving comparison details
2024/08/01 11:10:56 INFO Results stored in "/results/2024-08-01T11:10:56Z"
2024/08/01 11:10:56 WARN Differences found.

Applying helper resources for fetching results
volumesnapshot.snapshot.storage.k8s.io/shoot-comparator-pvc created
persistentvolumeclaim/shoot-comparator-pvc-read-only created
pod/fetch-test-comparison-results created

Waiting for fetch-test-comparison-results pod to be ready
pod/fetch-test-comparison-results condition met
fetch-test-comparison-results pod is ready
Copying comparison results to /tmp/shoot_compare
```

Please mind that volume operations such as snapshot creation and PVC creation may take some time. The job will be started only after the volumes are ready.

In case timeout occurs when waiting for the job to complete, the script will exit with the following message:
```
Waiting for the job to complete. It may take couple of minutes. Please, be patient!
error: timed out waiting for the condition on jobs/compare-shoots
Job is still not completed. Please check it manually. Exiting...
```
> Note: mind the script creates additional resources in the `kcp-system` namespace. Once you are done with the comparison, you can clean up the resources by executing the following command:
> ```bash
> ./cleanup-comparison-resources.sh
> ```

In such case you can check the job status manually by executing the following command:
```bash
kubectl get job compare-shoots -n kcp-system
```

In case timeout occurs when waiting for the pod to be ready, the script will exit with the following message:
```
Waiting for fetch-test-comparison-results pod to be ready
error: timed out waiting for the condition on pods/fetch-test-comparison-results
fetch-test-comparison-results pod is not ready. Please check it manually. Exiting...`
```

In such case you can check the pod status manually by executing the following commands:
```bash
kubectl get po fetch-test-comparison-results -n kcp-system
kubectl describe po fetch-test-comparison-results -n kcp-system
```

## Comparing files starting from a specific date

If you want to compare files older than a specific date, you can specify the date in the `./manifests/job.yaml` script. 

## Analyzing comparison results

If any differences were detected you can analyze the results by examining the content of the `result.txt` file stored in the output directory. The file will contain the details of the comparison, such as the names of the files that differ.

The following is an example of the `result.txt` file content:
```
Comparing files older than:0001-01-01 00:00:00 +0000 UTC

Number of files in /Users/myuser/dev/temp/kim-test/shoot-comparator/test2/kim directory = 2
Number of files in /Users/myuser/dev/temp/kim-test/shoot-comparator/test2/provisioner directory = 2

Differences found.

------------------------------------------------------------------------------------------
Files that differ: 
- shoot1.yaml
------------------------------------------------------------------------------------------
```

In order to fetch the compared files you must copy the contents of the files. You can do it by executing the following command:
```bash
./fetch_shoots_for_provisioner_and_kim.sh
```
## Cleaning volume with shoots, and mutations for Provisioner

If you need to remove some files from the volume perform the following steps:
1. Check what node Provisioner is running on:
```bash
kubectl get po/<Provisioner pod> -n kcp-system -ojsonpath={.spec.nodeName}
```
2. Modify the `./manifests/mount-prov-volume-with-write-access.yaml` file by providing the node name in the `nodeName` field.
3. Apply the changes by executing the following command:
```bash
kubectl apply -f ./manifests/mount-prov-volume-with-write-access.yaml
```
4. Wait for the job to complete. Once the job is completed, exec into pod with the following command:
```bash
kubectl exec mount-prov-volume-with-write-access -n kcp-system -it -- /bin/sh
```
5. Remove the files from the `/testdata/provisioner` directory.
6. Remove the pod with the following command:
```bash
kubectl delete -f ./manifests/mount-prov-volume-with-write-access.yaml
```

## Cleaning volume with shoots, and runtime CRs for KIM

If you need to remove some files from the volume perform the following steps:
1. Check what node KIM is running on:
```bash
kubectl get po/<KIM pod> -n kcp-system -ojsonpath={.spec.nodeName}
```
2. Modify the `./manifests/mount-kim-volume-with-write-access.yaml` file by providing the node name in the `nodeName` field.
3. Apply the changes by executing the following command:
```bash
kubectl apply -f ./manifests/mount-kim-volume-with-write-access.yaml
```
4. Wait for the job to complete. Once the job is completed, exec into pod with the following command:
```bash
kubectl exec mount-kim-volume-with-write-access -n kcp-system -it -- /bin/sh
```
5. Remove the files from the `/testdata/kim` directory.
6. Remove the pod with the following command:
```bash
kubectl delete -f ./manifests/mount-kim-volume-with-write-access.yaml
```