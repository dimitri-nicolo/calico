---
title: calicoq policy
---

`calicoq policy <policy-id>` shows the endpoints that are relevant to policy
`<policy-id>`, comprising:

- the endpoints for which ingress or egress traffic is policed according to the
  rules in that policy

- the endpoints that the policy's rule selectors allow or disallow as data
  sources or destinations.

It shows output that is equivalent to running `calicoq eval <selector>` for the
policy's `spec.selector` and for any `selector` or `notSelector` expressions in
the `source` or `destination` of the policy's rules.

## Options

```
-r --hide-rule-matches     Don't show the list of profiles and policies whose
                           rule selectors match <endpoint-id> as an allowed or
                           disallowed source/destination.

-s --hide-selectors        Don't show the detailed selector expressions involved
                           (that cause each displayed profile or policy to match
                           <endpoint-id>).
```

## Examples

In this example there are two groups of three endpoints each.  Policy "p1"
applies to all of the endpoints in the first group, and its rules reference
both groups as possible (allowed or denied) sources or destinations:
```
$ calicoq policy p1

Endpoints matching policy p1:
  Workload endpoint host1/k8s/g1w1/eth0
    applicable endpoints; selector "calico/k8s_ns == 'group1'"
    outbound rule 1 destination match; selector "calico/k8s_ns == 'group1'"
  Workload endpoint host1/k8s/g1w2/eth0
    applicable endpoints; selector "calico/k8s_ns == 'group1'"
    outbound rule 1 destination match; selector "calico/k8s_ns == 'group1'"
  Workload endpoint host1/k8s/g1w3/eth0
    applicable endpoints; selector "calico/k8s_ns == 'group1'"
    outbound rule 1 destination match; selector "calico/k8s_ns == 'group1'"
  Workload endpoint host1/k8s/g2w1/eth0
    inbound rule 1 source match; selector "calico/k8s_ns == 'group2'"
  Workload endpoint host1/k8s/g2w2/eth0
    inbound rule 1 source match; selector "calico/k8s_ns == 'group2'"
  Workload endpoint host1/k8s/g2w3/eth0
    inbound rule 1 source match; selector "calico/k8s_ns == 'group2'"
```

You can simplify that output by specifying `--hide-selectors`:
```
$ calicoq policy p1 --hide-selectors

Endpoints matching policy p1:
  Workload endpoint host1/k8s/g1w1/eth0
    applicable endpoints
    outbound rule 1 destination match
  Workload endpoint host1/k8s/g1w2/eth0
    applicable endpoints
    outbound rule 1 destination match
  Workload endpoint host1/k8s/g1w3/eth0
    applicable endpoints
    outbound rule 1 destination match
  Workload endpoint host1/k8s/g2w1/eth0
    inbound rule 1 source match
  Workload endpoint host1/k8s/g2w2/eth0
    inbound rule 1 source match
  Workload endpoint host1/k8s/g2w3/eth0
    inbound rule 1 source match
```

If you only wanted to know the endpoints whose ingress or egress traffic is
policed according to that policy, you could simplify the output further by
adding `--hide-rule-matches`:
```
$ calicoq policy p1 --hide-rule-matches --hide-selectors

Endpoints matching policy p1:
  Workload endpoint host1/k8s/g1w1/eth0
    applicable endpoints
  Workload endpoint host1/k8s/g1w2/eth0
    applicable endpoints
  Workload endpoint host1/k8s/g1w3/eth0
    applicable endpoints
```

## See also

-  [calicoq eval]({{site.baseurl}}/{{page.version}}/reference/calicoq/eval) for
   more detail about the underlying `calico eval` command.
-  [Policy]({{site.baseurl}}/{{page.version}}/reference/calicoctl/resources/policy) for
   more information about the Calico selector-based policy model.
