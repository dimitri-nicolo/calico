---
title: Audit logs
canonical_url: https://docs.tigera.io/v2.3/usage/logs/elastic/ee-audit
---

### Default audit logs sent to Elasticsearch

Our manifests include a policy that enables audit logging on the following {{site.tseeprodname}} resources.

| Resource              | Description                                                                           |
| --------------------- | ------------------------------------------------------------------------------------- |
| `GlobalNetworkPolicy` | [Reference documentation](../../../reference/calicoctl/resources/globalnetworkpolicy) |
| `GlobalNetworkSet`    | [Reference documentation](../../../reference/calicoctl/resources/globalnetworkset)    |
| `NetworkPolicy`       | [Reference documentation](../../../reference/calicoctl/resources/networkpolicy)       |
| `Tier`                | [Reference documentation](../../../reference/calicoctl/resources/tier)                |
| `HostEndpoint`        | [Reference documentation](../../../reference/calicoctl/resources/hostendpoint)        |

By default, the logs:
- Include `create`, `patch`, `update`, and `delete` events.
- Are at the `RequestResponse` level.
- Omit the `RequestReceived` stage.

The logs can be [viewed in Elasticsearch or Kibana](view)

### Changing audit policy for {{site.tseeprodname}} resources

The audit policy for {{site.tseeprodname}} resources (above) is contained in a ConfigMap.  To update this policy,
follow these steps.

1. Edit the `audit-policy-ee` ConfigMap in `kube-system` (contained in `cnx-api.yaml`).

   ```
   vi cnx-api.yaml
   kubectl apply -f cnx-api.yaml
   ```

   Or

   ```
   kubectl edit configmap audit-policy-ee -n kube-system
   ```

1. Restart the cnx-apiserver pod to pick up the new policy (note that this will cause API downtime).

   ```
   kubectl delete pod -l k8s-app=cnx-apiserver -n kube-system
   ```

### Enabling auditing for other resources

As part of setting up your cluster we recommend you enable auditing for
Kubernetes/OpenShift resources as well.
The following sections describe setting up auditing on the Kubernetes/OpenShift API Server for the following
resources that are directly involved in network policy evaluation and are required for {{site.tseeprodname}} Compliance 
Analytics, similar to the {{site.tseeprodname}} resources above:

- `Pod`
- `Namespace`
- `ServiceAccount`
- `NetworkPolicy` (Kubernetes/OpenShift)
- `Endpoints`

You may wish to audit resources beyond those involved in network policy.  Consult the [Kubernetes docs](https://kubernetes.io/docs/tasks/debug-application-cluster/audit/#audit-policy), or
look at [this function](https://github.com/kubernetes/kubernetes/blob/cc67ccfd7f4f0bc96d7f1c8e5fe8577821757d03/cluster/gce/gci/configure-helper.sh#L752)
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

{{site.tseeprodname}} configures fluentd to read audit logs from `/var/log/calico/audit/kube-audit.log`, so to take
advantage of the {{site.tseeprodname}} Elasticsearch and Kibana dashboards, send your audit logs to that file.

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

Set up an [OpenShift Audit Configuration](https://docs.openshift.com/container-platform/3.11/install_config/master_node_configuration.html#master-node-config-audit-config){:target="_blank"}
using the `openshift_master_audit_config` variable in Ansible or `auditConfig` in the master configuration file (typically found in /etc/origin/master/master-config.yaml).
At minimum you need to set the following configuration parameters:
- `enabled`: Boolean flag used to enable audit logging.
- `auditFilePath`: The path to the file that audit logs are written to.
- `policyFile`: Path to the audit policy configuration.
- `logFormat`: Format of the audit logs. Should be `json` if using sending the logs to the {{site.tseeprodname}} Elasticsearch.
For more details on configuration parameters and their values, please reference the [OpenShift Advanced Audit documentation](https://docs.openshift.com/container-platform/3.11/install_config/master_node_configuration.html#master-node-config-advanced-audit){:target="_blank"}.

1. Either set the appropriate ansible variable in your inventory file (adjust paths as necessary):

   ```
   openshift_master_audit_config={"enabled": true, "auditFilePath": "/var/lib/origin/audit/kube-audit.log", "logFormat": "json", "policyFile": "/etc/origin/master/audit.yaml"}
   ```

   Or add the following to the master configuration file (adjust paths as necessary).

   ```
   auditConfig:
     enabled: true
     auditFilePath: "/var/lib/origin/audit/kube-audit.log"
     policyFile: "/etc/origin/master/audit.yaml"
     logFormat: "json"
   ```

1. Distribute the audit policy file to the appropriate location on all master nodes (`/etc/origin/master/audit.yaml` in the above example).

1. Restart the OpenShift API server. If you set the audit log configuration in an ansible variable in your inventory file,
   rerun the ansible provisioner. If you set the audit log configuration in the master configuration file, then restart the
   API server pod.
