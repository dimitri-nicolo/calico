---
title: Global Report Resource (GlobalReport)
---

A global report resource is a configuration for generating compliance reports. A global report configuration in {{ site.prodname }} lets you:
- Specify report contents, frequency, and data filtering
- Specify the node(s) to include in report generation jobs
- Enable/disable creation of new jobs for generating the report

**Note**: Currently, global reports can only be configured using the `kubectl` command using these case-sensitive aliases:
`globalreport.projectcalico.org`, `globalreports.projectcalico.org` and abbreviations.

### Sample YAML

```
apiVersion: projectcalico.org/v3
kind: GlobalReport
metadata:
  name: weekly-full-inventory
spec:
  reportType: inventory
  schedule: 0 0 * * 0
  jobNodeSelector:
    nodetype: infrastructure

---

apiVersion: projectcalico.org/v3
kind: GlobalReport
metadata:
  name: hourly-accounts-networkaccess
spec:
  reportType: network-access
  endpointsSelection:
    namespaces:
      names: ["payable", "collections", "payroll"]
  schedule: 0 * * * *

---

apiVersion: projectcalico.org/v3
kind: GlobalReport
metadata:
  name: monthly-widgets-controller-tigera-policy-audit
spec:
  reportType: policy-audit
  schedule:  0 0 1 * *
  endpointsSelection:
    serviceAccount:
      names: ["controller"]
    namespace:
      names: ["widgets"]
```

### GlobalReport Definition

#### Metadata

| Field       | Description                              | Accepted Values   | Schema  |
|-------------|------------------------------------------|-------------------|---------|
| name        | The name of this report.                 | Lower-case alphanumeric with optional `-` or `.`  | string  |
| labels      | A set of labels to apply to this report. |                   | map     |

#### Spec

| Field                | Description                                    | Required | Accepted Values | Schema    |
|----------------------|------------------------------------------------|----------|-----------------|-----------|
| reportType           | Name of the report type which defines the report content. {{ site.prodname }} supported report types are inventory, network-access, and policy-audit. | Yes | inventory, network-access, audit | string |
| endpointsSelection   | Specify which endpoints are in scope. If omitted, selects everything. ||| [EndpointsSelection](#endpointsselection) |
| schedule             | Configure report frequency by specifying start and end time in [cron-format][cron-format]. Reports are started 30 minutes (configurable) after the scheduled value to allow enough time for data archival. Cron format enforces a maximum limit of two reports an hour. | Yes || string |
| jobNodeSelector      | Specify the node(s) for scheduling the report jobs using selectors. ||| map |
| suspend              | Disable future scheduled report jobs. In-flight reports are not affected. ||| bool |


#### EndpointsSelection

| Field            | Description                                  | Schema              |
|------------------|----------------------------------------------|---------------------|
| endpointSelector | Endpoint label selector to restrict endpoint selection. | string              |
| namespaces       | Namespace name and label selector to restrict endpoints by selected namespaces. | NamesAndLabelsMatch |
| serviceAccounts  | Service account name and label selector to restrict endpoints by selected service accounts. | NamesAndLabelsMatch |

#### NamesAndLabelsMatch

| Field    | Description                            | Schema |
|----------|----------------------------------------|--------|
| names    | Set of resource names.                 | list   |
| selector | Selects a set of resources by label.   | string |

Use the `NamesAndLabelsMatch`to limit the scope of endpoints. If both `names`
and `selector` are specified, the resource is identified using label *AND* name
match.

> **Note**: To use the {{site.prodname}} compliance reporting feature, you must ensure all required resource types
> are being audited and the logs archived in Elasticsearch. You must explicitly configure the [Kubernetes API
> Server](/{{page.version}}/security/logs/elastic/ee-audit#kubernetes) to send audit logs for Kubernetes-owned resources
> to Elasticsearch. 

### Supported operations

| Datastore type        | Create/Delete | Update | Get/List | Notes|
|-----------------------|---------------|--------|----------|------|
| etcdv3                | Yes           | Yes    | Yes      ||
| Kubernetes API server | Yes           | Yes    | Yes      ||

[cron-format]: https://en.wikipedia.org/wiki/Cron
