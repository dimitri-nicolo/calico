---
title: Tiered Policy Demo
---

This guide will introduce tiered policies and walk through a simple example
for working with the [Tiers](../../reference/calicoctl/resources/tier) resource.

Some use cases for using tiers and tiered policies are:

  - Allow privileged users to define security Policy that takes precedence over
    other users.
  - Translating hierarchies of physical firewalls directly into Calico Policy.
  - Group policies together under a tier.

Specifically, the example presented here will discuss some aspects of the
first use case.

### Prerequisites

- A Kubernetes cluster configured with [Calico Networking and Tigera Essentials Toolkit](../kubernetes/installation/hosted/essentials/index)
- [calicoctl installed and set up](../../reference/calicoctl/setup) to work properly with your cluster

### Create Namespace and Enable Isolation

Create 2 Namespaces.

```
$ kubectl create ns policy-demo
$ kubectl create ns external-demo
```

Create some nginx pods in the `policy-demo` and `external-demo` Namespaces,
and expose them through a Service.

```
# Run the Pods.
$ kubectl run --namespace=policy-demo nginx --replicas=2 --image=nginx

# Create the Service.
$ kubectl expose --namespace=policy-demo deployment nginx --port=80

# Run the Pods.
$ kubectl run --namespace=external-demo nginx --replicas=2 --image=nginx

# Create the Service.
$ kubectl expose --namespace=external-demo deployment nginx --port=80
```

Ensure that `nginx` services in both Namespaces are accessible.

```
# Run a Pod and try to access the `nginx` Service.
$ kubectl run --namespace=policy-demo access --rm -ti --image busybox /bin/sh
If you don't see a command prompt, try pressing enter.

/ # wget -q nginx.policy-demo -O -
/ # wget -q nginx.external-demo -O -
```
You should see a response from both `nginx` services.

Now enable isolation in the `policy-demo` Namespace.

```
$ kubectl create -f - <<EOF
kind: NetworkPolicy
apiVersion: extensions/v1beta1
metadata:
  name: default-deny
  namespace: policy-demo
spec:
  podSelector:
EOF
```

Now, let's check if the `nginx` Service in the `policy-demo` namespace is
accessible.
```
# Run a Pod and try to access the `nginx` Service in the `policy-demo` namespace.
$ kubectl run --namespace=policy-demo access --rm -ti --image busybox /bin/sh
If you don't see a command prompt, try pressing enter.

/ # wget -q --timeout=5 nginx.policy-demo -O -
wget: download timed out
# You have isolated the `policy-demo` Namespace.
```

### Examine the Policy Resource

Let's first look at existing policies and tiers that are created when Calico
is deployed. All policies that are created so far will end up in a `default` tier
called `default`. You can view existing policies by running:

```
$ calicoctl get policy
NAME                                              TIER
policy-demo.default-deny                          default
```

Notice that there is a new `TIER` column. This means that the
`policy-demo.default-deny` policy exists under the `default` tier. We can get
the same information in YAML format as well by running:

```
$ calicoctl get policy -o yaml
- apiVersion: v1
  kind: policy
  metadata:
    name: policy-demo.default-deny
    tier: default
  spec:
    egress:
    - action: allow
      destination: {}
      source: {}
    order: 1000
    selector: calico/k8s_ns == 'policy-demo'
```

Notice the Policy resource includes the information about the tier in the
`metadata` field. Also, note the use of special label `calico/k8s_ns` in the
Policy selector. This label is automatically added to endpoints and the value
populated with the pod's Kubernetes namespace. For more information about this
checkout [Advanced Policy with Calico](../kubernetes/tutorials/advanced-policy).

You can also get a list of all current tiers that exist by running:

```
$ calicoctl get tiers
NAME      ORDER
default   1000
```

### Working with Tiers and Tiered Policies

Let's create a `netops` tier. We'd like policies in this tier to apply
before the policies in `default` tier. Let's create a new `netops` tier
and give it a higher order of precedence than the `default` tier.

```
$ calicoctl create -f - <<EOF
apiVersion: v1
kind: tier
metadata:
  name: netops
spec:
  order: 100
EOF
```

You should see the message `Successfully created 1 'tier' resource(s)` to
indicate that the tier was created successfully. You can view your current
tiers as follows.

```
$ calicoctl get tiers
NAME      ORDER
default   1000
netops    100
```

Notice that the order value of the netops tier is lower than the value of the
`default` tier. Lower order values have a higher precedence.

<div class="alert alert-info" role="alert"><b>Note</b>:
Read more about order values in the <a href="../../reference/calicoctl/">calicoctl reference section.</a>
</div>

