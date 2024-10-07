#!/bin/sh

# Enable multi cluster client mode for compliance and es proxy
kubectl patch deployment -n tigera-manager tigera-manager --patch \
'{"metadata": {"annotations": {"unsupported.operator.tigera.io/ignore": "true"}}, "spec":{"template":{"spec":{"containers":[{"name":"tigera-ui-apis","env":[{"name":"ENABLE_MULTI_CLUSTER_CLIENT","value":"true"}]}]}}}}'
kubectl patch deployment -n tigera-compliance compliance-server --patch \
'{"metadata": {"annotations": {"unsupported.operator.tigera.io/ignore": "true"}}, "spec":{"template":{"spec":{"containers":[{"name":"compliance-server","env":[{"name":"ENABLE_MULTI_CLUSTER_CLIENT","value":"true"}]}]}}}}'



# Allow Guardian to reach Compliance Server
kubectl apply -f - <<EOF
apiVersion: projectcalico.org/v3
kind: NetworkPolicy
metadata:
  name: allow-tigera.compliance-server
  namespace: tigera-compliance
spec:
  tier: allow-tigera
  order: 1
  selector: k8s-app == "compliance-server"
  serviceAccountSelector: ''
  ingress:
    - action: Allow
      protocol: TCP
      source:
        selector: k8s-app == "tigera-manager"
        namespaceSelector: name == "tigera-manager"
      destination:
        ports:
          - '5443'
    - action: Allow
      protocol: TCP
      source:
        selector: k8s-app == "tigera-guardian"
        namespaceSelector: name == "tigera-mcm"
      destination:
        ports:
          - '5443'
  types:
    - Ingress
EOF
