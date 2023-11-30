#!/bin/bash

kubectl delete pod -n tigera-fluentd prometheus-client-0 prometheus-k8s-0 # prometheus-self-0
kubectl delete pod prometheus-calico-node-0 -n calico-system
kubectl delete pod prometheus-calico-typha-0 -n calico-system
