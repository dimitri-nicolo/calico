---
title: Archive logs to storage
description: Archive logs to S3, Syslog, or Splunk for maintaining compliance data.
---

### Big picture

Archive logs to Amazon S3, Syslog, or Splunk to meet data storage requirements for compliance.

### Value

Archiving your {{site.prodname}} Elasticsearch logs to storage services like Amazon S3, Syslog, or Splunk are reliable 
options for maintaining and consolidating your compliance data long term

### Features

This how-to guide uses the following {{site.prodname}} features:
- **LogCollector** resource

### How to

* [Archive logs to Amazon S3](#archive-logs-to-amazon-s3)
* [Archive logs to Syslog](#archive-logs-to-syslog)
* [Archive logs to Splunk](#archive-logs-to-splunk)


#### Archive logs to Amazon S3

**Operator install**

1. Create an AWS bucket to store your logs.
   You will need the bucket name, region, key, secret key, and the path in the following steps.

2. Create a Secret in the `tigera-operator` namespace named `log-collector-s3-credentials` with the fields `key-id` and `key-secret`.
   Example:

   ```
    kubectl create secret generic log-collector-s3-credentials \
    --from-literal=key-id=<AWS-access-key-id> \
    --from-literal=key-secret=<AWS-secret-key> \
    -n tigera-operator
   ```

3. Update the [LogCollector]({{site.baseurl}}/reference/installation/api#operator.tigera.io/v1.LogCollector)
   resource named, `tigera-secure` to include an [S3 section]({{site.baseurl}}/reference/installation/api#operator.tigera.io/v1.S3StoreSpec)
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
   by applying it, or after installation by editing the resource with the command:

   ```
   kubectl edit logcollector tigera-secure
   ```

**Helm deployment**

{% include content/s3_fluentd.md %}


#### Archive logs to Syslog

**Operator install**

1. Update the
   [LogCollector]({{site.baseurl}}/reference/installation/api#operator.tigera.io/v1.LogCollector)
   resource named `tigera-secure` to include
   a [Syslog section]({{site.baseurl}}/reference/installation/api#operator.tigera.io/v1.SyslogStoreSpec)
   with your syslog information.
   Example:
   ```
   apiVersion: operator.tigera.io/v1
   kind: LogCollector
   metadata:
     name: tigera-secure
   spec:
     additionalStores:
       syslog:
         # Syslog endpoint, in the format protocol://host:port
         endpoint: tcp://1.2.3.4:514
         # Packetsize is optional, if messages are being truncated set this
         #packetSize: 1024
   ```
   This can be done during installation by editing the custom-resources.yaml
   by applying it or after installation by editing the resource with the command:
   ```
   kubectl edit logcollector tigera-secure
   ```

**Helm deployment**

{% include content/syslog-fluentd.md %}


#### Archive logs to Splunk

**Operator deployment**

{{site.prodname}} uses Splunk's **HTTP Event Collector** to send data to Splunk server. To copy the flow, audit, and dns logs to Splunk, follow these steps:

1. Create a HTTP Event Collector token by following the steps listed in Splunk's documentation for your specific Splunk version. Here is the link to do this for {% include open-new-window.html text='Splunk version 8.0.0' url='https://docs.splunk.com/Documentation/Splunk/8.0.0/Data/UsetheHTTPEventCollector' %}.

2. Create a Secret in the `tigera-operator` namespace named `logcollector-splunk-credentials` with the field `token`.
   Example:

   ```
    kubectl create secret generic logcollector-splunk-credentials \
    --from-literal=token=<splunk-hec-token> \
    -n tigera-operator
   ```

3. Update the
   [LogCollector]({{site.baseurl}}/reference/installation/api#operator.tigera.io/v1.LogCollector)
   resource named `tigera-secure` to include
   a [Splunk section]({{site.baseurl}}/reference/installation/api#operator.tigera.io/v1.SplunkStoreSpec)
   with your Splunk information.
   Example:
   ```
   apiVersion: operator.tigera.io/v1
   kind: LogCollector
   metadata:
     name: tigera-secure
   spec:
     additionalStores:
       splunk:
         # Splunk HTTP Event Collector endpoint, in the format protocol://host:port
         endpoint: https://1.2.3.4:8088
   ```
   This can be done during installation by editing the custom-resources.yaml
   by applying it or after installation by editing the resource with the command:
   ```
   kubectl edit logcollector tigera-secure
   ```

**Helm deployment**

Currently not supported.
