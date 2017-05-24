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

## Examples

To find all endpoints that match the `role=='frontend'` selector, i.e. that
have a `role` label with value `frontend`:
```
$ calicoq eval "role=='frontend'"
Endpoints matching selector role=='frontend':
  Host endpoint webserver1/eth0
  Host endpoint webserver2/eth0
```

To find all endpoints that have an `app` label (with any value):
```
$ calicoq eval "has(app)"
Endpoints matching selector has(app):
  Workload endpoint rack1-host1/k8s/default.frontend-5gs43/eth0
```
(In this case the answer is a Kubernetes pod.)

In case the specified selector did not match any endpoints, you would see:
```
$ calicoq eval "role=='endfront'"
Endpoints matching selector role=='endfront':
```

## See also

-  [calicoq eval]({{site.baseurl}}/{{page.version}}/reference/calicoq/eval) for
   more detail about the underlying `calico eval` command.
-  [Policy]({{site.baseurl}}/{{page.version}}/reference/calicoctl/resources/policy) for
   more information about the Calico selector-based policy model.
