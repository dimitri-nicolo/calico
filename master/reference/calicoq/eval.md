---
title: calicoq eval
---

`calicoq eval <selector>` is used to display the endpoints that are matched by
`<selector>`.

> **Recap:** Selectors can be used in three contexts in Calico security policy
> definitions.
>
> - A selector is used in the definition of each Calico Policy object, to
>   specify the endpoints (pods) that that Policy applies to (`spec.selector`).
>
> - A selector can be used in each ingress Rule, to specify that the Rule only
>   applies to packets sent from a particular set of endpoints
>   (`source.selector`), or to packets from all endpoints except a particular
>   set (`source.notSelector`).
>
> - A selector can be used in each egress Rule, to specify that the Rule only
>   applies to packets sent to a particular set of endpoints
>   (`destination.selector`), or to packets to all endpoints except a
>   particular set (`destination.notSelector`).
>
> Kubernetes NetworkPolicy definitions are similar but less general: they do
> not support egress rules or the `notSelector` options.

Given a `<selector>` expression, therefore, it is useful to compute and display
the endpoints that match that expression.

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

-  [Policy]({{site.baseurl}}/{{page.version}}/reference/calicoctl/resources/policy) for
   more information about the Calico selector-based policy model.
