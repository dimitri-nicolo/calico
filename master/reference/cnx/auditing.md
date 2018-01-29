---
title: CNX Resource Auditing with Kubernetes
---

This document describes audit logging setup for {{site.prodname}}
resources and how to set it up.

Kubernetes provides a rich set of auditing features to audit log resources access
activities in chronological order. See [Kubernetes audit logging documentation](https://kubernetes.io/docs/tasks/debug-application-cluster/audit/) for more details.

Audit logging needs to be setup at kubernetes kube-apiserver level. 
At minimum you need the `--audit-log-path` and `--audit-policy-file` [kube-apiserver flags](https://kubernetes.io/docs/reference/generated/kube-apiserver/)
specified, with the former one being the path to a file to output the audit logs and the
later one being the audit policy configuration.

> **Note**: Kubernetes audit logging won't log anything by default if audit policy file
> i.e. `--audit-policy-file` is not provided or is empty.

## Sample Auditing Policy

The sample auditing policy provided below will log the following CNX resources:

- `GlobalNetworkPolicy`
- `NetworkPolicy`
- `Tier`

It will log the following actions on those resources:

- `create`
- `update`
- `delete`

For brevity, we are omitting `RequestReceived` stage logging here since we are logging
the request when they are processed, rather than when they are received.

We are also recording the events with their response from the API server at `RequestResponse` level
for better insights into the request life-cycle. 

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

Kubernetes provides a few options to log the audit events.

1. Logging the events to a file, which can be [mounted](https://kubernetes.io/docs/concepts/storage/volumes/) to the the host as a file or a [persistent volume](https://kubernetes.io/docs/concepts/storage/persistent-volumes/)
1. Exporting the audit logs to a remote API such as Logstach or Fluentd. See [kubernetes docs on sample config](https://kubernetes.io/docs/tasks/debug-application-cluster/audit/#log-collector-examples)



## See also

- [Policy Audit Mode](policy-auditing)
- [Policy Violation Alerting](policy-violations)
- [{{site.prodname}} Resources]({{site.baseurl}}/{{page.version}}/reference/calicoctl/resources/) for details on the {{site.prodname}} resource types.
- [Kubernetes Auditing](https://kubernetes.io/docs/tasks/debug-application-cluster/audit/) for details on configuring auditing.
- [Kubernetes API Server flags](https://kubernetes.io/docs/reference/generated/kube-apiserver/) for audit logging related flags.