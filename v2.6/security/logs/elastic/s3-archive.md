---
title: Archive logs to S3
redirect_from: latest/security/logs/elastic/s3-archive
canonical_url: https://docs.tigera.io/v2.3/usage/logs/elastic/s3-archive
---

{{site.prodname}} supports archiving flow and audit logs to Amazon S3.  This provides
a reliable option for maintaining your compliance data long term.  

### Configure S3 archiving

Configure S3 archiving based your {{site.prodname}} deployment:
- [Operator deployment](#operator-deployment)
- [Manual/Helm deployment](#manualhelm-deployment)

#### Operator deployment

To copy the flow/audit/dns logs to Amazon S3 cloud object storage, follow these steps:

1. Create an AWS bucket to store your logs.
   You will need the bucket name, region, key, secret key, and the path in the following steps.

1. Create a Secret in the `tigera-operator` namespace named `log-collector-s3-credentials` with the fields `key-id` and `key-secret`.
   Example:

   ```
    kubectl create secret generic log-collector-s3-credentials \
    --from-literal=key-id=<AWS-access-key-id> \
    --from-literal=key-secret=<AWS-secret-key> \
    -n tigera-operator
   ```

1. Update the [LogCollector](/{{page.version}}/reference/installation/api#operator.tigera.io/v1.LogCollector)
   resource named, `tigera-secure` to include an [S3 section](/{{page.version}}/reference/installation/api#operator.tigera.io/v1.S3StoreSpec)
   with your information noted from above.
   Example:

   ```
   apiVersion: operator.tigera.io/v1
   kind: LogCollector
   metadata:
     name: tigera-secure
   spec:
     additionalStores:
       s3:
         bucketName: <S3-bucket-name>
         bucketPath: <path-in-S3-bucket>
         region: <S3-bucket region>
   ```
   This can be done during installation by editing the custom-resources.yaml
   before applying it, or after installation by editing the resource with the command:

   ```
   kubectl edit logcollector tigera-secure
   ```

#### Manual/Helm deployment

{% include {{page.version}}/s3_fluentd.md %}
