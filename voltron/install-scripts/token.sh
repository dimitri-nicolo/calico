#!/bin/sh

echo "Retrieving token for ui-user"
kubectl get secret $(kubectl get secrets --all-namespaces | awk '/ui-user/{print $2}') -n default -o jsonpath="{.data.token}" | base64 --decode


