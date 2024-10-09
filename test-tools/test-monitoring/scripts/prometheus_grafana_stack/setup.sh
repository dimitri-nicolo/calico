#!/bin/bash

# Reference Doc: https://github.com/prometheus-community/helm-charts/tree/main/charts/kube-prometheus-stacks

echo "Setting up the kube, grafana and prometheus monitoring stack using helm charts"

helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo add grafana https://grafana.github.io/helm-charts
helm repo update

echo "Installing prometheus-grafana"
helm install prometheus prometheus-community/kube-prometheus-stack --namespace monitoring --create-namespace

echo "Getting grafana admin password"
PASSWORD=$(kubectl get secret --namespace monitoring prometheus-grafana -o jsonpath="{.data.admin-password}" | base64 --decode ; echo)

echo "In another terminal, port forward to grafana dashboard by running:"
echo "kubectl port-forward --namespace monitoring svc/prometheus-grafana 3000:80"

echo "Then open a browser and go to grafana at https://127.0.0.1:3000"
echo "Use these credentials to login to grafana:"
echo "User: admin"
echo "Password: ${PASSWORD}"

# Uninstalling
# helm uninstall prometheus -n monitoring
