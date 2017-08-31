---
title: Tiered Policy Demo
---

This guide will introduce Tiered policies and walk through a simple example
for working with the [Tiers](../../reference/calicoctl/resources/tiers) resource.
It is recommended that you are familiar with the
[Advanced Policy with Calico](../getting-started/kubernetes/tutorials/advanced-policy)
example and comfortable using [calicoct](../../reference/calicoctl) to view and manipulate
Calico network policy.

This guide requires a Kubernetes cluster configured with Calico Networking and
Tigera Essentials Toolkit. It also expects that you've [setup calicoctl](../../reference/calicoctl/setup)
to work properly with your cluster.

### Create Namespace and Enable Isolation

Create 2 Namespaces.

```
$ kubectl create ns policy-demo
$ kubectl create ns external-demo
```

Create some nginx pods in the `policy-demo` Namespace and `external-demo`,
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

Ensure that the `nginx` service on both namespaces are accessible.

```
# Run a Pod and try to access the `nginx` Service.
$ kubectl run --namespace=policy-demo access --rm -ti --image busybox /bin/sh
If you don't see a command prompt, try pressing enter.

/ # wget -q nginx.policy-demo -O -
/ # wget -q nginx.external-demo -O -
```
You should see a response from both `nginx` services.

Now enable isolation in the `policy-demo` Namespace. This will prevent
connections to pods in the `policy-demo` Namespace.

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

### Examine the Policy Resource

Let's first look at existing policies and tiers that are created when Calico
is deployed. All policies that are created so far will end up in a default tier
called `default`. You can view existing policies by running:

```
$ calicoctl get policy
NAME                                              TIER
policy-demo.default-deny                          default
```

Notice that there is now a new `TIER` column. This means that the
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

The Policy resource now includes the information about the Tier in the
`metadata` field.

You can also get a list of all current tiers that exist by running:

```
$ calicoctl get tiers
NAME      ORDER
default   1000
```

### Working with Tiers and Tiered Policies

To create a "netops" Tier, where in we'd like policies in this tier to apply
before the policies in `default` Tier, run:

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
indicate that the tier was created successfully. You can view the tiers
created by running:

```
$ calicoctl get tiers
NAME      ORDER
default   1000
netops    100
```

Launch a pod in the policy-demo namespace and test DNS connectivity to 8.8.8.8

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

To add a policy to the "netops" tier, define a policy like you normally would
and add the `tier` field to the Policy's metadata.

```
# Policy in the netops tier that will prevent DNS requests to Google DNS
# servers from pods in the policy-demo namespace.
# We use the "pass" action to give other lower ordered tiers a chance to define
# policies. If the "pass" action were not used, other policies will not be
# evaluated beyond this one.
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

Launch a pod in the policy-demo namespace and test DNS connectivity to 8.8.8.8

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

Create a `devops` tier

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

and prevent traffic from/to the `policy-demo` namespace from entering or leaving
the namespace.

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
be created under the `default` Tier. Let's enable access to the nginx Service
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

The `access-nginx` policy is created under the `default` Tier.

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

### Cleaning things up

You can clean up the demo by deleting the demo Namespace:

```
kubectl delete ns policy-demo
kubectl delete ns external-demo
```

Delete policies that are outside the default tier.

```
# Delete a single policy in a tier.
$ calicoctl delete policy policy-demo-isolation --tier devops
# Or delete the entier tier
$ calicoctl delete tier devops
$ calicoctl delete tier netops
```
