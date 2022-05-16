#!/bin/sh

echo "Creating service account ui-user"
kubectl create sa ui-user
echo "Binding ui-user to tigera-network-admin clusterrole"
kubectl create clusterrolebinding binding-ui-user --clusterrole tigera-network-admin --serviceaccount default:ui-user

