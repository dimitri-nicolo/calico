#!/bin/bash

DATASTORE=${DATASTORE:="etcdv3"}
VERSION=${VERSION:="v2.3"}
TARGET_DIR=${TARGET_DIR:="."}
NAMESPACE=${NAMESPACE:="--all-namespaces"}

CALICOCTL="kubectl exec -nkube-system calicoctl -- /calicoctl"

ensure_calicoctl() {
	echo "# Checking if calicoctl pod exists..."
	if ! $CALICOCTL get nodes > /dev/null 2>&1; then
		if [[ "$DATASTORE" == "etcdv3" ]]; then
			echo "# Creating etcd calicoctl pod..."
			kubectl apply -f https://docs.tigera.io/$VERSION/getting-started/kubernetes/installation/hosted/calicoctl.yaml
		else
			echo "# Creating kdd calicoctl pod..."
			kubectl apply -f https://docs.tigera.io/$VERSION/getting-started/kubernetes/installation/hosted/kubernetes-datastore/calicoctl.yaml
		fi
		echo "# calicoctl pod successfully started!"
		sleep 2
	else
		echo "# calicoctl pod exists, continuing"
	fi
}

# Pull k8s resources
declare -a k8s_resources=("pods" "namespaces" "endpoints" "services" "serviceaccounts" "networkpolicies")

for res in "${k8s_resources[@]}"
do
	file=$TARGET_DIR/k8s.$res.yaml
	kubectl get $res $NAMESPACE -oyaml > $file
	echo "+" $file
done

# Pull Tigera resources
ensure_calicoctl

# Non-namespaced resources
declare -a global_tigera_resources=("globalnetworkpolicies" "hostendpoints" "globalnetworksets")

for res in "${global_tigera_resources[@]}"
do
	file=$TARGET_DIR/tigera.$res.yaml
	$CALICOCTL get $res -oyaml > $file
	echo "+" $file
done

# Namespaced resources
declare -a namespaced_tigera_resources=("networkpolicies" "networksets")

for res in "${namespaced_tigera_resources[@]}"
do
	file=$TARGET_DIR/tigera.$res.yaml
	$CALICOCTL get $res $NAMESPACE -oyaml > $file
	echo "+" $file
done
