---
title: Tigera Secure EE Resource Auditing in OpenShift
canonical_url: https://docs.tigera.io/v2.3/usage/logs/elastic/ee-audit
---

This document describes how to set up audit logging for {{site.tseeprodname}}
resources in OpenShift versions 3.7+.

In OpenShift, you need to set up an [OpenShift Audit Configuration](https://docs.openshift.com/container-platform/3.10/install_config/master_node_configuration.html#master-node-config-audit-config){:target="_blank"}
using the `openshift_master_audit_config` variable in Ansible or `auditConfig` in the master config file.
At minimum you need to set the following configuration parameters:
1.  `enabled`: Boolean flag used to enable audit logging.
2. `auditFilePath`: The path to the file that audit logs are written to.
3. `policyFile`: Path to the audit policy configuration.
4. `logFormat`: Format of the audit logs. Can be either `legacy` or `json`.
For more details on configuration parameters and their values, please reference the [OpenShift Advanced Audit documentation](https://docs.openshift.com/container-platform/3.10/install_config/master_node_configuration.html#master-node-config-advanced-audit){:target="_blank"}.

Example Ansible setup:

```
openshift_master_audit_config={"enabled": true, auditFilePath: "/var/log/audit-ocp.log", "logFormat": "json", "policyFile": "/etc/origin/master/audit.yaml"
```

Example master config file setup:

```
auditConfig:
  enabled: true
  auditFilePath: "/var/log/audit-ocp.log"
  policyFile: "/etc/origin/master/audit.yaml"
  logFormat: "json"
```

## Sample auditing policy

The sample auditing policy provided below will log the following {{site.tseeprodname}} resources:

- `GlobalNetworkPolicy`
- `NetworkPolicy`
- `Tier`

It will log the following actions on those resources:

- `create`
- `update`
- `delete`

For brevity, we omit the `RequestReceived` stage logging below since we log
the requests as they are processed, rather than when they are received.

We also record the events with their response from the API server at the `RequestResponse` level
for better insights into the request life cycle. 

```yaml
apiVersion: audit.k8s.io/v1beta1
kind: Policy
rules:
- level: RequestResponse
  omitStages:
  - RequestReceived
  verbs:
  - create
  - patch
  - delete
  resources:
  - group: projectcalico.org
    resources:
    - globalnetworkpolicies
    - tiers
  - group: networking.k8s.io
    resources:
    - networkpolicies
```

> **Note**: For OpenShift version 3.7 clusters, the `apiVersion` needs to be changed from `audit.k8s.io/v1beta1` to `audit.k8s.io/v1alpha1`.
{: .alert .alert-info}


## Audit log viewing options

Use either of the following options to log audit events.

1. Logging the events to a file, which can be [mounted](https://kubernetes.io/docs/concepts/storage/volumes/){:target="_blank"} to the the host as a file or a [persistent volume](https://kubernetes.io/docs/concepts/storage/persistent-volumes/){:target="_blank"}.
1. Exporting the audit logs to a remote API such as Logstash or Fluentd. See the Kubernetes documentation for [Log Collector Examples](https://kubernetes.io/docs/tasks/debug-application-cluster/audit/#log-collector-examples){:target="_blank"} and [OpenShift documentation on Aggregating Container Logs](https://docs.openshift.com/enterprise/3.1/install_config/aggregate_logging.html){:target="_blank"}.


## See also

- [Kubernetes Audit Logging](auditing)
- [Policy Audit Mode](policy-auditing)
- [Policy Violation Alerting](policy-violations)
- [{{site.tseeprodname}} Resources]({{site.baseurl}}/{{page.version}}/reference/calicoctl/resources/) for details on the {{site.tseeprodname}} resource types.
- [Kubernetes documentation on Auditing](https://kubernetes.io/docs/tasks/debug-application-cluster/audit/){:target="_blank"} for details on configuring auditing.
