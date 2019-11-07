---
title: Tiered Policy Demo
redirect_from: latest/security/tiered-policy
canonical_url: https://docs.tigera.io/v2.3/getting-started/cnx/tiered-policy-cnx/
---

This guide will introduce tiered policies and walk through a simple example
for working with the [Tiers](/{{page.version}}/reference/resources/tier) resource.

Some use cases for using tiers and tiered policies are:

  - Allow privileged users to define a security policy that takes precedence over
    other users.
  - Translating hierarchies of physical firewalls directly into {{site.prodname}} policy.
  - Group policies together under a tier.

Specifically, the example presented here will discuss some aspects of the
first use case.

### Prerequisites

- A Kubernetes cluster configured with [{{site.prodname}}]({{site.url}}/{{page.version}}/getting-started/)
- [calicoctl installed and set up](/{{page.version}}/getting-started/calicoctl/configure/) to work properly with your cluster
- [calicoq installed and set up](/{{page.version}}/reference/calicoq/) to work with your cluster

> **Note**: Commands using calicoctl can be replaced with kubectl if the {{site.prodname}} Manager and {{site.prodname}} API server are installed.
{: .alert .alert-info}

### Create Namespace and Enable Isolation

Create 2 Namespaces.

```
kubectl create ns policy-demo
kubectl create ns external-demo
```

Create some nginx pods in the `policy-demo` and `external-demo` Namespaces,
and expose them through a service.

Start the nginx pods in the `policy-demo` namespace.
```
kubectl run --namespace=policy-demo nginx --replicas=2 --image=nginx
```

Create a service to expose your nginx pods in the `policy-demo` namespace.

```
kubectl expose --namespace=policy-demo deployment nginx --port=80
```

Start more nginx pods in a different namespace.

```
kubectl run --namespace=external-demo nginx --replicas=2 --image=nginx
```

Create a service to expose the nginx pods in a different namespace.

```
kubectl expose --namespace=external-demo deployment nginx --port=80
```

Ensure that `nginx` services in both namespaces are accessible
by running a pod within the namespace.

```
kubectl run --namespace=policy-demo access --rm -ti --image busybox /bin/sh
```

This should open a shell within the `access` pod, as shown below.

```
If you don't see a command prompt, try pressing enter.

/ #
```
{: .no-select-button}

From inside the `access` pod, attempt to access the `nginx` services with the following commands.

```
wget -q nginx.policy-demo -O -
wget -q nginx.external-demo -O -
```

You should see a response from both `nginx` services.

Now enable isolation in the `policy-demo` Namespace.

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

Now, let's check if the `nginx` service in the `policy-demo` namespace is
accessible. We can do this by running a pod and trying to access the `nginx`
service in the `policy-demo` namespace.

```
kubectl run --namespace=policy-demo access --rm -ti --image busybox /bin/sh
```

This should open a shell within the `access` pod, as shown below.

```
If you don't see a command prompt, try pressing enter.

/ #
```
{: .no-select-button}

From inside the `access` pod, use `wget` to try and access the `nginx` service.

```
wget -q --timeout=5 nginx.policy-demo -O -
```

Since you have isolated the `policy-demo` namespace, you should see
the following output.

```
wget: download timed out
```
{: .no-select-button}

### Examine the Policy Resource

Let's first look at existing policies and tiers that are created when {{site.prodname}}
is deployed. All policies that are created so far will end up in a `default` tier
called `default`. You can view existing policies by running:

```
calicoctl get networkpolicy --namespace policy-demo
```

This should return the following output.

```
NAMESPACE	NAME                                            TIER
policy-demo	knp.default.default-deny                        default
```
{: .no-select-button}

Note that there is a new `TIER` column. This means that the
`default-deny` policy in the `policy-demo` namespace exists under the
`default` tier. Also note that the name is in the form of `knp.<tier>.<policy name>`.
The `knp` prefix is added because we created a Kubernetes NetworkPolicy resource using
kubectl, whereas the NetworkPolicy created by `calicoctl` is the {{site.prodname}}-rendered equivalent
resource. This is dynamically calculated if you are using Kubernetes API Server as the
datastore, or is created by the {{site.prodname}} Kubernetes Controller if you are using etcdv3 as
the datastore. All Kubernetes-backed NetworkPolicy resources are added to the default tier.
We can get the same information in YAML format as well by running:

```
calicoctl get networkpolicy knp.default.default-deny -o yaml --namespace policy-demo
```

You should see the following output.

```
apiVersion: projectcalico.org/v3
kind: NetworkPolicy
metadata:
  creationTimestamp: 2018-01-12T00:49:33Z
  name: knp.default.default-deny
  namespace: policy-demo
  resourceVersion: /24516
  uid: 748a6a2b-f732-11e7-b78a-42010a800036
spec:
  order: 1000
  selector: projectcalico.org/orchestrator == 'k8s'
  tier: default
  types:
  - Ingress
```
{: .no-select-button}

