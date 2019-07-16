
calicoctl apply -f - <<EOF
---
apiVersion: projectcalico.org/v3
kind: NetworkPolicy
metadata:
  name: allow-cnx.allow-voltron-access
  namespace: calico-monitoring
spec:
  tier: allow-cnx
  order: 0
  selector: k8s-app == "cnx-voltron"
  ingress:
    - action: Allow
      protocol: TCP
      source:
        nets:
          - 0.0.0.0/0
      destination:
        ports:
          - '9443'
          - '9449'
  types:
    - Ingress
EOF

kubectl apply -f voltron.yaml
if [ $? -ne 0 ]; then
    echo >&2 "Deploying voltron failed"
    exit 1
fi

kubectl rollout status -n calico-monitoring deployment/cnx-voltron

kubectl patch deployment -n calico-monitoring cnx-manager --patch \
    '{"spec": {"template": {"spec": {"containers": [{"name": "cnx-manager", "env": [{"name": "CNX_VOLTRON_API_URL", "value": "https://127.0.0.1:30003"}]}]}}}}'
kubectl patch deployment -n calico-monitoring cnx-manager --patch \
    '{"spec": {"template": {"spec": {"containers": [{"name": "cnx-manager", "image":"gcr.io/unique-caldron-775/cnx/tigera/cnx-manager:cluster-selection"}]}}}}'

kubectl patch deployment -n calico-monitoring cnx-manager --patch \
    '{"spec": {"template": {"spec": {"containers": [{"name": "cnx-manager-proxy", "env": [{"name": "CNX_KUBE_APISERVER", "value": "cnx-voltron-server.calico-monitoring.svc.cluster.local:9443"}]}]}}}}'
kubectl patch deployment -n calico-monitoring cnx-manager --patch \
    '{"spec": {"template": {"spec": {"containers": [{"name": "cnx-manager-proxy", "image":"gcr.io/tigera-dev/cnx/tigera/cnx-manager-proxy:voltron-ssl-no-verify"}]}}}}'


kubectl rollout status -n calico-monitoring deployment/cnx-manager
if [ $? -ne 0 ]; then
    echo >&2 "Patching cnx-manager deployment failed"
    exit 1
fi
