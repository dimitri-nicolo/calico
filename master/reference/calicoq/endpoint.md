---
title: calicoq endpoint
---

`calicoq endpoint <substring>` shows you the Calico profiles and policies that
relate to endpoints whose full name includes `<substring>`.  It can display, in
separate lists:

- the profiles and policies that Calico uses to police traffic that is arriving
  at or departing from those endpoints

- the profiles and policies for other endpoints that allow (or disallow, using
  selectors) one of those endpoints as a source/destination for some traffic
  to/from them.

> **Note:** You may be wondering what the difference is between those.  For
> example, what is the difference between (A):
>
> ```
> allow 'role=server' pods to receive data on port 443 from 'role=client' pods
> ```
>
> and (B):
>
> ```
> allow 'role=client' pods to send data on port 443 to 'role=server' pods
> ```
>
> The answer is that it depends on the cluster defaults.  Most commonly those
> defaults are for all *egress* traffic from an endpoint to be *allowed*, and for
> all *ingress* traffic to an endpoint to be *denied* - and in that case (A) will
> have the intended effect, because it opens a pinhole in the generally closed
> ingress policy on the server pods; whereas (B) would have no effect and would
> not allow the intended connectivity.
>
> It's also possible that a cluster might choose to default to denying both
> ingress and egress traffic.  In that case both (A) and (B) would be needed to
> permit the intended connectivity between 'client' and 'server' pods.

By default `calico endpoint` shows both of those lists; the second list can be
suppressed by giving the `-r` option.

`<substring>` can be any substring of an endpoint's full name, which is formed
as `<host-name>/<orchestrator-id>/<workload-id>/<endpoint-id>`.

## Options

```
-r --hide-rule-matches     Don't show the list of profiles and policies whose
                           rule selectors match <endpoint-id> as an allowed or
                           disallowed source/destination.

-s --hide-selectors        Don't show the detailed selector expressions involved
                           (that cause each displayed profile or policy to match
                           <endpoint-id>).
```

## Example

Here is an example with three workloads in a group, named with a prefix that
specifies the group; so `calicoq endpoint` with that prefix returns information
about all three endpoints.
```
$ calicoq endpoint g1w

Policies that match each endpoint:

Workload endpoint k8s/g1w1/eth0
  Policies:
    Policy "p1" (order 500; selector "calico/k8s_ns == 'group1'")
  Profiles:
    Profile group1

Workload endpoint k8s/g1w2/eth0
  Policies:
    Policy "p1" (order 500; selector "calico/k8s_ns == 'group1'")
  Profiles:
    Profile group1

Workload endpoint k8s/g1w3/eth0
  Policies:
    Policy "p1" (order 500; selector "calico/k8s_ns == 'group1'")
  Profiles:
    Profile group1
```

Here is an example of a workload to which both normal and untracked policy
applies.  The untracked policy is listed first because Calico enforces
untracked policies before normal ones.
```
$ calicoq endpoint tigera-lwr-kubetest-02/

Policies that match each endpoint:

Workload endpoint k8s/advanced-policy-demo.nginx-2371676037-bk6v2/eth0
  Policies:
    Policy "donottrack" (order 500; selector "calico/k8s_ns == 'advanced-policy-demo'") [untracked]
    Policy "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ" (order 400; selector "calico/k8s_ns == 'advanced-policy-demo'")
  Profiles:
    Profile k8s_ns.advanced-policy-demo
```

## See also

-  [Policy]({{site.baseurl}}/{{page.version}}/reference/calicoctl/resources/policy) for
   more information about the Calico policy model.
