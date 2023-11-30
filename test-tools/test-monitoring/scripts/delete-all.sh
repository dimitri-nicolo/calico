#!/bin/bash

# This script deletes all resources from the test operator.
kubectl delete -f ../examples/prometheus/ &
kubectl delete -f ../examples/grafana/ &

wait

# Delete the clusterrole binding we created.
kubectl delete clusterrolebinding default-admin

# Clean up some stuff that gets left over.
kubectl delete statefulsets -n tigera-fluentd prometheus-client prometheus-k8s prometheus-self
kubectl delete statefulsets -n kube-system prometheus-calico-node
kubectl delete statefulsets -n kube-system prometheus-calico-typha
kubectl delete secret -n tigera-fluentd prometheus-client prometheus-k8s prometheus-self
kubectl delete secret -n kube-system prometheus-calico-node prometheus-calico-typha
