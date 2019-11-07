---
title: Compliance reports
redirect_from: latest/security/compliance-reports
---

### Big picture

Assess Kubernetes workloads and environments for regulatory compliance to enforce controls, and generate audit
and evidence data -- so you can prove compliance for these highly dynamic and ephemeral workloads.

### Value

{{ site.prodname }} provides compliance reports and a dashboard so you can easily assess Kubernetes workloads for
regulatory compliance.

Existing compliance tools that rely on periodic snapshots, do not provide accurate assessments of Kubernetes workloads
against your compliance standards. {{ site.prodname }} compliance reports provide a complete inventory of regulated
workloads, along with evidence of enforcement of network controls for these workloads. Additionally, audit reports are
available to see changes to any network security controls. These reports are essential tools for compliance managers to
prove compliance for regulated workloads.

### Features

This how-to guide uses the following {{ site.prodname }} features:

- Predefined **compliance reports** (Inventory, Network Access, Policy Audit) that are installed with {{ site.prodname }}
- A **GlobalReport** resource to schedule periodic report generation, specify which nodes to include, and manually run reports
- The **compliance dashboard** in {{ site.prodname }} Manager to view and export reports from Elasticsearch
- Kubernetes RBAC to grant report view/manage permissions

### Concepts

#### Compliance reports at a glance

Compliance report are based on archived flow logs and audit logs for the following resources:

- Pods
- Host endpoints
- Service accounts
- Namespaces
- Kubernetes service endpoints
- Global network sets
- Calico and Kubernetes network policies
- Global network policies

Compliance reports provide the following high-level information:

- **Protection**
  - Endpoints explicitly protected using ingress or egress policy
  - Endpoints with Envoy enabled

- **Policies and services**
  - Policies and services associated with endpoints
  - Policy audit logs

- **Traffic**
  - Allowed ingress/egress traffic to/from namespaces
  - Allowed ingress/egress traffic to/from the internet

From {{ site.prodname }} Manager, you can export:

- Compliance report contents (csv file)
- Policy audit logs (json or yaml format)

### Before you begin...

To ensure accurate timestamps for audit logs, time synchronize all nodes in your Kubernetes clusters using NTP or similar.

### How To

