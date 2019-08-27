#!/usr/bin/env bash

export KUBECONFIG="$(kind get kubeconfig-path --name="kind")"
kubectl create clusterrolebinding superuser --clusterrole=cluster-admin --serviceaccount=default:default > /dev/null

token=`kubectl get secret -o=jsonpath="{.items[0].data.token}"`
echo ${token}
