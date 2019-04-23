---
title: Global Report Resource (GlobalReport)
---

A global report resource is a configuration for generating compliance reports. A global report configuration in {{ site.prodname }} lets you:
- Specify report contents, frequency, and data filtering
- Specify the node(s) to include in report generation jobs
- Enable/disable creation of new jobs for generating the report

**Note**: Currently, global reports can only be configured using the `kubectl` command using these case-sensitive aliases: `globalreport`, `globalreports`

### Sample YAML

```
apiVersion: projectcalico.org/v3
kind: GlobalReport
metadata:
  name: inventory-report
  labels:
    deployment: production
spec:
  reportType: inventory
  endpointsSelection:
    endpointSelector: resource == "high"
    namespaces:
      names:
        - app-storage
      selector: app == "identity-manager"
    serviceAccounts:
      names:
        - app-dev
      selector: role == "appDev"
  schedule: "0 * * * *"
  jobNodeSelector:
    resource: high
  suspend: false

---

apiVersion: projectcalico.org/v3
kind: GlobalReport
metadata:
  name: audit-report
  labels:
    deployment: production
spec:
  reportType: audit
  auditEventsSelection:
    resources:
      - kind: globalnetworkpolicy
        name: internal-access
        namespace: app-storage
  schedule: "30 * * * *"
  jobNodeSelector:
    resource: high
  suspend: false
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
| reportType           | Name of the report type resource and the content that is generated. {{ site.prodname }} out-of-the-box report types includes audit, inventory, and networkaccess.| Yes | audit, inventory, networkaccess | string |
| endpointsSelection   | Specify which endpoints are in scope, if endpoints data is included in the report. ||| [EndpointsSelection](#endpointsselection) |
| auditEventsSelection | Specify which audit events are in scope, and if auditing data is included in the report. ||| [AuditEventsSelection](#auditeventsselection) |
| schedule             | Configure report frequency by specifying start and end time in [cron-format][cron-format]. Reports are started 30 minutes (configurable) after the scheduled value to allow enough time for data archival. Cron format enforces a maximum limit of two reports an hour. | Yes || string |
| jobNodeSelector      | Specify the node(s) for scheduling the report jobs using selectors. ||| map |
| suspend              | Disable future scheduled report jobs. In-flight reports are not affected. ||| bool |


#### EndpointsSelection

| Field            | Description                                  | Schema              |
|------------------|----------------------------------------------|---------------------|
| endpointSelector | Selects endpoints by endpoint labels. You can specify endpoints using labels under `endpointSelector`. Use `namespaces` and `serviceAccounts` for further scoping.       | string              |
| namespaces       | Namespaces to include in report scope.       | NamesAndLabelsMatch |
| serviceAccounts  | Service accounts to include in report scope. | NamesAndLabelsMatch |

#### NamesAndLabelsMatch

| Field    | Description                            | Schema |
|----------|----------------------------------------|--------|
| names    | Set of resource names.                 | list   |
| selector | Specifies a set of resource by labels. | string |

Use the `NamesAndLabelsMatch`to limit the scope of endpoints. If both `names`
and `selector` are specified, the resource is identified using label *AND* name
match.

#### AuditEventsSelection

| Field     | Description         | Schema |
|-----------|---------------------|--------|
| resources | List of [ResourceID](#resourceid) that will be included in the report if including audit logs. | list   |

Use `AuditEventsSelection` to scope resources for inclusion in the audit report.

#### ResourceID

| Field     | Description           | Schema |
|-----------|-----------------------|--------|
| kind      | Resource type.        | string |
| name      | Resource Name.        | string |
| namespace | Resource namespace.   | string |

Use `ResourceID` to filter resources in the report configuration. Blank fields
value are treated as wildcards. For example, if Kind is set to `NetworkPolicy`
and all other fields are blank, the report includes all `NetworkPolicy`
resources across all namespaces, including both Calico and Kubernetes resource
types.

> **Note**: To include resources not added by {{ site.prodname }} in compliance
> audit reporting, you must configure the [Kubernetes API
> Server]({{site.url}}/{{page.version}}/security/logs/elastic/ee-audit#kubernetes)
> to send audit logs to Elasticsearch.

### Supported operations

| Datastore type        | Create/Delete | Update | Get/List | Notes|
|-----------------------|---------------|--------|----------|------|
| etcdv3                | Yes           | Yes    | Yes      ||
| Kubernetes API server | Yes           | Yes    | Yes      ||

[cron-format]: https://en.wikipedia.org/wiki/Cron
