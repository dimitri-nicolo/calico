#!/bin/bash

echo "Installing Titan scale testing tool..."

echo "Creating titan namespace"
kubectl create ns titan

echo "============================== STEP 1 =============================="

echo "Run the create-managed-clusters.go script in fake-guardian. This will create the managed cluster connection secrets on the managed clusters (default is 10), and the managed cluster resource on the management cluster."
echo "CALICO_PRIVATE_PATH=<path to calico-private> cd \$CALICO_PRIVATE_PATH/fake-guardian/cmd/create-managed-clusters"
echo "MANAGEMENT_KUBECONFIG=<path> TENANT_ID=<id>  MANAGED_KUBECONFIG=<path> go run main.go"

echo -e "====================================================================\n"

echo "Once this step is complete, press any key to continue setting up titan..."
read -n 1 -s

echo "============================== STEP 2 =============================="
## Apply tigera-pull-secret
echo "Retrieve an existing pull secret and applying it to the titan namespace:"
kubectl -n tigera-operator get secret tigera-pull-secret -o json | jq 'del(.metadata)|.metadata.name="tigera-pull-secret"|.metadata.namespace="titan"' | kubectl apply -f -

## Apply required secrets
kubectl -n tigera-fluentd get secret fluentd-node-tigera-linseed-token -o json | jq 'del(.metadata)|.metadata.name="fluentd-node-tigera-linseed-token"|.metadata.namespace="titan"' | kubectl apply -f -

kubectl -n tigera-fluentd get secret tigera-fluentd-prometheus-tls -o json | jq 'del(.metadata)|.metadata.name="tigera-fluentd-prometheus-tls"|.metadata.namespace="titan"' | kubectl apply -f -

# Copy over the CA bundle configmap for tigera-linseed to titan namespace
kubectl -n tigera-fluentd get cm tigera-ca-bundle -o json | jq -c 'del(.metadata)|.metadata.name="tigera-ca-bundle-linseed"|.metadata.namespace="titan"' | kubectl apply -f -

# Copy over the CA bundle configmap for tigera-guardian to titan namespace
kubectl -n tigera-guardian get cm tigera-ca-bundle -o json | jq -c 'del(.metadata)|.metadata.name="tigera-ca-bundle"|.metadata.namespace="titan"' | kubectl apply -f -

## Scale replicas to desired amount
echo "How many replicas of titan do you want to spin up? Enter an integer."
read replicas
echo "Sounds good, will spin up titan statefulset to ${replicas} replicas."

echo "============================== STEP 3 =============================="
echo "Install titan by applying the titan.yaml manifest:"
echo "CALICO_PRIVATE_PATH=<path to calico-private> kubectl apply -f \$CALICO_PRIVATE_PATH/test-tools/titan/manifests/titan.yaml"
echo -e "====================================================================\n"

echo "Once this step is complete, press any key to continue scaling titan to ${replicas} replicas"
read -n 1 -s

echo "Scaling titan statefulset to ${replicas} replicas."
kubectl -n titan scale statefulset titan-ss --replicas ${replicas}
kubectl rollout status --watch --timeout=300s statefulset titan-ss -n titan

echo "============================== END =============================="
echo "Titan install process complete! ツ"
kubectl get pods -n titan

echo -e "====================================================================\n"
echo "If you want to monitor the cluster further you can deploy the grafana prometheus stack to your cluster by running the setup script in test-monitoring folder:"
echo "cd .. && ./test-monitoring/scripts/prometheus_grafana_stack/setup.sh"
