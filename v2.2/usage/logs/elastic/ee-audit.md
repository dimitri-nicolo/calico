---
title: Audit logs
canonical_url: https://docs.tigera.io/v2.2/usage/logs/elastic/ee-audit
---

### About the audit logs sent to Elasticsearch by default

Default audit logging varies per orchestrator.
- [Kubernetes](#default-audit-logs-on-kubernetes)
- [OpenShift](#default-audit-logs-on-openshift)


### Default audit logs on Kubernetes

Our Kubernetes manifests include a policy that enables audit logging on the following resources.

| Resource              | Description                                                                           |
| --------------------- | ------------------------------------------------------------------------------------- |
| `GlobalNetworkPolicy` | [Reference documentation](../../../reference/calicoctl/resources/globalnetworkpolicy) |
| `GlobalNetworkSet`    | [Reference documentation](../../../reference/calicoctl/resources/globalnetworkset)    |
| `NetworkPolicy`       | [Reference documentation](../../../reference/calicoctl/resources/networkpolicy)       |
| `Tier`                | [Reference documentation](../../../reference/calicoctl/resources/tier)                |

By default, the logs:
- Include `create`, `patch`, `update`, and `delete` events.
- Are at the `RequestResponse` level.
- Omit the `RequestReceived` stage.

You can adjust the default logging policy or enable audit logs for additional resources. For more information,
refer to [Configuring Kubernetes audit logs](k8s-audit#configuring-kubernetes-audit-logs).

### Default audit logs on OpenShift

On OpenShift, {{site.prodname}} sends no audit logs to
Elasticsearch by default. To enable audit logging on OpenShift, refer to
[Configuring OpenShift audit logging](k8s-audit#configuring-openshift-audit-logging).