- [Configure audit logs](#configure-audit-logs)
- [Configure report permissions](#configure-report-permissions)
- [Configure and schedule reports](#configure-and-schedule-reports)
- [View report generation status](#view-report-generation-status)
- [Manually run reports](#manually-run-reports)

#### Configure audit logs

By default, {{ site.prodname }} provides audit logs in Elasticsearch for the following {{ site.prodname }} resources:

- GlobalNetworkPolicy
- GlobalNetworkSet
- NetworkPolicy
- Tier
- HostEndpoint

For compliance reports, you must configure audit logs for these additional Kubernetes resources:

- Pod
- Namespace
- ServiceAccount
- NetworkPolicy

Follow these steps to configure and produce audit logs:

1. Start using `kubectl` (rather than calicoctl). Because compliance reports use audited data, you must use `kubectl`,
   the Kubernetes API, or the {{ site.prodname }} Manager to manage policy, tiers and host endpoints. If you use `calicoctl`
   you will not get audit logs for changes to these resources.
1. [Configure Kubernetes audit logs](/{{page.version}}/security/logs/elastic/ee-audit#enabling-auditing-for-other-resources).

#### Configure report permissions

Report permissions are granted using the standard Kubernetes RBAC based on ClusterRole and ClusterRoleBindings. The
following table outlines the required RBAC verbs for each resource type for a specific user actions.

| **Action**                                              | **globalreporttypes**           | **globalreports**                 | **globalreports/status** |
| ------------------------------------------------------- | ------------------------------- | --------------------------------- | ------------------------ |
| Manage reports (create/modify/delete)                   |                                 | *                                 | get                      |
| View status of report generation through kubectl        |                                 | get                               | get                      |
| List the generated reports and summary status in the UI |                                 | list + get (for required reports) |                          |
| Export the generated reports from the UI                | get (for the particular report) | get (for required reports)        |                          |

The following sample manifest creates RBAC for three users: Paul, Candice and David.

- Paul has permissions to create/modify/delete the report schedules and configuration, but does not have permission to export generated reports from the UI.
- Candice has permissions to list and export generated reports from the UI, but cannot modify the report schedule or configuration.
- David has permissions to list and export generated `dev-inventory` reports from the UI, but cannot list or download other reports, nor modify the report
  schedule or configuration.

```
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1beta1
metadata:
  name: tigera-compliance-manage-report-config
rules:
- apiGroups: ["projectcalico.org"]
  resources: ["globalreports"]
  verbs: ["*"]
- apiGroups: ["projectcalico.org"]
  resources: ["globalreports/status"]
  verbs: ["get", "list", "watch"]

---

kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1beta1
metadata:
  name: tigera-compliance-manage-report-config
subjects:
- kind: User
  name: paul
  apiGroup: rbac.authorization.k8s.io
roleRef:
  kind: ClusterRole
  name: tigera-compliance-manage-report-config
  apiGroup: rbac.authorization.k8s.io

---

kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1beta1
metadata:
  name: tigera-compliance-list-download-all-reports
rules:
- apiGroups: ["projectcalico.org"]
  resources: ["globalreports"]
  verbs: ["get", "list"]
- apiGroups: ["projectcalico.org"]
  resources: ["globalreporttypes"]
  verbs: ["get"]

---

kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1beta1
metadata:
  name: tigera-compliance-list-download-all-reports
subjects:
- kind: User
  name: candice
  apiGroup: rbac.authorization.k8s.io
roleRef:
  kind: ClusterRole
  name: tigera-compliance-list-download-all-reports
  apiGroup: rbac.authorization.k8s.io

---

kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1beta1
metadata:
  name: tigera-compliance-list-download-dev-inventory
rules:
- apiGroups: ["projectcalico.org"]
  resources: ["globalreports"]
  verbs: ["list"]
- apiGroups: ["projectcalico.org"]
  resources: ["globalreports"]
  verbs: ["get"]
  resourceNames: ["dev-inventory"]
- apiGroups: ["projectcalico.org"]
  resources: ["globalreporttypes"]
  verbs: ["get"]
  resourceNames: ["dev-inventory"]

---

kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1beta1
metadata:
  name: tigera-compliance-list-download-dev-inventory
subjects:
- kind: User
  name: david
  apiGroup: rbac.authorization.k8s.io
roleRef:
  kind: ClusterRole
  name: tigera-compliance-list-download-dev-inventory
  apiGroup: rbac.authorization.k8s.io
```

#### Configure and schedule reports

To configure and schedule a compliance report, create a [GlobalReport](/{{page.version}}/reference/resources/globalreport) with the following information.

| **Fields**      | **Description**                                              |
| --------------- | ------------------------------------------------------------ |
| name            | Unique name for your report.                                 |
| reportType      | One of the following predefined report types: `inventory`, `network-access`, `policy-audit`. |
| schedule        | The start and end time of the report using [crontab format](https://en.wikipedia.org/wiki/Cron).  To allow for archiving, reports are generated approximately 30 minutes after the end time. A single report is limited to a maximum of two per hour. |
| endpoints       | **Optional**. For inventory and network-access reports, specifies the endpoints to include in the report.  For the policy-audit report, restricts audit logs to include only policies that apply to the selected endpoints. If not specified, the report includes all endpoints and audit logs.   |
| jobNodeSelector | **Optional**. Limits report generation jobs to specific nodes. |
| suspend         | **Optional**. Suspends report generation. All in-flight reports will complete, and future scheduled reports are suspended. |

>**Note**: GlobalReports can only be configured using kubectl (not calicoctl); and they cannot be edited in the Tigera
Secure EE Manager UI.
{: .alert .alert-info}

The following sections provide sample schedules for the predefined reports.

#### Weekly reports, all endpoints

The following report schedules weekly inventory reports for *all* endpoints. The jobs that create the reports will run
on the infrastructure nodes (e.g. nodetype == 'infrastructure').

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
```

#### Daily reports, selected endpoints

The following report schedules daily inventory reports for production endpoints (e.g. deployment == ‘production’).

```
apiVersion: projectcalico.org/v3
kind: GlobalReport
metadata:
  name: daily-production-inventory
spec:
  reportType: inventory
  endpoints:
    selector: deployment == 'production'
  schedule: 0 0 * * *
```

#### Hourly reports, endpoints in named namespaces

The following report schedules hourly network-access reports for the accounts department endpoints, that are
specified using the namespace names: **payable**, **collections** and **payroll**.

```
apiVersion: projectcalico.org/v3
kind: GlobalReport
metadata:
  name: hourly-accounts-networkaccess
spec:
  reportType: network-access
  endpoints:
    namespaces:
      names: ["payable", "collections", "payroll"]
  schedule: 0 * * * *
```

#### Daily reports, endpoints in selected namespaces

The following report schedules daily network-access reports for the accounts department with endpoints specified using
 a namespace selector.

```
apiVersion: projectcalico.org/v3
kind: GlobalReport
metadata:
  name: daily-accounts-networkaccess
spec:
  reportType: network-access
  endpoints:
    namespaces:
      selector: department == 'accounts'
  schedule: 0 0 * * *
```

#### Monthly reports, endpoints for named service accounts in named namespaces

The following schedules monthly audit reports. The audited policy is restricted to policy that applies to
widgets/controller endpoints specified by the namespace **widgets** and service account **controller**.

```
apiVersion: projectcalico.org/v3
kind: GlobalReport
metadata:
  name: monthly-widgets-controller-tigera-policy-audit
spec:
  reportType: policy-audit
  schedule:  0 0 1 * *
  endpoints:
    serviceAccounts:
      names: ["controller"]
    namespaces:
      names: ["widgets"]
```

#### View report generation status

To view the status of a report, you must use the `kubectl` command. For example:

```bash
kubectl get globalreports.projectcalico.org daily-inventory.p -o yaml
```

In a report, the job status types are:

- **lastScheduledReportJob**:
  The most recently scheduled job for generating the report. Because reports are scheduled in order, the “end time” of
  this report will be the “start time” of the next scheduled report.
- **activeReportJobs**:
  Default = allows up to 5 concurrent report generation jobs.
- **lastFailedReportJobs**:
  Default = keeps the 3 most recent failed jobs and deletes older ones. A single report generation job will be retried
  up to 6 times (by default) before it is marked as failed.
- **lastSuccessfulReportJobs**:
  Default = keeps the 2 most recent successful jobs and deletes older ones.

#### Change the default report generation time

By default, reports are generated 30 minutes after the end of the report, to ensure all of the audit data is archived.
(However, this gap does not affect the data collected “start/end time” for a report.)

You can adjust the time for audit data for cases like initial report testing, to demo a report, or when manually
creating a report that is not counted in global report status.

To change the delay, go to the installation manifest, and uncomment and set the environment
`TIGERA_COMPLIANCE_JOB_START_DELAY`. Specify value as a [Duration string][parse-duration].

#### Manually run reports

You can manually run reports at any time. For example, run a manual report:

- To specify a different start/end time
- If a scheduled report fails

{{ site.prodname }} GlobalReport schedules Kubernetes Jobs which create a single-run pod to generate a report and store it
in Elasticsearch. Because you need to run manual reports as a pod, you need higher permissions: allow `create` access
access for pods in namespace `tigera-compliance` using the `tigera-compliance-reporter` service account.

To manually run a report:

1. Download the pod template corresponding to your installation method.  
   **Operator**

   ```bash
   curl -O {{site.url}}/{{page.version}}/manifests/compliance-reporter-pod.yaml
   ```
   **Helm**
   ```bash
   curl -o {{site.url}}/{{page.version}}/manifests/compliance-reporter-pod-es-config.yaml
   ```

1. If you installed using the Operator, properly configure the template to read the correct certificates.
   First we will need to get the name of the `compliance-reporter-token`
   ```bash
   kubectl get secrets -n tigera-compliance
   ```
   The secret name should look something like this: `compliance-reporter-token-hmpaq`

   With the proper secret name, add it to your pod template
   ```bash
   export COMPLIANCE_REPORTER_TOKEN=<secret name>
   sed -i -e "s?<COMPLIANCE_REPORTER_TOKEN>?$COMPLIANCE_REPORTER_TOKEN?g" compliance-reporter-pod.yaml
   ```

1. Edit the template as follows:
   - Edit the pod name if required.
   - If you are using your own docker repository, update the container image name with your repo and image tag.
   - Set the following environments according to the instructions in the downloaded manifest:
     - `TIGERA_COMPLIANCE_REPORT_NAME`
     - `TIGERA_COMPLIANCE_REPORT_START_TIME`
     - `TIGERA_COMPLIANCE_REPORT_END_TIME`
1. Apply the updated manifest, and query the status of the pod to ensure it completes.
   Upon completion, the report is available in {{ site.prodname }} Manager.

   ```bash
   # Apply the compliance report pod
   kubectl apply -f compliance-reporter-pod.yaml

   # Query the status of the pod
   kubectl get pod <podname> -n tigera-compliance
   ```

>**Note**: Manually-generated reports do not appear in GlobalReport status.
{: .alert .alert-info}

### Above and beyond

- For details on configuring and scheduling reports, see [Global Reports](/{{page.version}}/reference/resources/globalreport).
- For report field descriptions, see [Compliance Reports](/{{page.version}}/reference/compliance-reports/).

[parse-duration]: https://golang.org/pkg/time/#ParseDuration
