#!/bin/bash

./cleanup-comparison-resources.sh

echo "Preparing data for comparison"
kubectl apply -f ./manifests/snapshot-for-comparison.yaml

printf "\n"

printf "Running comparison job \n"
kubectl apply -f ./manifests/job.yaml

printf "\n"

# wait for completion
echo "Waiting for the job to complete. It may take couple of minutes. Please, be patient!"
kubectl wait --for=condition=complete job/compare-shoots -n kcp-system --timeout=10m

result=$?
if (( $result == 0 ))
then
  echo "Job completed"
else
  if kubectl wait --for=condition=failed --timeout=0 job/compare-shoots -n kcp-system 2>/dev/null; then
      echo "Job failed to complete. Exiting..."
      exit 1
  fi

  echo "Job is still not completed. Please check it manually. Exiting..."
  exit 2
fi

printf "\n"

echo "Fetching logs for the job"
kubectl logs job/compare-shoots -n kcp-system

printf "\n"

echo "Applying helper resources for fetching results"
kubectl apply -f ./manifests/fetch-results-pod.yaml

printf "\n"

echo "Waiting for fetch-test-comparison-results pod to be ready"
kubectl wait --for=condition=ready pod/fetch-test-comparison-results -n kcp-system --timeout=5m

result=$?
if (( $result == 0 ))
then
  echo "fetch-test-comparison-results pod is ready"
else
  echo "fetch-test-comparison-results pod is not ready. Please check it manually. Exiting..."
  exit 3
fi

echo "Copying comparison results to /tmp/shoot_compare"
kubectl cp kcp-system/fetch-test-comparison-results:results/ /tmp/shoot_compare