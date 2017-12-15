---
title: calicoq policy
---

`calicoq policy <policy-name>` shows the endpoints that are relevant to the
named policy, comprising:

- the endpoints that the policy applies to (for which ingress or egress traffic
  is policed according to the rules in that policy)

- the endpoints that match the policy's rule selectors (that are allowed or
  disallowed as data sources or destinations).

(For example, if you have a database and a webserver, you might have a policy
that says `policy selector: role==‘db’; rule: allow from role == ‘webserver’`.

Then the “policy applies to” selector is `role == ‘db’` and the “policy’s rule
selector” is `role == ‘webserver’`.)

It shows output that is equivalent to running `calicoq eval <selector>` for the
policy's `spec.selector` and for any `selector` or `notSelector` expressions in
the `source` or `destination` of the policy's rules.

## Options

```
-r --hide-rule-matches         Don't show the list of endpoints that match the
                               policy's rules as allowed or disallowed sources or
                               destinations.

-s --hide-selectors            Don't show the detailed selector expressions involved
                               (that cause the policy to apply to or match various
                               endpoints).

-o <OUTPUT> --output=<OUTPUT>  Set the output format. Should be one of yaml, json, or
                               ps. If nothing is set, defaults to ps.
```

## Examples

In this example there are two groups of three endpoints each.  Policy "p1"
applies to all of the endpoints in the first group, and its rules reference
both groups as possible (allowed or denied) sources or destinations:
```
$ calicoq policy p1

Policy "p1" applies to these endpoints:
  Workload endpoint host1/k8s/g1w1/eth0; selector "calico/k8s_ns == 'group1'"
  Workload endpoint host1/k8s/g1w2/eth0; selector "calico/k8s_ns == 'group1'"
  Workload endpoint host1/k8s/g1w3/eth0; selector "calico/k8s_ns == 'group1'"

Endpoints matching policy "p1" rules:
  Workload endpoint host1/k8s/g1w1/eth0
    outbound rule 1 destination match; selector "calico/k8s_ns == 'group1'"
  Workload endpoint host1/k8s/g1w2/eth0
    outbound rule 1 destination match; selector "calico/k8s_ns == 'group1'"
  Workload endpoint host1/k8s/g1w3/eth0
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

Policy "p1" applies to these endpoints:
  Workload endpoint host1/k8s/g1w1/eth0
  Workload endpoint host1/k8s/g1w2/eth0
  Workload endpoint host1/k8s/g1w3/eth0

Endpoints matching policy "p1" rules:
  Workload endpoint host1/k8s/g1w1/eth0
    outbound rule 1 destination match
  Workload endpoint host1/k8s/g1w2/eth0
    outbound rule 1 destination match
  Workload endpoint host1/k8s/g1w3/eth0
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

Policy "p1" applies to these endpoints:
  Workload endpoint host1/k8s/g1w1/eth0
  Workload endpoint host1/k8s/g1w2/eth0
  Workload endpoint host1/k8s/g1w3/eth0
```

## See also

-  [calicoq eval]({{site.baseurl}}/{{page.version}}/reference/calicoq/eval) for
   more detail about the related `calico eval` command.
-  [Policy]({{site.baseurl}}/{{page.version}}/reference/calicoctl/resources/policy) for
   more information about the {{site.prodname}} selector-based policy model.
