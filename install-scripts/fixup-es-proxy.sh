#!/bin/sh

kubectl patch deployment -n calico-monitoring cnx-manager --patch '{"spec": {"template": {"spec": {"containers": [{"name": "tigera-es-proxy", "env": [{"name": "LISTEN_ADDR", "value": ":8443"}]}]}}}}'

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
