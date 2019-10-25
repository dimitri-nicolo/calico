#!/bin/sh

# Patch the following for CNX Manager containers: 
# - For cnx-manager, enable ENABLE_MULTI_CLUSTER_MANAGEMENT
# - For tigera-voltron, mount volume to use secret cnx-voltron-tunnel
# - For tigera-voltron, open 9449 port to accept tunnels

# Create certs that will be used for Voltron
mkdir -p /tmp/certs
echo ${PWD}

bash clean-self-signed.sh /tmp/certs
bash self-signed.sh /tmp/certs

os=$(uname -s)
BASE64_ARGS=""
if [ "${os}" = "Linux" ]; then
    BASE64_ARGS="-w 0"
fi

CERT64=$(base64 ${BASE64_ARGS} /tmp/certs/cert)
KEY64=$(base64 ${BASE64_ARGS} /tmp/certs/key)

kubectl apply -f - <<EOF
apiVersion: v1
kind: Secret
metadata:
  name: cnx-voltron-tunnel
type: Opaque
data:
  cert: ${CERT64}
  key: ${KEY64}
EOF


# Extract the internal ip of a the master to populate VOLTRON_PUBLIC_IP
INTERNAL_IP=$(kubectl get nodes -o wide | grep "master" | awk '{print $6}')
echo "Using Voltron Public Ip ${INTERNAL_IP}"
kubectl set env deployment cnx-manager  -ncalico-monitoring -c tigera-voltron VOLTRON_PUBLIC_IP=${INTERNAL_IP}:30449

kubectl patch deployment -n calico-monitoring cnx-manager --patch \
'{"spec":{"template":{"spec":{"containers":[{"name":"cnx-manager","env":[{"name": "ENABLE_MULTI_CLUSTER_MANAGEMENT", "value": "true"}]},{ "name":"tigera-voltron","env":[{"name": "VOLTRON_TUNNEL_PORT", "value": "9449"}], "volumeMounts":[{"mountPath":"/certs/tunnel/","name":"cnx-voltron-tunnel"}, {"mountPath":"/certs/tunnel-not-user/","name":"cnx-manager-tls"}]}],"volumes":[{"name":"cnx-voltron-tunnel","secret":{"secretName":"cnx-voltron-tunnel"}}]}}}}'

kubectl apply -f - <<EOF
apiVersion: v1
kind: Service
metadata:
  name: cnx-voltron
  namespace: calico-monitoring
spec:
  type: NodePort
  ports:
    - port: 9449
      nodePort: 30449
      protocol: TCP
      name: tunnels
  selector:
    k8s-app: cnx-manager

EOF

# Monitor deployment for cnx-manager
kubectl rollout status -n calico-monitoring deployment/cnx-manager
if [ $? -ne 0 ]; then
  echo >&2 "Patching cnx-manager deployment failed"
  exit 1
fi