Note that the policy resource includes the information about the tier in the
prefix to the `metadata.name` field as well as the `spec` field. Also, note the
use of the special label `projectcalico.org/orchestrator` in the policy selector.
This label and a namespace label (which is not shown in the output) is automatically
added to endpoints and the values appropriately populated. The namespace
selector is not shown because its usage is implied from the namespaced network policy
resources.

You can use `calicoq` command to query the workload endpoints that the
`default-deny` Kubernetes NetworkPolicy the `policy-demo` namespace applies to.

```
calicoq policy policy-demo/knp.default.default-deny
```

This should return the following output.

```
Policy "policy-demo/knp.default.default-deny" applies to these endpoints:
  Workload endpoint host1/k8s/policy-demo.access-79f4758b79-6q8qs/eth0; selector "(projectcalico.org/orchestrator == 'k8s') && projectcalico.org/namespace == 'policy-demo'"
  Workload endpoint host1/k8s/policy-demo.nginx-7c87f569d-vvjtb/eth0; selector "(projectcalico.org/orchestrator == 'k8s') && projectcalico.org/namespace == 'policy-demo'"
  Workload endpoint host1/k8s/policy-demo.nginx-7c87f569d-zcgrh/eth0; selector "(projectcalico.org/orchestrator == 'k8s') && projectcalico.org/namespace == 'policy-demo'"

Endpoints matching Policy "policy-demo/knp.default.default-deny" rules:
```
{: .no-select-button}

You can also get a list of all current tiers that exist by running:

```
calicoctl get tiers
```

Your output should look like the following.

```
NAME      ORDER
allow-cnx 100
default   <nil>
```
{: .no-select-button}

### Working with Tiers and Tiered Policies

Let's create a `netops` tier. We'd like policies in this tier to apply
before the policies in `default` tier. Let's create a new `netops` tier
and give it a higher order of precedence than the `default` tier.

```
calicoctl create -f - <<EOF
apiVersion: projectcalico.org/v3
kind: Tier
metadata:
  name: netops
spec:
  order: 100
EOF
```

You should see the message `Successfully created 1 'Tier' resource(s)` to
indicate that the tier was created successfully. You can view your current
tiers as follows.

```
calicoctl get tiers
```

You should now see:

```
NAME      ORDER
allow-cnx 100
netops    100
default   <nil>
```
{: .no-select-button}

Notice that the order value of the netops tier is lower than the value of the
`default` tier since `<nil>` is treated as "infinite". Lower order values have
a higher precedence.

> **Note**: Read more about order values in the
> [calicoctl reference section]({{site.url}}/{{page.version}}/reference/calicoctl/).
{: .alert .alert-info}

Launch a pod in the `policy-demo` Namespace to test DNS connectivity to 8.8.8.8 .

```
kubectl run --namespace=policy-demo access --rm -ti --image busybox /bin/sh
```

This should open a shell within the `access` pod, as shown below.

```
If you don't see a command prompt, try pressing enter.

/ #
```
{: .no-select-button}

From inside the `access` pod, test DNS connectivity within your pod. This should resolve `tigera.io` using the DNS server at 8.8.8.8 .

```
nslookup tigera.io 8.8.8.8
```

This should return the following output.

```
Server:    8.8.8.8
Address 1: 8.8.8.8 google-public-dns-a.google.com

Name:      tigera.io
Address 1: 2620:12a:8000::2
Address 2: 2620:12a:8001::2
Address 3: 23.185.0.2
```
{: .no-select-button}

To add a GlobalNetworkPolicy to a tier, specify the name of the tier you want to add
it to, under `metadata`. The YAML sample below adds the
`netops.no-public-dns-for-policy-demo` policy to the `netops` tier. Note how we prefix
`netops.`, which is the name of the tier, followed by a `.`. This is a requirement for
tiered policy names and `calicoctl` will exit with an error message if this requirement
isn't met.

The following `GlobalNetworkPolicy` in the `netops` tier will prevent DNS requests to Google DNS
servers from pods in the policy-demo namespace. We use the "Pass" action to give the lower
ordered tiers a chance to define policies. If a policy in a tier is applied to an endpoint but
no policy in the tier acts on the traffic, it will be dropped at the end of the tier.

```
calicoctl create -f - <<EOF
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

Let's find out all endpoints that this GlobalNetworkPolicy applies to.

```
calicoq policy netops.no-public-dns-for-policy-demo
```

You should see the following output.

```
Tier "netops" Policy "netops.no-public-dns-for-policy-demo" applies to these endpoints:
  Workload endpoint host1/k8s/policy-demo.access-79f4758b79-6q8qs/eth0; selector "projectcalico.org/namespace == "policy-demo""
  Workload endpoint host1/k8s/policy-demo.nginx-7c87f569d-vvjtb/eth0; selector "projectcalico.org/namespace == "policy-demo""
  Workload endpoint host1/k8s/policy-demo.nginx-7c87f569d-zcgrh/eth0; selector "projectcalico.org/namespace == "policy-demo""

Endpoints matching Tier "netops" Policy "netops.no-public-dns-for-policy-demo" rules:
```
{: .no-select-button}

Launch a pod in the `policy-demo` Namespace for testing DNS connectivity to 8.8.8.8.

```
kubectl run --namespace=policy-demo access --rm -ti --image busybox /bin/sh
```

This should open a shell within the `access` pod, as shown below.

```
If you don't see a command prompt, try pressing enter.

