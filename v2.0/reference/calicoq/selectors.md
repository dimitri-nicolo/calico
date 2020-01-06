---
title: calicoq and selectors
---

The queries that you can make with `calicoq` all involve computing the matches
between endpoints and policies or profiles, via selectors.

Selectors can be used in the following three contexts in {{site.tseeprodname}}
security policy definitions:

- A selector can be used in the definition of each {{site.tseeprodname}} Policy object,
  to specify the endpoints (pods) that that Policy applies to (`spec.selector`).

- A selector can be used in each ingress Rule, to specify that the Rule only
  matches packets sent from a particular set of endpoints (`source.selector`),
  or packets from all endpoints except a particular set (`source.notSelector`).

- A selector can be used in each egress Rule, to specify that the Rule only
  matches packets sent to a particular set of endpoints
  (`destination.selector`), or packets to all endpoints except a particular set
  (`destination.notSelector`).

Note: the use of selectors in {{site.tseeprodname}} policy is described in detail by
[NetworkPolicy]({{site.baseurl}}/{{page.version}}/reference/calicoctl/resources/networkpolicy) and
[GlobalNetworkPolicy]({{site.baseurl}}/{{page.version}}/reference/calicoctl/resources/globalnetworkpolicy).

Kubernetes NetworkPolicy definitions are similar but less general: they do
not support egress rules or the `notSelector` options.
