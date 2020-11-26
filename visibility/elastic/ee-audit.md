---
title: Audit logs
description: Enable and change default audit logs sent to Elasticsearch. 
canonical_url: /visibility/elastic/ee-audit
---

### Default audit logs sent to Elasticsearch

Our manifests include a policy that enables audit logging on the following {{site.prodname}} resources.

| Resource              | Description                                                                           |
| --------------------- | ------------------------------------------------------------------------------------- |
| `GlobalNetworkPolicy` | [Reference documentation]({{site.baseurl}}/reference/resources/globalnetworkpolicy) |
| `GlobalNetworkSet`    | [Reference documentation]({{site.baseurl}}/reference/resources/globalnetworkset)    |
| `NetworkPolicy`       | [Reference documentation]({{site.baseurl}}/reference/resources/networkpolicy)       |
| `Tier`                | [Reference documentation]({{site.baseurl}}/reference/resources/tier)                |
| `HostEndpoint`        | [Reference documentation]({{site.baseurl}}/reference/resources/hostendpoint)        |

By default, the logs:
- Include `create`, `patch`, `update`, and `delete` events.
- Are at the `RequestResponse` level.
- Omit the `RequestReceived` stage.

The logs can be viewed in Elasticsearch or Kibana.

### Changing audit policy for {{site.prodname}} resources

The audit policy for {{site.prodname}} resources (above) is contained in a ConfigMap.  To update this policy,
follow these steps.

1. Edit the `audit-policy-ee` ConfigMap in `kube-system`. For clusters installed with `helm`:

   ```
   kubectl edit configmap audit-policy-ee -n kube-system
   ```

   or for clusters installed with the `operator`

   ```
   kubectl edit configmap tigera-audit-policy -n tigera-system
   ```

1. Restart the cnx-apiserver pod to pick up the new policy (note that this will cause API downtime).

   ```
   kubectl delete pod -l k8s-app=cnx-apiserver -n kube-system
   ```

### Enabling auditing for other resources

As part of setting up your cluster we recommend you enable auditing for
Kubernetes/OpenShift resources as well.
The following sections describe setting up auditing on the Kubernetes/OpenShift API Server for the following
resources that are directly involved in network policy evaluation and are required for {{site.prodname}} Compliance
Analytics, similar to the {{site.prodname}} resources above:

- `Pod`
- `Namespace`
- `ServiceAccount`
- `NetworkPolicy` (Kubernetes/OpenShift)
- `Endpoints`

You may wish to audit resources beyond those involved in network policy.  Consult the [Kubernetes docs](https://kubernetes.io/docs/tasks/debug-application-cluster/audit/#audit-policy){:target="_blank"}, or
look at [this function](https://github.com/kubernetes/kubernetes/blob/cc67ccfd7f4f0bc96d7f1c8e5fe8577821757d03/cluster/gce/gci/configure-helper.sh#L752){:target="_blank"}
which generates the GKE audit policy for inspiration.  Here is the sample policy which audits changes to `Pod`,
`Namespace`, `ServiceAccount`, `Endpoints` and `NetworkPolicy` resources.

```yaml
apiVersion: audit.k8s.io/v1beta1
kind: Policy
omitStages:
- RequestReceived
rules:
  - level: RequestResponse
    verbs:
      - create
      - patch
      - update
      - delete
    resources:
    - group: networking.k8s.io
      resources: ["networkpolicies"]
    - group: extensions
      resources: ["networkpolicies"]
    - group: ""
      resources: ["pods", "namespaces", "serviceaccounts", "endpoints"]
```

#### Kubernetes

Audit logging needs to be set up at the kube-apiserver level.
At minimum you need the `--audit-log-path` and `--audit-policy-file` [kube-apiserver flags](https://kubernetes.io/docs/reference/command-line-tools-reference/kube-apiserver/){:target="_blank"}
specified, with the former one being the path to a file to output the audit logs and the
later one being the audit policy configuration (above).

{{site.prodname}} configures fluentd to read audit logs from `/var/log/calico/audit/kube-audit.log`, so to take
advantage of the {{site.prodname}} Elasticsearch and Kibana dashboards, send your audit logs to that file.

> **Note**: Kubernetes audit logging won't log anything by default if an audit policy file
> is not provided, i.e., `--audit-policy-file`.
{: .alert .alert-info}

1. Configure the Kubernetes API server with the following arguments (see the
   [Kubernetes audit logging documentation](https://kubernetes.io/docs/tasks/debug-application-cluster/audit/){:target="_blank"} for more details).

   ```
   --audit-log-path=/var/log/calico/audit/kube-audit.log
   --audit-policy-file=<path to file containing audit policy above>
   ```

1. Distribute the audit policy file to all master nodes, ensuring that it is available to
   Kubernetes API server (e.g. by volume mounting it into the pods).

1. Restart the Kubernetes API server (the command for this depends on how you installed Kubernetes).

### OpenShift

> **Note**: Audit logging is `not supported` on Openshift v4.x
{: .alert .alert-info}