/ #
```
{: .no-select-button}

From inside the `access` pod, test the DNS connectivity to 8.8.8.8.

```
nslookup tigera.io 8.8.8.8
```

No results should be returned and the query should eventually time out.
You should see the following.

```
Server:    8.8.8.8
Address 1: 8.8.8.8

nslookup: can't resolve 'tigera.io'
```
{: .no-select-button}

We can create additional tiered policies to police `Pass`-ed traffic from the
`netops.no-public-dns-for-policy-demo` GlobalNetworkPolicy.

Create a `devops` tier.

```
calicoctl create -f - <<EOF
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
calicoctl create -f - <<EOF
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

The `GlobalNetworkPolicy` created above used a selector in the ingress and egress
rules to match only selected endpoints. We can use the `calicoq` command to query
the set of endpoints that the the rule selector applies to.

```
calicoq policy devops.policy-demo-isolation
```

You should see the following output.

```
Tier "devops" Policy "devops.policy-demo-isolation" applies to these endpoints:
  Workload endpoint host1/k8s/policy-demo.access-79f4758b79-6q8qs/eth0; selector "projectcalico.org/namespace == 'policy-demo'"
  Workload endpoint host1/k8s/policy-demo.nginx-7c87f569d-vvjtb/eth0; selector "projectcalico.org/namespace == 'policy-demo'"
  Workload endpoint host1/k8s/policy-demo.nginx-7c87f569d-zcgrh/eth0; selector "projectcalico.org/namespace == 'policy-demo'"

Endpoints matching Tier "devops" Policy "devops.policy-demo-isolation" rules:
  ...
  Workload endpoint host1/k8s/external-demo.nginx-7c87f569d-d9ktq/eth0
    inbound rule 1 source match; selector "projectcalico.org/namespace != 'policy-demo'"
    outbound rule 1 destination match; selector "projectcalico.org/namespace != 'policy-demo'"
  Workload endpoint host1/k8s/external-demo.nginx-7c87f569d-nt5gp/eth0
    inbound rule 1 source match; selector "projectcalico.org/namespace != 'policy-demo'"
    outbound rule 1 destination match; selector "projectcalico.org/namespace != 'policy-demo'"
  ...

```
{: .no-select-button}

### Allow Access using a NetworkPolicy and the default Tier

You can still use Kubernetes NetworkPolicy to define policies. These policies
will always be created under the `default` tier. Let's enable access to the
nginx service using a Kubernetes NetworkPolicy.

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

The `access-nginx` policy is created under the `default` tier.
Get all the policies with the following command.

```
calicoctl get networkpolicy --namespace policy-demo
```

You should see the following output.

```
NAMESPACE	NAME                                            TIER
policy-demo	knp.default.access-nginx                        default
policy-demo	knp.default.default-deny                        default
```
{: .no-select-button}

Start another pod to test our `access-nginx` policy and leave it running temporarily.

```
kubectl run --namespace=policy-demo access --rm -ti --image busybox /bin/sh
```

This should open a shell within the `access` pod, as shown below.

```
If you don't see a command prompt, try pressing enter.

/ #
```
{: .no-select-button}

From inside the `access` pod, try to access the `nginx` service.

```
wget -q --timeout=5 nginx -O -
```

From another terminal, we should now be able to see all the endpoints our `access-nginx` policy applies
to as well as the endpoint for our `access` pod we used to test the accessibility
of the service.

```
calicoq policy policy-demo/knp.default.access-nginx
```

You should see the following output.

```
Policy "policy-demo/knp.default.access-nginx" applies to these endpoints:
  Workload endpoint host1/k8s/policy-demo.nginx-7c87f569d-vvjtb/eth0; selector "(projectcalico.org/orchestrator == 'k8s' && run == 'nginx') && projectcalico.org/namespace == 'policy-demo'"
  Workload endpoint host1/k8s/policy-demo.nginx-7c87f569d-zcgrh/eth0; selector "(projectcalico.org/orchestrator == 'k8s' && run == 'nginx') && projectcalico.org/namespace == 'policy-demo'"

Endpoints matching Policy "policy-demo/knp.default.access-nginx" rules:
  Workload endpoint host1/k8s/policy-demo.access-79f4758b79-6q8qs/eth0
    inbound rule 1 source match; selector "(projectcalico.org/namespace == 'policy-demo') && (projectcalico.org/orchestrator == 'k8s' && run == 'access')"
```
{: .no-select-button}

### Cleaning up

You can clean up the demo by deleting the demo Namespace.

```
kubectl delete ns policy-demo
kubectl delete ns external-demo
```

Delete policies that are not part of the `default` tier and
the namespaces.

```
calicoctl delete globalnetworkpolicy devops.policy-demo-isolation
calicoctl delete globalnetworkpolicy netops.no-public-dns-for-policy-demo
```

Once all of the policies within a tier are deleted, the tier
itself can be deleted.

```
calicoctl delete tier devops
calicoctl delete tier netops
```
