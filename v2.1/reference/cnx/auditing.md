---
title: Tigera Secure EE Resource Auditing with Kubernetes
canonical_url: https://docs.tigera.io/v2.3/usage/logs/elastic/ee-audit
---

This document describes how to set up audit logging for {{site.tseeprodname}}
resources. See [OpenShift Audit Logging](openshift-auditing){:target="_blank"} if you are running OpenShift.

Kubernetes provides a rich set of auditing features to audit log resources access
activities in chronological order. See [Kubernetes audit logging documentation](https://kubernetes.io/docs/tasks/debug-application-cluster/audit/){:target="_blank"} for more details.

Audit logging needs to be setup at kubernetes kube-apiserver level. 
At minimum you need the `--audit-log-path` and `--audit-policy-file` kube-apiserver flags
specified, with the former one being the path to a file to output the audit logs and the
later one being the audit policy configuration.

> **Note**: Kubernetes audit logging won't log anything by default if audit policy file
> i.e. `--audit-policy-file` is not provided or is empty.
{: .alert .alert-info}

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


## Audit log viewing options

Use either of the following options to log audit events.

1. Logging the events to a file, which can be [mounted](https://kubernetes.io/docs/concepts/storage/volumes/){:target="_blank"} to the the host as a file or a [persistent volume](https://kubernetes.io/docs/concepts/storage/persistent-volumes/){:target="_blank"}.
1. Exporting the audit logs to a remote API such as Logstash or Fluentd. See the Kubernetes documentation for [Log Collector Examples](https://kubernetes.io/docs/tasks/debug-application-cluster/audit/#log-collector-examples){:target="_blank"}.



## See also

- [OpenShift Audit Logging](openshift-auditing)
- [Policy Audit Mode](policy-auditing)
- [Policy Violation Alerting](policy-violations)
- [{{site.tseeprodname}} Resources]({{site.baseurl}}/{{page.version}}/reference/calicoctl/resources/) for details on the {{site.tseeprodname}} resource types.
- [Kubernetes documentation on Auditing](https://kubernetes.io/docs/tasks/debug-application-cluster/audit/){:target="_blank"} for details on configuring auditing.
