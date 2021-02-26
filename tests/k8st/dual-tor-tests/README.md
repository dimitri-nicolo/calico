
# Manual setup equivalent to what the tests do

Start of day: create client and server pods and services:

```
kubectl create ns dualtor
kubectl run --generator=run-pod/v1 client -n dualtor --image calico-test/busybox-with-reliable-nc --image-pull-policy Never --labels='pod-name=client' --overrides='{ "apiVersion": "v1", "spec": { "nodeSelector": { "kubernetes.io/hostname": "kind-worker" }, "terminationGracePeriodSeconds": 0 } }' --command /bin/sleep -- 3600
kubectl run --generator=run-pod/v1 client-host -n dualtor --image calico-test/busybox-with-reliable-nc --image-pull-policy Never --labels='pod-name=client-host'  --overrides='{ "apiVersion": "v1", "spec": { "hostNetwork": true, "nodeSelector": { "kubernetes.io/hostname": "kind-worker" }, "terminationGracePeriodSeconds": 0 } }' --command /bin/sleep -- 3600
kubectl run --generator=run-pod/v1 ttt -n dualtor --image calico-test/busybox-with-reliable-nc --image-pull-policy Never --labels='pod-name=ra-server,app=server'  --overrides='{ "apiVersion": "v1", "spec": { "nodeSelector": { "kubernetes.io/hostname": "kind-control-plane" }, "terminationGracePeriodSeconds": 0 } }' --command /bin/sleep -- 3600
kubectl run --generator=run-pod/v1 rb-server -n dualtor --image calico-test/busybox-with-reliable-nc --image-pull-policy Never --labels='pod-name=rb-server,app=server'  --overrides='{ "apiVersion": "v1", "spec": { "nodeSelector": { "kubernetes.io/hostname": "kind-worker3" }, "terminationGracePeriodSeconds": 0 } }' --command /bin/sleep -- 3600
kubectl wait --timeout=1m --for=condition=ready pod/client -n dualtor
kubectl wait --timeout=1m --for=condition=ready pod/client-host -n dualtor
kubectl wait --timeout=1m --for=condition=ready pod/ra-server -n dualtor
kubectl wait --timeout=1m --for=condition=ready pod/rb-server -n dualtor

kubectl apply -f - <<EOF
apiVersion: v1
kind: Service
metadata:
  namespace: dualtor
  name: ra-server
  labels:
    name: ra-server
spec:
  ports:
    - port: 8090
  selector:
    pod-name: ra-server
  type: NodePort
---
apiVersion: v1
kind: Service
metadata:
  namespace: dualtor
  name: rb-server
  labels:
    name: rb-server
spec:
  ports:
    - port: 8090
  selector:
    pod-name: rb-server
  type: NodePort
EOF
```

Per test case: start servers:

```
kubectl exec -n dualtor ra-server -- nc -l -p 8090 >ra.log 2>&1 &
kubectl exec -n dualtor rb-server -- nc -l -p 8090 >rb.log 2>&1 &
```

Client for host access test case

```
rb_pod_ip=10.244.195.198
kubectl exec -n dualtor -t client-host -- /bin/sh -c 'for i in `seq 1 3000`; do echo $i -- ; sleep .5; done | nc -w 1 $rb_pod_ip 8090' &
```

Client for pod IP test case

```
kubectl exec -n dualtor -t client -- /bin/sh -c 'for i in `seq 1 3000`; do echo $i -- ; sleep .5; done | nc -w 1 $rb_pod_ip 8090' &
```

Client for node port test case

```
worker2_ip=172.31.20.4
rb_node_port=31535
kubectl exec -n dualtor -t client -- /bin/sh -c 'for i in `seq 1 3000`; do echo $i -- ; sleep .5; done | nc -w 1 172.31.20.4 31535' &
```

Client for service IP test case

```
rb_svc_ip=10.96.215.109
kubectl exec -n dualtor -t client -- /bin/sh -c 'for i in `seq 1 3000`; do echo $i -- ; sleep .5; done | nc -w 1 $rb_svc_ip 8090' &
```
