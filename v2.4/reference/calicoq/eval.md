---
title: calicoq eval
redirect_from: latest/reference/calicoq/eval
canonical_url: https://docs.tigera.io/v2.3/reference/calicoq/eval
---

`calicoq eval <selector>` is used to display the endpoints that are matched by
`<selector>`.

## Examples

To find all endpoints that match the `role=='frontend'` selector, i.e. that
have a `role` label with value `frontend`:

```
calicoq eval "role=='frontend'"
```

Sample output follows.

```
Endpoints matching selector role=='frontend':
  Host endpoint webserver1/eth0
  Host endpoint webserver2/eth0
```
{: .no-select-button}

To find all endpoints that have an `app` label (with any value):

```
calicoq eval "has(app)"
```

Sample output follows.

```
Endpoints matching selector has(app):
  Workload endpoint rack1-host1/k8s/default.frontend-5gs43/eth0
```
{: .no-select-button}

(In this case the answer is a Kubernetes pod.)

To find endpoint for a selector that does not match any endpoints:
```
calicoq eval "role=='endfront'"
```

Sample output follows.

```
Endpoints matching selector role=='endfront':
```
{: .no-select-button}

## See also

-  [NetworkPolicy]({{site.url}}/{{page.version}}/reference/calicoctl/resources/networkpolicy) and
   [GlobalNetworkPolicy]({{site.url}}/{{page.version}}/reference/calicoctl/resources/globalnetworkpolicy)
   for more information about the {{site.prodname}} policy model.
-  [calicoq and selectors]({{site.url}}/{{page.version}}/reference/calicoq/selectors) for
   a recap on how selectors are used in {{site.prodname}} policy.
