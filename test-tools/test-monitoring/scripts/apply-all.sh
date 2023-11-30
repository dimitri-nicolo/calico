#!/bin/bash

# Import utility functions.
source ./util-functions.sh

# Assert that a CLUSTER_SIZE has been provided.
if [ "$CLUSTER_SIZE" == "" ]; then
	echo "Error: must specify a CLUSTER_SIZE environment variable"
	exit 1
fi

PROM_MEM=$(get_prom_mem ${CLUSTER_SIZE})
PROM_HIGH_MEM=$(get_prom_high_mem ${CLUSTER_SIZE})
PROM_CPU=$(get_prom_cpu ${CLUSTER_SIZE})
PROM_HIGH_CPU=$(get_prom_high_cpu ${CLUSTER_SIZE})

echo "Configured for a cluster with $CLUSTER_SIZE node(s)"
echo "Setting prometheus memory to $PROM_MEM"
echo "Setting prometheus high memory to $PROM_HIGH_MEM"
echo "Setting prometheus CPU to $PROM_CPU"
echo "Setting prometheus high CPU to $PROM_HIGH_CPU"

if [ $(kubectl get deployment -n tigera-prometheus calico-prometheus-operator --no-headers 2>&1 | grep -c NotFound)  -ne "0" ]; then
  echo "There is no prometheus operator already, so installing one"
  kubectl apply -f ../examples/prometheus/prometheus-operator-service-account.yaml
  kubectl apply -f ../examples/prometheus/prometheus-operator-cluster-role.yaml
  kubectl apply -f ../examples/prometheus/prometheus-operator-cluster-role-binding.yaml
  kubectl apply -f ../examples/prometheus/prometheus-operator.yaml
  echo "Waiting for Prometheus operator to start..."
  podstatus=$(kubectl get pods -n tigera-prometheus -l operator=prometheus --no-headers | awk '{print $3}')
  while [ "$podstatus" != "Running" ];
  do
    podstatus=$(kubectl get pods -n tigera-prometheus -l operator=prometheus --no-headers | awk '{print $3}')
    sleep 2
  done
else
  echo "found calico-prometheus-operator, update it to also watch the namespaces we want to add prometheus scrapers to"
  kubectl get deployment -n tigera-prometheus calico-prometheus-operator -o yaml | \
  sed 's/--namespaces=tigera-prometheus/--namespaces=tigera-prometheus,calico-system,tigera-fluentd/g' | \
  kubectl apply -f -
fi

echo "Waiting for CustomResourceDefinitions to be created"
crdexists=$(kubectl get crd | grep prometheus)
while [ "$crdexists" == "" ];
do
	crdexists=$(kubectl get crd | grep prometheus)
	sleep 2
done
sleep 10

echo ""

for prom in k8s; do
  # We need to encode the config file as a secret.  We use --dry-run=client and pipe
  # that to apply in order to make the command declarative/idempotent.  kubectl create
  # fails if the resource already exists.
  kubectl create secret generic prometheus-$prom \
          --namespace=tigera-fluentd \
          --from-file=prometheus.yaml.gz=../examples/prometheus/$prom.config.yaml \
          -o yaml --dry-run=client | kubectl apply -f -
done

for f in ../examples/prometheus/*.rules.yaml; do
  # Apply prometheus rules for each.
  kubectl apply -f $f
done

for f in ../examples/prometheus/*.spec.yaml; do
  cat "$f" | \
    sed 's/__PROMETHEUS_MEM__/'${PROM_MEM}'/' | \
    sed 's/__PROMETHEUS_HIGH_MEM__/'${PROM_HIGH_MEM}'/' | \
    sed 's/__PROMETHEUS_CPU__/'${PROM_CPU}'/' | \
    sed 's/__PROMETHEUS_HIGH_CPU__/'${PROM_HIGH_CPU}'/' | \
    kubectl apply -f -
done

echo "Waiting for node exporter to start..."

function node_exporter_running() {
	kubectl get daemonset -l app=node-exporter --no-headers | awk '{print $6}'
}

node_exporter_count=$(kubectl get daemonset -l app=node-exporter --no-headers | awk '{print $2}')
while [ "$node_exporter_count" != "$(node_exporter_running)" ];
do
	echo "$(node_exporter_running) of ${node_exporter_count} node exporters have started."
	sleep 2
done
echo "Node exporters have started..."

echo ""

# Enable Felix and Typha prom stats
kubectl patch felixconfiguration default --type merge --patch '{"spec":{"prometheusMetricsEnabled": true}}'
kubectl patch installation default --type=merge -p '{"spec": {"typhaMetricsPort":9093}}'

# We need to enable admin for kube-system pods so that grafana-import succeeds on an RBAC enabled cluster.
echo ""
echo "Enabling admin for kube-system pods"
kubectl create clusterrolebinding default-admin --clusterrole=cluster-admin --serviceaccount=kube-system:default

echo ""

# Optionally skip spinning up grafana to save on resource utilization.
if [ "$SKIP_GRAFANA" == "" ]; then
  # Apply the grafana deployment and related resources.
  for f in ../examples/grafana/*.spec.yaml; do
    cat "$f" | \
      sed 's/__GRAFANA_MEM__/'${PROM_MEM}'/' | \
      sed 's/__GRAFANA_CPU__/'${PROM_CPU}'/' | \
      kubectl apply -f -
  done

  # Wait for Grafana to start, and give some time for things to settle
  # before starting the grafana-import job.
  echo "Waiting for grafana to start"
  state=$(kubectl get pod --no-headers -n kube-system -l app=grafana | awk '{print $3}')
  while [ "$state" != "Running" ] ; do
      state=$(kubectl get pod --no-headers -n kube-system -l app=grafana | awk '{print $3}')
      echo "grafana state: $state"
      sleep 1
  done
  sleep 2

  # Then, deploy any import jobs for grafana.
  echo "Deploy grafana import job"
  for f in ../examples/grafana/*.import.yaml; do
      kubectl apply -f $f
  done

  grafana_import=$(kubectl get pod --no-headers -n kube-system -l job-name=grafana-import | awk '{print $1}')
  while true; do
      grafana_import_status=$(kubectl get pod --no-headers -n kube-system $grafana_import | awk '{print $3}')
      if [ "$grafana_import_status" == "Completed" ]; then
          echo "grafana-import has completed..."
          echo ""
          break
      fi
  done

  echo "kubectl logs -n kube-system $grafana_import..."
  kubectl logs -n kube-system "$grafana_import"
fi

# If an explicit GRAFANA_IP was provided, use that. Otherwise, look for
# the external IP of the grafana service.
if [ "$GRAFANA_IP" == "" ]; then
	echo "Waiting for Grafana service to be available externally..."
	GRAFANA_IP=$(kubectl get svc -n kube-system grafana -o go-template='{{range .status.loadBalancer.ingress}}{{.ip}}{{end}}')
	while [ "$GRAFANA_IP" == "" ]; do
	    GRAFANA_IP=$(kubectl get svc -n kube-system grafana -o go-template='{{range .status.loadBalancer.ingress}}{{.ip}}{{end}}')
	    sleep 1
	done
	echo "Complete"
fi

cat <<EOF
In another terminal, run the following command to proxy localhost to the cluster:

    kubectl proxy -p 8080

You can access grafana at this URL:

    http://$GRAFANA_IP:3000/dashboard/db/live-test-data

EOF
