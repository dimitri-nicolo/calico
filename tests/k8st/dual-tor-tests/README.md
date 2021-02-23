
# Manual setup equivalent to what the tests do

Start of day: create client and server pods:

```
kubectl create ns dualtor
kubectl run --generator=run-pod/v1 client -n dualtor --image busybox:1.32 --labels='pod-name=client' --overrides='{ "apiVersion": "v1", "spec": { "nodeSelector": { "kubernetes.io/hostname": "kind-worker" }, "terminationGracePeriodSeconds": 0 } }' --command /bin/sleep -- 3600
kubectl run --generator=run-pod/v1 client-host -n dualtor --image busybox:1.32 --labels='pod-name=client-host'  --overrides='{ "apiVersion": "v1", "spec": { "hostNetwork": true, "nodeSelector": { "kubernetes.io/hostname": "kind-worker" }, "terminationGracePeriodSeconds": 0 } }' --command /bin/sleep -- 3600
kubectl run --generator=run-pod/v1 ra-server -n dualtor --image busybox:1.32 --labels='pod-name=ra-server,app=server'  --overrides='{ "apiVersion": "v1", "spec": { "nodeSelector": { "kubernetes.io/hostname": "kind-control-plane" }, "terminationGracePeriodSeconds": 0 } }' --command /bin/sleep -- 3600
kubectl run --generator=run-pod/v1 rb-server -n dualtor --image busybox:1.32 --labels='pod-name=rb-server,app=server'  --overrides='{ "apiVersion": "v1", "spec": { "nodeSelector": { "kubernetes.io/hostname": "kind-worker3" }, "terminationGracePeriodSeconds": 0 } }' --command /bin/sleep -- 3600
kubectl wait --timeout=1m --for=condition=ready pod/client -n dualtor
kubectl wait --timeout=1m --for=condition=ready pod/client-host -n dualtor
kubectl wait --timeout=1m --for=condition=ready pod/ra-server -n dualtor
kubectl wait --timeout=1m --for=condition=ready pod/rb-server -n dualtor
```

Per test case: start servers:

```
kubectl exec -n dualtor ra-server -- nc -l -p 8090 >ra.log 2>&1 &
kubectl exec -n dualtor rb-server -- nc -l -p 8090 >rb.log 2>&1 &
```

Per test case: start clients:

```
kubectl exec -n dualtor -t client-host -- /bin/sh -c 'for i in `seq 1 3000`; do echo $i -- ; sleep 1; done | nc -w 1 10.244.195.197 8090' &
```
