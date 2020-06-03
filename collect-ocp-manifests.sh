#!/bin/bash
set -e

# helper script for manifests archive, collects all the manifests required to create an OCP cluster
# used in make release-archive
# docs: https://docs.tigera.io/getting-started/openshift/installation/

if [ ! -d "manifests" ]; then
    echo "must have manifests directory to proceed"
    exit 1
fi

echo "creating \"ocp-manifests\" directory..."
mkdir -p ocp-manifests

echo "collecting manifests.."
find ./manifests/ocp/ -name "*.yaml" | xargs -I{} cp -u {} ./ocp-manifests
echo "done"

