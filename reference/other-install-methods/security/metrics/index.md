---
title: Overview
canonical_url: https://docs.tigera.io/v2.3/usage/metrics/
---

{{site.prodname}} uses a Prometheus operator to deploy a Prometheus and Alertmanager instance.

By default, Prometheus scrapes the following policy metrics: `calico_denied_packets`, `calico_denied_bytes`,
`cnx_policy_rule_bytes`, and `cnx_policy_rule_connections`. For more information about the policy metrics and
some sample queries, refer to [Policy metrics in Prometheus](/{{page.version}}/reference/other-install-methods/security/metrics/).

You can also:
- [Modify the default policy metrics](/{{page.version}}/reference/other-install-methods/security/configuration/prometheus).
- [Set up alerts or different storage](/{{page.version}}/reference/other-install-methods/security/configuration/alertmanager).

In addition to policy metrics, you can enable whitebox metrics. Refer to the [Felix reference documentation](/{{page.version}}/reference/felix/prometheus)
for more information.
