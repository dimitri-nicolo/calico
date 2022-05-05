#!/usr/bin/env bash

APISERVER=$(kubectl config view --minify -o jsonpath='{.clusters[0].cluster.server}')

echo "Kubernetes Api Server is ${APISERVER}"
