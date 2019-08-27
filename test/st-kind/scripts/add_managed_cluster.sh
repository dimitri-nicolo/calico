#!/usr/bin/env bash

# Setup kubectl
export KUBECONFIG="$(kind get kubeconfig-path --name="kind")"

kubectl apply -o=jsonpath='{.metadata.uid}' -f - <<EOF
apiVersion: projectcalico.org/v3
kind: ManagedCluster
metadata:
  name: managed-cluster
spec:
  installManifest:
EOF
