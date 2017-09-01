---
title: calicoq and selectors
---

The queries that you can make with `calicoq` all involve computing the matches
between endpoints and policies or profiles, via selectors.  The use of
selectors in Calico policy is described in detail
by
[Policy]({{site.baseurl}}/{{page.version}}/reference/calicoctl/resources/policy),
but to recap:

- Selectors can be used in three contexts in Calico security policy
  definitions.

- A selector is used in the definition of each Calico Policy object, to specify
  the endpoints (pods) that that Policy applies to (`spec.selector`).

- A selector can be used in each ingress Rule, to specify that the Rule only
  matches packets sent from a particular set of endpoints (`source.selector`),
  or packets from all endpoints except a particular set (`source.notSelector`).

- A selector can be used in each egress Rule, to specify that the Rule only
  matches packets sent to a particular set of endpoints
  (`destination.selector`), or packets to all endpoints except a particular set
  (`destination.notSelector`).

Kubernetes NetworkPolicy definitions are similar but less general: they do
not support egress rules or the `notSelector` options.
