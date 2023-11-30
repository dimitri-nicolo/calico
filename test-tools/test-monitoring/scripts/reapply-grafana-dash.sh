#!/bin/bash

# This script deletes and re-adds the grafana dashboard import job.

kubectl delete jobs -n kube-system grafana-import
kubectl apply -f ../examples/grafana/grafana.import.yaml

