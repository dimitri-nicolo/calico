#!/bin/sh

# Enable multi cluster client mode for compliance and es proxy
kubectl patch deployment -n calico-monitoring cnx-manager --patch \
'{"spec":{"template":{"spec":{"containers":[{"name":"tigera-es-proxy","env":[{"name":"ENABLE_MULTI_CLUSTER_CLIENT","value":"true"}]}]}}}}'
kubectl patch deployment -n calico-monitoring compliance-server --patch \
'{"spec":{"template":{"spec":{"containers":[{"name":"compliance-server","env":[{"name":"ENABLE_MULTI_CLUSTER_CLIENT","value":"true"}]}]}}}}'


# Allow Guardian to reach Compliance Server
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
