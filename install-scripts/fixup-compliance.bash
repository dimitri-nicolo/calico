#!/bin/sh

#Patch compliance server
kubectl patch deployment -n calico-monitoring cnx-manager --patch     '{"spec": {"template": {"spec": {"containers": [{"name": "cnx-manager-proxy", "env": [{"name": "CNX_COMPLIANCE_SERVER", "value": "cnx-voltron-server.calico-monitoring.svc.cluster.local:9443"}]}]}}}}'


kubectl apply -f - << EOF
apiVersion: projectcalico.org/v3
kind: NetworkPolicy
metadata:
  name: allow-cnx.compliance-server
  namespace: calico-monitoring
spec:
  tier: allow-cnx
  order: 1
  selector: k8s-app == "compliance-server"
  ingress:
    - action: Allow
      protocol: TCP
      source:
        selector: k8s-app == "cnx-manager"||k8s-app == "cnx-guardian"
      destination:
        ports:
          - '5443'
  types:
    - Ingress
EOF

kubectl patch deployment -n calico-monitoring compliance-server --patch '{"spec": {"template": {"spec": {"containers": [{"name": "compliance-server", "image": "gcr.io/tigera-dev/cnx/tigera/compliance-server:impersonation-v1"}]}}}}'

