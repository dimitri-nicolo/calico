#!/bin/sh

# Patch the following for CNX Manager containers:
# - For cnx-manager-proxy, replace HA proxy with Voltron
# - For cnx-manager-proxy, update liveness probe path
# - For cnx-manager-proxy, set VOLTRON_PORT to 9443
# - For cnx-manager-proxy, mount https and tunnel certificates for Voltron

kubectl patch deployment -n calico-monitoring cnx-manager --patch \
'{"spec":{"template":{"spec":{"containers":[{"name":"cnx-manager-proxy","image":"gcr.io/unique-caldron-775/cnx/tigera/voltron:master","env":[{"name": "VOLTRON_PORT", "value": "9443"}], "volumeMounts":[{"mountPath":"/certs/https/","name":"cnx-manager-tls"},{"mountPath":"/certs/tunnel/","name":"cnx-manager-tls"}], "livenessProbe" : {"httpGet": {"path": "/voltron/api/health"}}}]}}}}'

# Patch the following for CNX Manager containers:
# - For cnx-manager, use master version
kubectl patch deployment -n calico-monitoring cnx-manager --patch \
'{"spec": {"template": {"spec": {"containers": [{"name": "cnx-manager", "image": "gcr.io/unique-caldron-775/cnx/tigera/cnx-manager:master"}]}}}}'

# Patch the following for CNX Manager containers:
# - For tigera-es-proxy, use master version
kubectl patch deployment -n calico-monitoring cnx-manager --patch \
'{"spec": {"template": {"spec": {"containers": [{"name": "tigera-es-proxy", "image": "gcr.io/unique-caldron-775/cnx/tigera/es-proxy:master"}]}}}}'

# Monitor deployment for cnx-manager
kubectl rollout status -n calico-monitoring deployment/cnx-manager
if [ $? -ne 0 ]; then
  echo >&2 "Patching cnx-manager deployment failed"
  exit 1
fi

# Create cluster roles to access managed clusters
kubectl apply -f - <<EOF
---
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1beta1
metadata:
    name: tigera-mcm-no-crd
rules:
    - apiGroups: ["projectcalico.org"]
      resources: ["managedclusters"]
      verbs: ["*"]
---
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1beta1
metadata:
    name: tigera-mcm
rules:
    - apiGroups: ["crd.projectcalico.org"]
      resources: ["managedclusters"]
      verbs: ["*"]
EOF

# Bind cnx-manager to tigera-mcm-no-crd
kubectl create clusterrolebinding mcm-binding-cnx-manager \
 --clusterrole=tigera-mcm-no-crd --serviceaccount=calico-monitoring:cnx-manager

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

# Patch the following for Compliance containers:
# - For compliance-server, use master version
kubectl patch deployment -n calico-monitoring compliance-server --patch \
'{"spec": {"template": {"spec": {"containers": [{"name": "compliance-server", "image": "gcr.io/unique-caldron-775/cnx/tigera/compliance-server:master"}]}}}}'

# Monitor deployment for cnx-manager
kubectl rollout status -n calico-monitoring deployment/compliance-server
if [ $? -ne 0 ]; then
  echo >&2 "Patching compliance-server deployment failed"
  exit 1
fi
