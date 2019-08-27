#!/usr/bin/env bash

# Setup kubectl
export KUBECONFIG="$(kind get kubeconfig-path --name="kind")"

echo "INFO: Installing guardian.yaml"
# Change IP of guardian manifest
IP=$(kubectl get nodes --namespace calico-monitoring -o jsonpath="{.items[0].status.addresses[0].address}")
PORT=$(kubectl get --namespace calico-monitoring -o jsonpath="{.spec.ports[1].nodePort}" services cnx-voltron-server)
echo "Connecting to ip ${IP} and port: ${PORT}"

# Modify the guardian image such that it uses your local image. Also change the voltron host.
sed -i 's/image:.*$/image: tigera\/guardian:st-image/g' test-resources/guardian.yaml
sed -i "s/cnx-guardian.voltron-url: 127.0.0.1:30449/cnx-guardian.voltron-url: $IP:$PORT/g" test-resources/guardian.yaml

kubectl apply -f test-resources/guardian.yaml
kubectl rollout status deployment/cnx-guardian -n calico-monitoring
