---
title: calicoq eval
redirect_from: latest/reference/calicoq/eval
---

`calicoq eval <selector>` is used to display the endpoints that are matched by
`<selector>`.

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
   more information about the {{site.prodname}} selector-based policy model.
-  [calicoq and selectors]({{site.baseurl}}/{{page.version}}/reference/calicoq/selectors) for
   a recap on how selectors are used in {{site.prodname}} policy.
