---
title: Networking Essentials for Kubernetes Demo
---

This guide is a variation of the simple policy demo intended to introduce the extra features of Tigera Networking Essentials to people already familiar with Project Calico for Kubernetes.

It requires a Kubernetes cluster configured with Calico networking and Networking Essentials, and expects that you have `kubectl` configured to interact with the cluster.

You can quickly and easily obtain such a cluster by setting up [Networking Essentials]({{site.baseurl}}/{page.version}}/getting-started/essentials), and then:
- following one of the [installation guides]({{site.baseurl}}/{{page.version}}/getting-started/kubernetes/installation), or
- [upgrading an existing cluster]({{site.baseurl}}/{page.version}}/getting-started/kubernetes/upgrade).

The key steps in moving to essentials are to change to the essentials version of calico-node, update its configuration, download calicoq and deploy Prometheus.

### Configure Namespaces

This guide will deploy pods in a Kubernetes Namespaces.  Let's create the `Namespace` object for this guide.

```
kubectl create ns policy-demo
```

### Create demo Pods

We'll use Kubernetes `Deployment` objects to easily create pods in the `Namespace`.

1) Create some nginx pods in the `policy-demo` Namespace, and expose them through a Service.

```shell
# Run the Pods.
kubectl run --namespace=policy-demo nginx --replicas=2 --image=nginx

# Create the Service.
kubectl expose --namespace=policy-demo deployment nginx --port=80
```

2) Ensure the nginx service is accessible.

```
# Run a Pod and try to access the `nginx` Service.
$ kubectl run --namespace=policy-demo access --rm -ti --image busybox /bin/sh
Waiting for pod policy-demo/access-472357175-y0m47 to be running, status is Pending, pod ready: false

If you don't see a command prompt, try pressing enter.

/ # wget -q nginx -O -
```

You should see a response from `nginx`.  Great! Our Service is accessible.  You can exit the Pod now.

Now let's inspect the network policies using calicoq.  calicoq complements calicoctl by inspecting the
dynamic aspects of Calico Policy: in particular displaying the endpoints actually affected by policies,
and the policies that actually apply to endpoints.

The full calicoq documentation is [here]({{site.baseurl}}/{{page.version}}/reference/calicoq).
```
# Point calicoq at etcd / the Kubernetes API Server in the same way as calicoctl.  You can also use a config file.
# The host command displays information about the policies that select endpoints on a host.
ETCD_ENDPOINTS=http://10.96.232.136:6666 ./calicoq host k8s-node1
Policies that match each endpoint:

Workload endpoint k8s/calico-monitoring.alertmanager-calico-node-alertmanager-0/eth0
  # These are the policies that apply directly to the endpoint.  calicoq can display both
  # Calico Policies and Kubernetes NetworkPolicies, although this example focuses on the latter.
  # They're listed in the order they apply.
  Policies:
    # These first two policies are defined in the calico-monitoring.yaml manifest.
    # The selectors here have been translated from the original NetworkPolicies to the Calico
    # format (note the addition of the namespace test).
    Policy "calico-monitoring.calico-node-alertmanager" (order 1000; selector "calico/k8s_ns == 'calico-monitoring' && app == 'alertmanager' && alertmanager == 'calico-node-alertmanager'")
    Policy "calico-monitoring.calico-node-alertmanager-mesh" (order 1000; selector "calico/k8s_ns == 'calico-monitoring' && app == 'alertmanager' && alertmanager == 'calico-node-alertmanager'")
    # This policy and the profile following it are created automatically by the policy controller.
    Policy "k8s-policy-no-match" (order 2000; selector "has(calico/k8s_ns)")
  Profiles:
    Profile k8s_ns.calico-monitoring
  # These are the policies that match the endpoint in their rules.
  Matched by policies:
    Policy calico-monitoring.calico-node-alertmanager-mesh (rule 0 inbound source match; selector "app in { "alertmanager" } && alertmanager in { "calico-node-alertmanager" } && calico/k8s_ns == 'calico-monitoring'")

...

Workload endpoint k8s/calico-monitoring.prometheus-calico-node-prometheus-0/eth0
  Policies:
    Policy "calico-monitoring.prometheus" (order 1000; selector "calico/k8s_ns == 'calico-monitoring' && app == 'prometheus' && prometheus == 'calico-node-prometheus'")
    Policy "k8s-policy-no-match" (order 2000; selector "has(calico/k8s_ns)")
  Profiles:
    Profile k8s_ns.calico-monitoring

# This endpoint has no NetworkPolicies configured - just the default Calico policy to allow traffic.
Workload endpoint k8s/policy-demo.nginx-2371676037-7w78m/eth0
  Policies:
    Policy "k8s-policy-no-match" (order 2000; selector "has(calico/k8s_ns)")
  Profiles:
    Profile k8s_ns.policy-demo

...
```

### Enable isolation

Let's turn on isolation in our policy-demo Namespace.  Calico will then prevent connections to pods in this Namespace.

```
kubectl annotate ns policy-demo "net.beta.kubernetes.io/network-policy={\"ingress\":{\"isolation\":\"DefaultDeny\"}}"
```

This will prevent all access to the nginx Service.  We can see the effect by trying to access the Service again.

```
# Run a Pod and try to access the `nginx` Service.
$ kubectl run --namespace=policy-demo access --rm -ti --image busybox /bin/sh
Waiting for pod policy-demo/access-472357175-y0m47 to be running, status is Pending, pod ready: false

If you don't see a command prompt, try pressing enter.

/ # wget -q --timeout=5 nginx -O -
wget: download timed out
/ #
```

The request should time out after 5 seconds.  By enabling isolation on the Namespace, we've prevented access to the Service.

### Allow Access using a NetworkPolicy

Now, let's enable access to the nginx Service using a NetworkPolicy.  This will allow incoming connections from our `access` Pod, but not
from anywhere else.

Create a network policy `access-nginx` with the following contents:

```
kubectl create -f - <<EOF
kind: NetworkPolicy
apiVersion: extensions/v1beta1
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

> Notice the NetworkPolicy allows traffic from Pods with the label `run: access` to Pods with the label `run: nginx`.  These are the labels automatically added to Pods started via `kubectl run` based on the name of the `Deployment`.


We should now be able to access the Service from the `access` Pod.

```
# Run a Pod and try to access the `nginx` Service.
$ kubectl run --namespace=policy-demo access --rm -ti --image busybox /bin/sh
Waiting for pod policy-demo/access-472357175-y0m47 to be running, status is Pending, pod ready: false

If you don't see a command prompt, try pressing enter.

/ # wget -q --timeout=5 nginx -O -
```

However, we still cannot access the Service from a Pod without the label `run: access`:

```
# Run a Pod and try to access the `nginx` Service.
$ kubectl run --namespace=policy-demo cant-access --rm -ti --image busybox /bin/sh
Waiting for pod policy-demo/cant-access-472357175-y0m47 to be running, status is Pending, pod ready: false

If you don't see a command prompt, try pressing enter.

/ # wget -q --timeout=5 nginx -O -
wget: download timed out
/ #
```

You can clean up the demo by deleting the demo Namespace:

```shell
kubectl delete ns policy-demo
```

This was just a simple example of the Kubernetes NetworkPolicy API and how Calico can secure your Kubernetes cluster.  For more
information on network policy in Kubernetes, see the [Kubernetes user-guide](http://kubernetes.io/docs/user-guide/networkpolicies/).

For a slightly more detailed demonstration of Policy, check out the [stars demo](stars-policy).
