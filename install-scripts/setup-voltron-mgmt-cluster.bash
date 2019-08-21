#!/bin/sh

# Patch the following for CNX Manager containers: 
# - For cnx-manager, define Voltron URL as 127.0.0.1:30003.  This is required for the UI.
# - For cnx-manager, use image for cluster-selection
# - For cnx-manager-proxy, forward Kube API and Compliance traffic to Voltron
# - For cnx-manager-proxy, use image for cluster-selection
# - For es-proxy, integrate with Guardian and allow access on port 8443

kubectl patch deployment -n calico-monitoring cnx-manager --patch \
'{"spec":{"template":{"spec":{"containers":[{"name":"cnx-manager","image":"gcr.io/unique-caldron-775/cnx/tigera/cnx-manager:v2.5.0-mcm0.1","env":[{"name":"CNX_VOLTRON_API_URL","value":"https://127.0.0.1:30003"}]},{"name":"cnx-manager-proxy","image":"gcr.io/unique-caldron-775/cnx/tigera/cnx-manager-proxy:v2.5.0-mcm0.1","env":[{"name":"CNX_KUBE_APISERVER","value":"cnx-voltron-server.calico-monitoring.svc.cluster.local:9443"},{"name":"CNX_COMPLIANCE_SERVER","value":"cnx-voltron-server.calico-monitoring.svc.cluster.local:9443"}]},{"name":"tigera-es-proxy","env":[{"name":"LISTEN_ADDR","value":":8443"}],"image":"gcr.io/tigera-dev/cnx/tigera/es-proxy:v2.5.0-mcm0.1"}]}}}}'

# Monitor deployment for cnx-manager
kubectl rollout status -n calico-monitoring deployment/cnx-manager
if [ $? -ne 0 ]; then
  echo >&2 "Patching cnx-manager deployment failed"
  exit 1
fi

# Apply CRD definition for ManagedCluster resource 
kubectl apply -f - <<EOF
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: managedclusters.crd.projectcalico.org
spec:
  scope: Cluster
  group: crd.projectcalico.org
  version: v1
  names:
    kind: ManagedCluster
    plural: managedclusters
    singular: managedcluster
EOF

# Switch to custom image for cnx-apiserver that knows how to handle the CRD managedclusters
kubectl patch deployment -n kube-system cnx-apiserver --patch \
'{"spec": {"template": {"spec": {"containers": [{"name": "cnx-apiserver", "image": "gcr.io/unique-caldron-775/cnx/tigera/cnx-apiserver:master"}]}}}}'

# Bind cnx-apiserverr to role tigera-mcm
kubectl create clusterrolebinding mcm-binding-cnx-apiserver --clusterrole=tigera-mcm --serviceaccount=kube-system:cnx-apiserver

# Monitor deployment for cnx-apiserver
kubectl rollout status -n kube-system deployment/cnx-apiserver
if [ $? -ne 0 ]; then
  echo >&2 "Patching cnx-apiserver deployment failed"
  exit 1
fi

# Bind authenticated users to role tigera-mcm  
kubectl create clusterrolebinding mcm-binding-user \
 --clusterrole=tigera-mcm-no-crd --group=system:authenticated

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

# Patch Compliance Server to integrate with Guardian
kubectl patch deployment -n calico-monitoring compliance-server --patch \
'{"spec": {"template": {"spec": {"containers": [{"name": "compliance-server", "image": "gcr.io/tigera-dev/cnx/tigera/compliance-server:v2.5.0-mcm0.1"}]}}}}'

# Monitor deployment for compliance-server
kubectl rollout status -n calico-monitoring deployment/compliance-server
if [ $? -ne 0 ]; then
  echo >&2 "Patching compliance-server deployment failed"
  exit 1
fi

