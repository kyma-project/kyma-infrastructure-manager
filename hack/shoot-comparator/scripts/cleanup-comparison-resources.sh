#!/bin/bash

echo "Cleaning up the resources created for shoot comparison"
printf "\n"

echo "Removing resources needed for fetching results"
kubectl delete -n kcp-system po fetch-test-comparison-results --ignore-not-found
kubectl delete -n kcp-system pvc shoot-comparator-pvc-read-only --ignore-not-found
kubectl delete -n kcp-system volumesnapshot shoot-comparator-pvc --ignore-not-found

printf "\n"

echo "Removing resources needed for performing comparison"
kubectl delete -n kcp-system job/compare-shoots --ignore-not-found
kubectl delete -n kcp-system pvc/shoot-comparator-pvc --ignore-not-found

kubectl delete -n kcp-system pvc test-prov-shoot-read-only test-kim-shoot-read-only --ignore-not-found
kubectl delete -n kcp-system volumesnapshot test-kim-shoot-spec-storage test-prov-shoot-spec-storage --ignore-not-found

printf "\n"
