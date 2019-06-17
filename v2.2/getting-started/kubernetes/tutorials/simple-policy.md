---
title: Simple Policy Demo
canonical_url: https://docs.tigera.io/v2.3/getting-started/kubernetes/tutorials/simple-policy
---

This guide provides a simple way to try out Kubernetes NetworkPolicy with {{site.prodname}}.  It requires a Kubernetes cluster configured with {{site.prodname}} networking, and expects that you have `kubectl` configured to interact with the cluster.

You can quickly and easily deploy such a cluster by following one of the [installation guides]({{site.baseurl}}/{{page.version}}/getting-started/kubernetes/installation/)

### Configure Namespaces

This guide will deploy pods in a Kubernetes namespaces.  Let's create the `Namespace` object for this guide.

```
kubectl create ns policy-demo
```

### Create demo Pods

We'll use Kubernetes `Deployment` objects to easily create pods in the `Namespace`.

1) Create some nginx pods in the `policy-demo` namespace, and expose them through a service.

Run the pods.

```shell
kubectl run --namespace=policy-demo nginx --replicas=2 --image=nginx
```

Create the service.

```
kubectl expose --namespace=policy-demo deployment nginx --port=80
```

2) Ensure the nginx service is accessible.

Run a pod within the `policy-demo` namespace.

```
kubectl run --namespace=policy-demo access --rm -ti --image busybox /bin/sh
```

Attempt to connect to the nginx service.

```
If you don't see a command prompt, try pressing enter.

/ # wget -q nginx -O -
```

You should see a response from `nginx`.  Great! Our service is accessible.  You can exit the pod now.

### Enable isolation

Let's turn on isolation in our policy-demo namespace.  {{site.prodname}} will then prevent connections to pods in this namespace.

Running the following command creates a NetworkPolicy which implements a default deny behavior for all pods in the `policy-demo` namespace.

```
kubectl create -f - <<EOF
kind: NetworkPolicy
apiVersion: networking.k8s.io/v1
metadata:
  name: default-deny
  namespace: policy-demo
spec:
  podSelector:
    matchLabels: {}
EOF
```

#### Test Isolation

This will prevent all access to the nginx service.  We can see the effect by trying to access the service again.

Run a pod in the `policy-demo` namespace.

```
kubectl run --namespace=policy-demo access --rm -ti --image busybox /bin/sh
```

Attempt to access the nginx service again.

```
If you don't see a command prompt, try pressing enter.

/ # wget -q --timeout=5 nginx -O -
```

After 5 seconds, the request should time out and you should see the following output.

```
wget: download timed out
```

By enabling isolation on the namespace, we've prevented access to the service.

### Allow Access using a NetworkPolicy

Now, let's enable access to the nginx service using a NetworkPolicy.  This will allow incoming connections from our `access` pod, but not
from anywhere else.

Create a network policy `access-nginx` with the following contents:

```
kubectl create -f - <<EOF
kind: NetworkPolicy
apiVersion: networking.k8s.io/v1
metadata:
  name: access-nginx
  namespace: policy-demo
spec:
  podSelector:
    matchLabels:
      run: nginx
  ingress:
    - from:
      - podSelector:
          matchLabels:
            run: access
EOF
```

> **Note**: The NetworkPolicy allows traffic from pods with
> the label `run: access` to pods with the label `run: nginx`. These
> are the labels automatically added to pods started via `kubectl run`
> based on the name of the `Deployment`.
{: .alert .alert-info}

We should now be able to access the service from the `access` pod.

Run a pod in the `policy-demo` namespace.

```
kubectl run --namespace=policy-demo access --rm -ti --image busybox /bin/sh
```

Attempt to access the nginx service.

```
If you don't see a command prompt, try pressing enter.

/ # wget -q --timeout=5 nginx -O -
```

However, we still cannot access the service from a pod without the label `run: access`:

Run a pod in the `policy-demo` namespace with a different label.

```
kubectl run --namespace=policy-demo cant-access --rm -ti --image busybox /bin/sh
```

Attempt to connect to the nginx service.

```
If you don't see a command prompt, try pressing enter.

/ # wget -q --timeout=5 nginx -O -
```

After 5 seconds, you should see the following output.

```
wget: download timed out
```

You can clean up the demo by deleting the demo namespace:

```shell
kubectl delete ns policy-demo
```

This was just a simple example of the Kubernetes NetworkPolicy API and how {{site.prodname}} can secure your Kubernetes cluster.  For more
information on network policy in Kubernetes, see the [Kubernetes user-guide](http://kubernetes.io/docs/user-guide/networkpolicies/).

For a slightly more detailed demonstration of Policy, check out the [stars demo](stars-policy/).
