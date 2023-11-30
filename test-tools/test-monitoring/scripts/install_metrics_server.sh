#!/bin/bash

kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/download/v0.5.0/components.yaml

kubectl patch deployment metrics-server \
  --namespace kube-system \
  --type='json' \
  -p='[
  {"op": "replace", "path": "/spec/template/spec/containers/0/args", "value": [
    "--cert-dir=/tmp",
    "--secure-port=443",
    "--kubelet-insecure-tls",
    "--kubelet-preferred-address-types=InternalIP,ExternalIP,Hostname",
    "--kubelet-use-node-status-port",
    "--metric-resolution=15s"
  ]}]'

# kubectl patch deployment metrics-server \
#   --namespace kube-system \
#   -p '{"spec": {"template": {"spec": {"nodeSelector": {"cloud.google.com/gke-nodepool": "infrastructure"}}}}}'
