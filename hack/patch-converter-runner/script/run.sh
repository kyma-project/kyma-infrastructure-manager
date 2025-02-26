#!/usr/bin/env bash

if [ -z "$1" ] || [ -z "$2" ] || [ -z "$3" ]; then
  echo "Usage: $0 <output_dir> $1 <runtime> $2 <shoot>"
  exit 1
fi

if [ -z "$KCP_KUBECONFIG" ]; then
  echo "Error: KCP_KUBECONFIG environment variable is not set."
  exit 1
fi

if [ -z "$GARDENER_KUBECONFIG" ]; then
  echo "Error: GARDENER_KUBECONFIG environment variable is not set."
  exit 1
fi

if [ -z "$GARDENER_NAMESPACE" ]; then
  echo "Error: GARDENER_NAMESPACE environment variable is not set."
  exit 1
fi

OUTPUTDIR=$1
RUNTIME=$2
SHOOT=$3

export KUBECONFIG=$KCP_KUBECONFIG
kubectl get runtime/$RUNTIME -n kcp-system -oyaml > $OUTPUTDIR/runtime.yaml
if [ $? -ne 0 ]; then
  echo "Error: Failed to get runtime $RUNTIME."
  exit 1
fi

export KUBECONFIG=$GARDENER_KUBECONFIG
kubectl get shoot/$SHOOT -n $GARDENER_NAMESPACE -oyaml > $OUTPUTDIR/current-shoot.yaml
if [ $? -ne 0 ]; then
  echo "Error: Failed to get shoot $SHOOT."
  exit 1
fi

../../bin/patch-converter-runner --output-path "$OUTPUTDIR/generated-shoot.yaml" --runtime-path  "$OUTPUTDIR/runtime.yaml" --shoot-path  "$OUTPUTDIR/current-shoot.yaml" --kcp-kubeconfig-path $KCP_KUBECONFIG
