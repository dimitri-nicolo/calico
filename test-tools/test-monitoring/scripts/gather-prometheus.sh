#!/bin/bash

# This script gathers prometheus data and stores it in .tgz files in
# $(pwd)/prometheus_databases/<prometheus_name>.tgz.

# Make directory.
mkdir -p prometheus_databases

# Get data for prometheus-client-0
kubectl exec prometheus-client-0 -c prometheus -- sh -c "tar -cvzf - /var/prometheus/data > /prometheus/prometheus_data.tgz"
kubectl cp -c prometheus prometheus-client-0:/prometheus/prometheus_data.tgz ./prometheus_databases/prometheus-client-0.tgz

# Get data for prometheus-calico-node-0
kubectl -n kube-system exec prometheus-calico-node-0 -c prometheus -- sh -c "tar -cvzf - /var/prometheus/data > /prometheus/prometheus_data.tgz"
kubectl -n kube-system cp -c prometheus prometheus-calico-node-0:/prometheus/prometheus_data.tgz ./prometheus_databases/prometheus-calico-node-0.tgz

# Get data for prometheus-self-0
kubectl exec prometheus-self-0 -c prometheus -- sh -c "tar -cvzf - /var/prometheus/data > /prometheus/prometheus_data.tgz"
kubectl cp -c prometheus prometheus-self-0:/prometheus/prometheus_data.tgz ./prometheus_databases/prometheus-self-0.tgz

# Get data for prometheus-k8s-0
kubectl exec prometheus-k8s-0 -c prometheus -- sh -c "tar -cvzf - /var/prometheus/data > /prometheus/prometheus_data.tgz"
kubectl cp -c prometheus prometheus-k8s-0:/prometheus/prometheus_data.tgz ./prometheus_databases/prometheus-k8s-0.tgz


echo ""
echo "Prometheus data saved to $(pwd)/prometheus_databases/prometheus-*.tgz"
