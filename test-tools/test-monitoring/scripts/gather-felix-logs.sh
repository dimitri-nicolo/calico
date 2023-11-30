#!/bin/bash

# This script uses kubectl to gather felix logs from each running calico/node Pod.

calico_pods=$(kubectl get pods --no-headers -n kube-system -l k8s-app=calico-node | awk '{print $1}')

# Go through each calico/node pod and get its logs, running in the background to do them in 
# parallel.
mkdir -p felix-logs
for pod in $calico_pods; do
	kubectl -n kube-system logs $pod -c calico-node > felix-logs/$pod.log &
done
wait
echo ""
echo "Felix logs saved to $(pwd)/felix-logs/"
