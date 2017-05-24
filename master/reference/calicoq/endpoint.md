---
title: calicoq endpoint
---

`calicoq endpoint <endpoint-id>` shows you the Calico profiles and policies
that relate to a particular endpoint `<endpoint-id>`.  It can display, in
separate lists:

- the profiles and policies that Calico uses to police traffic that is arriving
  at or departing from `<endpoint-id>`

- the profiles and policies for other endpoints that allow (or disallow, using
  selectors) `<endpoint-id>` as a source/destination for some traffic to/from
  them.

By default `calico endpoint` shows both of those lists; the second list can be
suppressed by giving the `-r` option.

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
> defaults are for all *egress* traffic from a workload to be *allowed*, and for
> all *ingress* traffic to a workload to be *denied* - and in that case (A) will
> have the intended effect, because it opens a pinhole in the generally closed
> ingress policy on the server pods; whereas (B) would have no effect and would
> not allow the intended connectivity.
>
> It's also possible that a cluster might choose to default to denying both
> ingress and egress traffic.  In that case both (A) and (B) would be needed to
> permit the intended connectivity between 'client' and 'server' pods.

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

```
$ calicoq endpoint ...

Workload endpoint k8s/calico-monitoring.alertmanager-calico-node-alertmanager-0/eth0
  Policies:
  Profiles:
    Profile ns.projectcalico.org/calico-monitoring
```

## See also

-  [Policy]({{site.baseurl}}/{{page.version}}/reference/calicoctl/resources/policy) for
   more information about the Calico policy model.
