---
title: Configure compliance reporting for managed cloud providers
---

### Big picture

Configure {{site.prodname}} compliance reporting and audit logs for your managed Kubernetes cloud provider.

### Value

{{site.prodname}} provides compliance reports and a dashboard so you can easily assess Kubernetes workloads for regulatory compliance. The reports provide a complete inventory of regulated workloads, along with evidence of enforcement of network controls for these workloads. Additionally, audit reports are available to see changes to any network security controls. These reports are essential tools for compliance managers to prove compliance for regulated workloads.

### Features

This how-to guide uses the following {{site.prodname}} features:

- **LogCollector** resource
- **GlobalReport** resource for compliance reporting

### Concepts

#### Compliance reporting with managed cloud providers

{{site.prodname}} compliance reports use audit logs data to generate reports. To extend compliance reporting to Kubernetes managed cloud providers, {{site.prodname}} gets audit logs from the native cloud provider logging service.

### Before you begin...

**Required**
- Install {{site.prodname}} for your managed cloud provider using [Managed public cloud]({{site.url}}/{{page.version}}/getting-started/kubernetes/managed-public-cloud/eks)
- Configure access to the [Tigera Secure EE Manager UI]({{site.url}}/{{page.version}}/getting-started/access-the-manager)
- Create a [user login]({{site.url}}/{{page.version}}/getting-started/create-user-login)

**Support**

In this release, {{site.prodname}} supports compliance reports for **Amazon EKS**.

### How to

Complete the following steps to configure Amazon EKS for compliance reporting:

1. [Enable audit logs in EKS](#enable-audit-logs-in-eks)
1. [Create a restricted AWS user for compliance reporting](#create-a-restricted-aws-user-for-compliance-reporting)
1. [Update Tigera Secure EE log collector with EKS values](#update-tigera-secure-ee-log-collector-with-eks-values)
1. [Configure authentication between Tigera Secure EE and Cloudwatch logs](#configure-authentication-between-tigera-secure-ee-and-cloudwatch-logs)

#### Enable audit logs in EKS

Amazon EKS writes Kubernetes audit logs to [Amazon Cloudwatch logs](https://aws.amazon.com/cloudwatch/).

1. In the EKS management console, access your EKS cluster.
1. Under **Logging**, click **Update**.
1. Enable the **Audit** option, and click **Update**.
   <img src="/images/audit-log.png" alt="Audit Log" width="300">
1. Wait for the update to complete.
  The blue progress bar at the top of the page displays the message, “Cluster config update in progress.”
1. Under **Logging**, **Cloudwatch**, make a note of the URL value for a later step, then click the link.
   <img src="/images/cloudwatch-url.png" alt="Cloudwatch Logging" width="400">
1. In the **Log Streams** list, make note of the common prefix (for example, kube-apiserver-audit) for a later step.
   <img src="/images/log-streams.png" alt="Log Streams" width="400">
1. Make note of the region where the cluster is hosted (for example, `us-west-2`) for a later step.

#### Create a restricted AWS user for compliance reporting

1. Go to the **AWS IAM console** and add a user.
1. On the **Add user** page, make these changes:

   a. Select **Access type**, **Programmatic access**.
      <img src="/images/programmatic-access.png" alt="Programmatic access" width="600">

   b. In the Set permissions section, select the policy, `CloudWatchLogsReadOnlyAccess` to set read only permissions.
      <img src="/images/cloudwatch-readonly.png" alt="Cloudwatch URL" width="400">
1. (Optional) In the **Add tags** section, add a tag for the user based on your cluster information.
1. Click **Submit** to create a restricted user.

#### Update Tigera Secure EE log collector with EKS values

1. Update the `tigera-secure` LogCollector resource with values from the EKS configuration.

   where:
   - `additionalSources`: Section where EKS Cloudwatch logs are specified.
   - `eksCloudwatchLog`: Configuration section containing EKS Cloudwatch logs.
   - `fetchInterval`: Interval in seconds for {{site.prodname}} to get logs from Cloudwatch. Default: 600 seconds. This value works for most use cases.
   - `groupName`: Name of the `Log Group` (value from "Enable audit logs in EKS")
   - `region`: AWS region where EKS cluster is hosted (value from "Enable audit logs in EKS")
   - `streamPrefix`: Prefix of `Log Stream` (value from "Enable audit logs in EKS")

   **Example**

   ```
   apiVersion: operator.tigera.io/v1
   kind: LogCollector
   metadata:
     name: tigera-secure
   spec:
     additionalSources:
       eksCloudwatchLog:
         fetchInterval: "600"
         groupName: /aws/eks/mitch-eks-kube-audit-log-forwarder/cluster
         region: us-west-2
         streamPrefix: kube-apiserver-audit-
   status:
     state: Ready
   ```

#### Configure authentication between Tigera Secure EE and Cloudwatch logs

In this step, you add AWS authentication information to enable {{site.prodname}} to get logs from the EKS Cloudwatch instance.

Add a Secret with the name, `tigera-eks-log-forwarder-secret` in the namespace, `tigera-operator`, and the AWS [Security Credentials](https://docs.aws.amazon.com/general/latest/gr/aws-sec-cred-types.html) in the data section.

```
apiVersion: v1
kind: Secret
metadata:
  name: tigera-eks-log-forwarder-secret
  namespace: tigera-operator
type: Opaque
data:
  aws-id: $(echo -n <YOUR-AWS-ACCESS-KEY-ID> | base64 -w0)
  aws-key: $(echo -n <YOUR-AWS-ACCESS-KEY-KEY-SECRET> | base64 -w0)
```

### Above and beyond

- [Inventory auditing]({{site.url}}/{{page.version}}/security/compliance-reports)
- [CIS benchmarks]({{site.url}}/{{page.version}}/security/compliance-reports-cis)
