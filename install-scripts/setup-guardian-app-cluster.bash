#!/bin/sh

# Patch ES-proxy to integrate with Guardian and allow access on port 8443
kubectl patch deployment -n calico-monitoring cnx-manager --patch \
'{"spec":{"template":{"spec":{"containers":[{"name":"tigera-es-proxy","env":[{"name":"LISTEN_ADDR","value":":8443"}],"image":"gcr.io/tigera-dev/cnx/tigera/es-proxy:v2.5.0-mcm0.1"}]}}}}'

# Allow Guardian to reach ES Proxy 
kubectl apply -f - << EOF
apiVersion: v1
kind: Service
metadata:
  labels:
    k8s-app: cnx-manager
  name: cnx-es-proxy-local
  namespace: calico-monitoring
spec:
  selector:
    k8s-app: cnx-manager
  ports:
    - port: 8443
      nodePort: 30843
      protocol: TCP
  type: NodePort
---
apiVersion: projectcalico.org/v3
kind: NetworkPolicy
metadata:
  name: allow-cnx.guardian-es-proxy
  namespace: calico-monitoring
spec:
  tier: allow-cnx
  order: 100
  selector: k8s-app == "cnx-manager"
  ingress:
    - action: Allow
      protocol: TCP
      source:
        nets:
          - 0.0.0.0/0
      destination:
        ports:
          - '8443'
  types:
    - Ingress
EOF

# Patch Compliance Server to integrate with Guardian
kubectl patch deployment -n calico-monitoring compliance-server --patch \
'{"spec": {"template": {"spec": {"containers": [{"name": "compliance-server", "image": "gcr.io/tigera-dev/cnx/tigera/compliance-server:v2.5.0-mcm0.1"}]}}}}'

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
