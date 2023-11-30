#!/bin/bash

for svc in prometheus-calico-node prometheus-calico-typha; do
  kubectl patch svc $svc \
    --namespace calico-system \
    -p '{"spec": {"type": "NodePort"}}'
done

for svc in prometheus-client prometheus-k8s; do
  kubectl patch svc $svc \
    --namespace tigera-fluentd \
    -p '{"spec": {"type": "NodePort"}}'
done

kubectl patch svc grafana \
  --namespace kube-system \
  -p '{"spec": {"type": "NodePort"}}'
