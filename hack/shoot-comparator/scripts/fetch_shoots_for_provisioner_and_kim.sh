#!/bin/bash
echo "Starting pod that mounts KIM, and Provisioner volumes"
kubectl apply -f ./manifests/fetch-kim-and-provisioner-files.yaml
printf "\n"

echo "Waiting for fetch-test-shoot-specs pod to be ready"
kubectl wait --for=condition=ready pod/fetch-test-shoot-specs -n kcp-system --timeout=5m

result=$?
if (( $result == 0 ))
then
  echo "fetch-test-shoot-specs pod is ready"
else
  echo "fetch-test-shoot-specs pod is not ready. Please check it manually. Exiting..."
  exit 1
fi

printf "\n"

echo "Copying KIM specs to /tmp/shoot_specs/kim"
kubectl cp kcp-system/fetch-test-shoot-specs:testdata/kim/ /tmp/shoot_specs/kim

echo "Copying Provisioner specs to /tmp/shoot_specs/provisioner"
kubectl cp kcp-system/fetch-test-shoot-specs:testdata/provisioner/ /tmp/shoot_specs/provisioner

printf "\n"

echo "Cleaning up the pod created for fetching shoots for provisioner and kim"
kubectl delete po fetch-test-shoot-specs -n kcp-system --ignore-not-found