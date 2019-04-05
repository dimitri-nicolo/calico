#!/bin/bash
#
# This script plays out a demo scenario of creating/updating/deleting
# kubernetes and calico resources. It sleeps for 5 seconds in between
# each client call to replicate a system that is modifying these
# resources over time.

NAMESPACE=compliance-testing
KUBECTL="kubectl -n$NAMESPACE"
SLEEP_TIME=10

getResourceVersion() {
  echo `$KUBECTL get $1.v3.projectcalico.org $2 -ocustom-columns=rv:metadata.resourceVersion --no-headers`
}

# create namespace
kubectl create ns $NAMESPACE
sleep $SLEEP_TIME

# create serviceaccount
$KUBECTL apply -f - << EOF
apiVersion: v1
kind: ServiceAccount
metadata:
  name: database
  labels:
    app: database
EOF
sleep $SLEEP_TIME

# create deployment
$KUBECTL apply -f - << EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: database
  labels:
    app: database
spec:
  replicas: 2
  selector:
    matchLabels:
      app: database
  template:
    metadata:
      labels:
        app: database
        version: v1
    spec:
      serviceAccountName: database
      containers:
      - name: database
        image: mysql
        ports:
        - containerPort: 3306
EOF
sleep $SLEEP_TIME

# create service
$KUBECTL apply -f - << EOF
apiVersion: v1
kind: Service
metadata:
  name: database
  labels:
    app: database
spec:
  ports:
  - port: 3306
    name: mysql
  selector:
    app: database
EOF
sleep $SLEEP_TIME

# create k8s networkpolicy
$KUBECTL apply -f - << EOF
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: default-deny
spec:
  podSelector:
    matchLabels: {}
EOF
sleep $SLEEP_TIME

# create calico tier
$KUBECTL apply -f - << EOF
apiVersion: projectcalico.org/v3
kind: Tier
metadata:
  name: internal-access
spec:
  order: 100
EOF
sleep $SLEEP_TIME

# create calico networkpolicy
$KUBECTL apply -f - << EOF
apiVersion: projectcalico.org/v3
kind: NetworkPolicy
metadata:
  name: internal-access.allow-mysql
spec:
  selector: role == "database"
  tier: internal-access
  types:
  - Ingress
  ingress:
  - action: Allow
    protocol: TCP
    destination:
      ports:
      - 3306
EOF
sleep $SLEEP_TIME

# create hostendpoint
$KUBECTL apply -f - << EOF
apiVersion: projectcalico.org/v3
kind: HostEndpoint
metadata:
  name: some.name
  labels:
    role: backend
spec:
  interfaceName: eth0
  node: myhost
EOF
sleep $SLEEP_TIME

# create globalnetworkset
$KUBECTL apply -f - << EOF
apiVersion: projectcalico.org/v3
kind: GlobalNetworkSet
metadata:
  name: google-dns
  labels:
    behavior: snoopy
spec:
  nets:
  - 8.8.8.8/32
EOF
sleep $SLEEP_TIME

# create globalnetworkpolicy
$KUBECTL apply -f - << EOF
apiVersion: projectcalico.org/v3
kind: GlobalNetworkPolicy
metadata:
  name: internal-access.block-google-dns
spec:
  selector: role == "database"
  tier: internal-access
  types:
  - Egress
  egress:
  - action: Deny
    destination:
      selector: behavior == "snoopy"
EOF
sleep $SLEEP_TIME

# modify serviceaccount
$KUBECTL apply -f - << EOF
apiVersion: v1
kind: ServiceAccount
metadata:
  name: database
  labels:
    app: database
    foo: bar
EOF
sleep $SLEEP_TIME

# modify deployment
$KUBECTL apply -f - << EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: database
  labels:
    app: database
spec:
  replicas: 2
  selector:
    matchLabels:
      app: database
  template:
    metadata:
      labels:
        app: database
        version: v2
    spec:
      serviceAccountName: database
      containers:
      - name: database
        image: mysql
        ports:
        - containerPort: 3307
EOF
sleep $SLEEP_TIME

# modify service
$KUBECTL apply -f - << EOF
apiVersion: v1
kind: Service
metadata:
  name: database
  labels:
    app: database
spec:
  ports:
  - port: 3307
    name: mysqls
  selector:
    app: database
EOF
sleep $SLEEP_TIME

# modify k8s network policy
$KUBECTL apply -f - << EOF
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: default-deny
spec:
  podSelector:
    matchLabels: 
      app: summary
EOF
sleep $SLEEP_TIME

# modify calico tier
$KUBECTL apply -f - << EOF
apiVersion: projectcalico.org/v3
kind: Tier
metadata:
  name: internal-access
spec:
  order: 200
EOF
sleep $SLEEP_TIME

# modify calico networkpolicy
$KUBECTL apply -f - << EOF
apiVersion: projectcalico.org/v3
kind: NetworkPolicy
metadata:
  name: internal-access.allow-mysql
  resourceVersion: "$(getResourceVersion networkpolicy internal-access.allow-mysql)"
spec:
  selector: role == "database"
  tier: internal-access
  types:
  - Ingress
  ingress:
  - action: Allow
    protocol: TCP
    destination:
      ports:
      - 3307
EOF
sleep $SLEEP_TIME

# modify hostendpoint
$KUBECTL apply -f - << EOF
apiVersion: projectcalico.org/v3
kind: HostEndpoint
metadata:
  name: some.name
  labels:
    role: backend
spec:
  interfaceName: eth1
  node: myhost
EOF
sleep $SLEEP_TIME

# modify globalnetworkset
$KUBECTL apply -f - << EOF
apiVersion: projectcalico.org/v3
kind: GlobalNetworkSet
metadata:
  name: google-dns
  labels:
    behavior: snoopier
spec:
  nets:
  - 8.8.8.8/32
EOF
sleep $SLEEP_TIME

# modify globalnetworkpolicy
$KUBECTL apply -f - << EOF
apiVersion: projectcalico.org/v3
kind: GlobalNetworkPolicy
metadata:
  name: internal-access.block-google-dns
  resourceVersion: "$(getResourceVersion globalnetworkpolicy internal-access.block-google-dns)"
spec:
  selector: role == "database"
  tier: internal-access
  types:
  - Egress
  egress:
  - action: Deny
    destination:
      selector: behavior == "snoopier"
EOF
sleep $SLEEP_TIME

# oopsy, the default-deny policy is wrong, let's fix that...
$KUBECTL patch networkpolicy default-deny -p '{"spec": {"podSelector": {"matchLabels": {}}}}'
sleep $SLEEP_TIME

# delete resources
$KUBECTL delete globalnetworkpolicy internal-access.block-google-dns
sleep $SLEEP_TIME

$KUBECTL delete globalnetworkset google-dns
sleep $SLEEP_TIME

$KUBECTL delete hostendpoint some.name
sleep $SLEEP_TIME

$KUBECTL delete networkpolicy.v3.projectcalico.org internal-access.allow-mysql
sleep $SLEEP_TIME

$KUBECTL delete tier internal-access
sleep $SLEEP_TIME

$KUBECTL delete networkpolicy default-deny
sleep $SLEEP_TIME

$KUBECTL delete service database
sleep $SLEEP_TIME

$KUBECTL delete deployment database
sleep $SLEEP_TIME

$KUBECTL delete serviceaccount database
sleep $SLEEP_TIME

kubectl delete ns $NAMESPACE