Launch a pod in the `policy-demo` Namespace and test DNS connectivity to 8.8.8.8

```
# Resolve tigera.io using 8.8.8.8 DNS server.
$ kubectl run --namespace=policy-demo access --rm -ti --image busybox /bin/sh
If you don't see a command prompt, try pressing enter.
/ # nslookup tigera.io 8.8.8.8
Server:    8.8.8.8
Address 1: 8.8.8.8 google-public-dns-a.google.com

Name:      tigera.io
Address 1: 2620:12a:8000::2
Address 2: 2620:12a:8001::2
Address 3: 23.185.0.2
```

To add a Policy to a tier, specify the name of the tier you want to add it to,
under `metadata`. The YAML sample below adds the `no-public-dns-for-policy-demo`
policy to the `netops` tier.

```
# Policy in the netops tier that will prevent DNS requests to Google DNS
# servers from pods in the policy-demo Namespace.
# We use the "pass" action to give other lower ordered tiers a chance to define
# policies. If a policy in a tier is applied to an endpoint but no policy in
# the tier acts on the traffic, it will be dropped at the end of tier.
$ calicoctl create -f - <<EOF
- apiVersion: v1
  kind: policy
  metadata:
    name: no-public-dns-for-policy-demo
    tier: netops
  spec:
    order: 100
    ingress:
    # Let lower order tiers define traffic for ingress
    # traffic
    - action: pass
      destination: {}
      source: {}
    egress:
    # Drop traffic to google-public-dns-a.google.com
    - action: deny
      destination:
        net: 8.8.8.8/32
    # Drop traffic to google-public-dns-b.google.com
    - action: deny
      destination:
        net: 8.8.4.4/32
    # Pass other traffic to next tier
    - action: pass
    selector: calico/k8s_ns == 'policy-demo'
EOF
```

Launch a pod in the `policy-demo` Namespace and test DNS connectivity to 8.8.8.8

```
# Resolve tigera.io using 8.8.8.8 DNS server.
$ kubectl run --namespace=policy-demo access --rm -ti --image busybox /bin/sh
If you don't see a command prompt, try pressing enter.
/ # nslookup tigera.io 8.8.8.8
Server:    8.8.8.8
Address 1: 8.8.8.8

nslookup: can't resolve 'tigera.io'
# No results returned and query should eventually timeout.
```

We can create additional tiered policies to police `pass`-ed traffic from the
`no-public-dns-for-policy-demo` Policy.

Create a `devops` tier.

```
$ calicoctl create -f - <<EOF
apiVersion: v1
kind: tier
metadata:
  name: devops
spec:
  order: 500
EOF
```

Then create a Policy in the `devops` tier that will prevent traffic from/to the.
`policy-demo` Namespace from entering or leaving this Namespace.

```
$ calicoctl create -f - <<EOF
- apiVersion: v1
  kind: policy
  metadata:
    name: policy-demo-isolation
    tier: devops
  spec:
    order: 200
    egress:
    # Deny traffic from leaving the namespace
    - action: deny
      destination:
        selector: calico/k8s_ns != 'policy-demo'
    - action: pass
    ingress:
    # Deny traffic from entering the namespace
    - action: deny
      source:
        selector: calico/k8s_ns != 'policy-demo'
    - action: pass
    selector: calico/k8s_ns == 'policy-demo'
EOF
```

### Allow Access using a NetworkPolicy and the default Tier

You can still use NetworkPolicy to define policies. These policies will always
be created under the `default` tier. Let's enable access to the nginx Service
using a NetworkPolicy.

Create a network policy `access-nginx` with the following contents:

```
$ kubectl create -f - <<EOF
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

The `access-nginx` policy is created under the `default` tier.

```
# Get all policies that belong in the default tier.
# Note the usage of the "--tier" option.
$ calicoctl get policies --tier default
NAME                                              TIER
policy-demo.access-nginx                          default
policy-demo.default-deny                          default
```

We should now be able to access the Service from the access Pod.

```
# Run a Pod and try to access the `nginx` Service.
$ kubectl run --namespace=policy-demo access --rm -ti --image busybox /bin/sh
If you don't see a command prompt, try pressing enter.

/ # wget -q --timeout=5 nginx -O -
```

### Cleaning up

You can clean up the demo by deleting the demo Namespace:

```
kubectl delete ns policy-demo
kubectl delete ns external-demo
```

Delete policies that are not part of the `default` tier.

```
# Delete a single policy in a tier.
$ calicoctl delete policy policy-demo-isolation --tier devops
# Or delete the entier tier
$ calicoctl delete tier devops
$ calicoctl delete tier netops
```
