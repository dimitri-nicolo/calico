---
title: Tiered Policy Demo
---

This guide will introduce tiered policies and walk through a simple example
for working with the [Tiers]({{site.baseurl}}/{{page.version}}/reference/calicoctl/resources/tier) resource.

Some use cases for using tiers and tiered policies are:

  - Allow privileged users to define security Policy that takes precedence over
    other users.
  - Translating hierarchies of physical firewalls directly into {{site.prodname}} Policy.
  - Group policies together under a tier.

Specifically, the example presented here will discuss some aspects of the
first use case.

### Prerequisites

- A Kubernetes cluster configured with [{{site.prodname}}](../kubernetes/installation/hosted/essentials/index)
- [calicoctl installed and set up]({{site.baseurl}}/{{page.version}}/reference/calicoctl/setup/) to work properly with your cluster

### Create Namespace and Enable Isolation

Create 2 Namespaces.

```
$ kubectl create ns policy-demo
$ kubectl create ns external-demo
```

Create some nginx pods in the `policy-demo` and `external-demo` Namespaces,
and expose them through a service.

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
    matchLabels: {}
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

Let's first look at existing policies and tiers that are created when {{site.prodname}}
is deployed. All policies that are created so far will end up in a `default` tier
called `default`. You can view existing policies by running:

```
$ calicoctl get networkpolicy --namespace policy-demo
NAMESPACE	NAME                                              TIER
policy-demo	knp.default.default-deny                          default
```

Notice that there is a new `TIER` column. This means that the
`default-deny` policy in the `policy-demo` namespace exists under the
`default` tier. We can get the same information in YAML format as well
by running:

```
$ calicoctl get networkpolicy -o yaml --namespace policy-demo
- apiVersion: projectcalico.org/v3
  kind: NetworkPolicy
  metadata:
    name: knp.default.default-deny
    namespace: policy-demo
  spec:
    tier: default
    egress:
    - action: Allow
      destination: {}
      source: {}
    order: 1000
    selector: projectcalico.org/namespace == 'policy-demo'
```

Notice the Policy resource includes the information about the tier in the
`metadata` field. Also, note the use of special label `projectcalico.org/namespace` in the
Policy selector. This label is automatically added to endpoints and the value
populated with the pod's Kubernetes namespace.

You can also get a list of all current tiers that exist by running:

```
$ calicoctl get tiers
NAME      ORDER
default   <nil>
```

### Working with Tiers and Tiered Policies

Let's create a `netops` tier. We'd like policies in this tier to apply
before the policies in `default` tier. Let's create a new `netops` tier
and give it a higher order of precedence than the `default` tier.

```
$ calicoctl create -f - <<EOF
apiVersion: projectcalico.org/v3
kind: Tier
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

> **Note**: Read more about order values in the
> [calicoctl reference section]({{site.baseurl}}/{{page.version}}/reference/calicoctl/).
{: .alert .alert-info}

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

To add a GlobalNetworkPolicy to a tier, specify the name of the tier you want to add
it to, under `metadata`. The YAML sample below adds the
`netops.no-public-dns-for-policy-demo` policy to the `netops` tier. Note how we prefix
`netops.`, which is the name of the tier, followed by a `.`. This is a requirement for
tiered policy names and calicoctl will exit with an error message if this requirement
isn't met.

```
# GlobalNetworkPolicyPolicy in the netops tier that will prevent DNS requests to
# Google DNS servers from pods in the policy-demo namespace.
# We use the "Pass" action to give other lower ordered tiers a chance to define
# policies. If a policy in a tier is applied to an endpoint but no policy in
# the tier acts on the traffic, it will be dropped at the end of tier.
$ calicoctl create -f - <<EOF
- apiVersion: projectcalico.org/v3
  kind: GlobalNetworkPolicy
  metadata:
    name: netops.no-public-dns-for-policy-demo
  spec:
    tier: netops
    order: 100
    ingress:
    # Let lower order tiers define traffic for ingress
    # traffic
    - action: Pass
      destination: {}
      source: {}
    egress:
    # Drop traffic to google-public-dns-a.google.com
    - action: Deny
      destination:
        nets: ["8.8.8.8/32"]
    # Drop traffic to google-public-dns-b.google.com
    - action: Deny
      destination:
        nets: ["8.8.4.4/32"]
    # Explicitly allow other outgoing DNS traffic
    - action: Allow
      protocol: UDP
      destination:
        ports: [53]
    # Pass other traffic to next tier
    - action: Pass
    selector: projectcalico.org/namespace == 'policy-demo'
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

We can create additional tiered policies to police `Pass`-ed traffic from the
`netops.no-public-dns-for-policy-demo` GlobalNetworkPolicy.

Create a `devops` tier.

```
$ calicoctl create -f - <<EOF
apiVersion: projectcalico.org/v3
kind: Tier
metadata:
  name: devops
spec:
  order: 500
EOF
```

Then create a GlobalNetworkPolicy in the `devops` tier that will prevent traffic from/to the.
`policy-demo` Namespace from entering or leaving this Namespace.

```
$ calicoctl create -f - <<EOF
- apiVersion: projectcalico.org/v3
  kind: GlobalNetworkPolicy
  metadata:
    name: devops.policy-demo-isolation
  spec:
    tier: devops
    order: 200
    egress:
    # Deny traffic from leaving the namespace
    - action: Deny
      destination:
        selector: projectcalico.org/namespace != 'policy-demo'
    - action: Pass
    ingress:
    # Deny traffic from entering the namespace
    - action: Deny
      source:
        selector: projectcalico.org/namespace != 'policy-demo'
    - action: Pass
    selector: projectcalico.org/namespace == 'policy-demo'
EOF
```

### Allow Access using a NetworkPolicy and the default Tier

You can still use Kubernetes NetworkPolicy to define policies. These policies
will always be created under the `default` tier. Let's enable access to the
nginx service using a Kubernetes NetworkPolicy.

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
# Get all NetworkPolicies
$ calicoctl get networkpolicy --namespace policy-demo
NAMESPACE	NAME                                              TIER
policy-demo	knp.default.access-nginx                          default
policy-demo	knp.default.default-deny                          default
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
$ calicoctl delete policy devops.policy-demo-isolation
# Or delete the entier tier
$ calicoctl delete tier devops
$ calicoctl delete tier netops
```
