---
title: Configuring audit logs
---

## About configuring audit logs

If you would like to modify the [default audit logging](ee-audit),
refer to the section that corresponds to your orchestrator.
- [Kubernetes](#configuring-kubernetes-audit-logs)
- [OpenShift](#configuring-openshift-audit-logging)

## Configuring Kubernetes audit logs

Audit logging needs to be set up at the kube-apiserver level.
At minimum you need the `--audit-log-path` and `--audit-policy-file` [kube-apiserver flags](https://kubernetes.io/docs/reference/command-line-tools-reference/kube-apiserver/){:target="_blank"}
specified, with the former one being the path to a file to output the audit logs and the
later one being the audit policy configuration.

> **Note**: Kubernetes audit logging won't log anything by default if an audit policy file,
> is not provided, i.e., `--audit-policy-file`.
{: .alert .alert-info}

The sample auditing policy provided below will log the following Kubernetes resources that are
directly involved in {{site.prodname}} policy evaluation:

- `Pod`
- `Namespace`
- `ServiceAccount`

It will log the following actions on those resources:

- `create`
- `update`
- `patch`
- `delete`

For brevity, we omit the `RequestReceived` stage logging below since we log
the requests as they are processed, rather than when they are received.

We also record the events with their response from the API server at the `RequestResponse` level
for better insights into the request life cycle.

Configure the Kubernetes API Server with the following policy file by following the [Kubernetes audit logging documentation](https://kubernetes.io/docs/tasks/debug-application-cluster/audit/){:target="_blank"}.

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
  - update
  - delete
  resources:
  - group: ""
    resources:
    - pods
    - namespaces
    - serviceaccounts
  - group: networking.k8s.io
    resources:
    - networkpolicies
  - group: extensions
    resources:
    - networkpolicies
```

## Configuring OpenShift audit logging

### About configuring OpenShift audit logging

{{site.prodname}} on OpenShift does not send any audit logs to ElasticSearch by default. To enable
audit logs on OpenShift, complete the following.

### Configure auditing on the master

Set up an [OpenShift Audit Configuration](https://docs.openshift.com/container-platform/latest/install_config/master_node_configuration.html#master-node-config-audit-config){:target="_blank"}
using the `openshift_master_audit_config` variable in Ansible or `auditConfig` in the master configuration file.
At minimum you need to set the following configuration parameters:
-  `enabled`: Boolean flag used to enable audit logging.
- `auditFilePath`: The path to the file that audit logs are written to.
- `policyFile`: Path to the audit policy configuration.
- `logFormat`: Format of the audit logs. Can be either `legacy` or `json`.
For more details on configuration parameters and their values, please reference the [OpenShift Advanced Audit documentation](https://docs.openshift.com/container-platform/latest/install_config/master_node_configuration.html#master-node-config-advanced-audit){:target="_blank"}.

Example Ansible setup:

```
openshift_master_audit_config={"enabled": true, auditFilePath: "/var/log/audit-ocp.log", "logFormat": "json", "policyFile": "/etc/origin/master/audit.yaml"
```

Example master configuration file setup:

```
auditConfig:
  enabled: true
  auditFilePath: "/var/log/audit-ocp.log"
  policyFile: "/etc/origin/master/audit.yaml"
  logFormat: "json"
```

### Set an audit log policy

The sample auditing policy provided below will log the following {{site.prodname}} resources:

- `GlobalNetworkPolicy`
- `GlobalNetworkSet`
- `NetworkPolicy`
- `Tier`

And the following Kubernetes resources that are directly involved in {{site.prodname}} policy
evaluation:

- `Pod`
- `Namespace`
- `ServiceAccount`

It will log the following actions on those resources:

- `create`
- `update`
- `patch`
- `delete`

For brevity, we omit the `RequestReceived` stage logging below since we log
the requests as they are processed, rather than when they are received.

We also record the events with their response from the API server at the `RequestResponse` level
for better insights into the request life cycle.

- **Example Kubernetes API server policy**

  Configure the Kubernetes API Server with the following audit policy to log changes to the
  Kubernetes objects involved in {{site.prodname}} policy.

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
    - update
    - delete
    resources:
    - group: ""
      resources:
      - pods
      - namespaces
      - serviceaccounts
    - group: networking.k8s.io
      resources:
      - networkpolicies
    - group: extensions
      resources:
      - networkpolicies
  ```

- **Example {{site.prodname}} API server policy**

  The {{site.prodname}} API Server is configured the same as the Kubernetes API Server.  Use the following policy file to
  log the {{site.prodname}} resources.

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
    - update
    - delete
    resources:
    - group: projectcalico.org
      resources:
      - globalnetworkpolicies
      - networkpolicies
      - globalnetworksets
      - tiers
  ```
