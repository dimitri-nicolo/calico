#!/bin/sh
ETCD_VERSION=v3.3.7
K8S_VERSION=v1.11.3

BASEDIR=`dirname $0`

docker run --detach \
    --net=host \
    --entrypoint=/usr/local/bin/etcd \
    --name k8s-etcd quay.io/coreos/etcd:${ETCD_VERSION} \
    --advertise-client-urls "http://127.0.0.1:2379,http://127.0.0.1:4001" \
    --listen-client-urls "http://127.0.0.1:2379,http://127.0.0.1:4001"

docker run \
        --net=host --name st-apiserver \
        --detach \
        -v `realpath ${BASEDIR}`/k8s-api-certs:/root/certs \
        -v `realpath ${BASEDIR}`/tokens:/root/tokens \
        gcr.io/google_containers/hyperkube-amd64:${K8S_VERSION} \
        /hyperkube apiserver \
            --bind-address=127.0.0.1 \
            --insecure-bind-address=127.0.0.1 \
            --etcd-servers=http://127.0.0.1:2379 \
            --admission-control=NamespaceLifecycle,LimitRanger,DefaultStorageClass,ResourceQuota,ServiceAccount \
            --authorization-mode=RBAC \
            --service-cluster-ip-range=10.101.0.0/16 \
            --token-auth-file=/root/tokens/tokens.csv \
            --v=10 \
            --logtostderr=true \
            --tls-cert-file /root/certs/k8s.crt \
            --tls-private-key-file /root/certs/k8s.key

while ! curl http://localhost:8080/api  ; do echo "Waiting for apiserver to come up..."; sleep 2; done

docker run \
        --net=host --name st-controller-manager \
        --detach \
        -v `realpath ${BASEDIR}`/k8s-api-certs:/root/certs \
        gcr.io/google_containers/hyperkube-amd64:${K8S_VERSION} \
        /hyperkube controller-manager \
                        --master=127.0.0.1:8080 \
                        --min-resync-period=3m \
                        --allocate-node-cidrs=true \
                        --cluster-cidr=10.10.0.0/16 \
                        --v=5 \
                        --cluster-signing-cert-file /root/certs/k8s.crt \
                        --cluster-signing-key-file /root/certs/k8s.key \
                        --service-account-private-key-file /root/certs/k8s.key


kubectl -s 127.0.0.1:8080 apply -f - << EOF
apiVersion: v1
kind: ServiceAccount
metadata:
  name: cnx-guardian
  namespace: default
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: cnx-guardian-impersonator-role
rules:
  - apiGroups: [""]
    resources: ["users", "groups"]
    verbs: ["impersonate"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: cnx-guardian-impersonator
roleRef:
  kind: ClusterRole
  name: cnx-guardian-impersonator-role
  apiGroup: rbac.authorization.k8s.io
subjects:
  - kind: ServiceAccount
    name: cnx-guardian
    namespace: default
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: reader-role
rules:
  - apiGroups: [""]
    resources: ["pods", "namespaces", "serviceaccounts"]
    verbs: ["get", "watch", "list"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: jane-reader-binding
subjects:
- kind: User
  name: Jane
  apiGroup: rbac.authorization.k8s.io
roleRef:
  kind: ClusterRole
  name: reader-role
  apiGroup: rbac.authorization.k8s.io
EOF

while ! kubectl -s 127.0.0.1:8080 get secrets | grep guardian; do
	echo "waiting for token"
	sleep 1
done

TMPDIR=${BASEDIR}/tmp
mkdir -p $TMPDIR

# get the service account token so that we can fake mounting it into a pod
kubectl -s 127.0.0.1:8080 get secret `kubectl  -s 127.0.0.1:8080 get secrets | grep guardian | awk '{print $1}'` -o jsonpath='{.data.token}' | base64 -d - > $TMPDIR/token
eval $TEST_CMD

docker rm -f st-controller-manager st-apiserver k8s-etcd
rm -rf $TMPDIR/*
