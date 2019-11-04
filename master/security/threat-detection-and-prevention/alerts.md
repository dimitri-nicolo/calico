---
title: Alerts
canonical_url: https://docs.tigera.io/master/security/threat-detection-and-prevention/alerts
---

### Big picture

Define alert criteria for the Alerts page in {site.prodname}} Manager based on collected flow, DNS, and audit logs. 

### Value 

When it comes to alerts that indicate cluster compromise, cluster administrator need flexibility to ensure fine-grain tuning; too many alerts become noise. With {{site.prodname}}, you can configure alerts to detect log entries that match patterns, or aggregate log entries over key fields and alert when entry counts or metrics on aggregated fields meet a condition. For higher fidelity, you can use the alerts domain-specific query language to select only relevant data.

### Features

This how-to guide uses the following {{site.prodname}} features:

- **GlobalAlert** resource

### Before you begin...

**Required**

Privileges to manage GlobalAlert

**Recommended**

We recommend that you turn down the aggregation of flow logs sent to Elasticsearch for configuring threat feeds. If you do not adjust flow logs, {{site.prodname}} aggregates over the external IPs for allowed traffic, and alerts will not provide pod-specific results (unless the traffic is denied by policy). Go to: [FelixConfiguration]({{site.baseurl}}/{{page.version}}/reference/resources/felixconfig) and set the field, **flowLogsFileAggregationKindForAllowed** to **1**.

### How To

#### Create a global alert

1. Create a yaml file containing one or more alerts.
1. Apply the alert to your cluster.

   ```shell
   kubectl apply -f <your_alert_filename>
   ```

1. Wait until the alert runs, and check the status.

   ```shell
   kubectl get globalalert <your_alert_name> -o yaml
   ```
1. In {{site.prodname}} Manager, go the **Alerts** page to view events
as alert conditions are satisfied.

#### Examples

In the following example, TBD.

```yaml
apiVersion: projectcalico.org/v3
kind: GlobalAlert
metadata:
  name: example1
spec:
  description: "Example alert"
  severity: 100
  dataSet: flows
  metric: count
  condition: gt
  threshold: 100
```

In the following example, we identify source IPs probing for the Enternal-Blue-Romance-Synergy-Champion
Exploit (EBD-ID [45925]). 

```
apiVersion: projectcalico.org/v3
kind: GlobalAlert
metadata:
  name: ebd-id-45925
spec:
  description: "Probe for Apache Spark Exploit by ${source_ip}"
  severity: 80
  dataSet: flows
  query: action=allow AND proto=tcp AND dest_port=6066
  field: num_flows
  metric: sum
  condition: gt
  threshold: 0
```

In the following example, we identify pods labeled as `pci: true` that are sending 1MB of data or more
per hour to the public Internet.

```yaml
apiVersion: projectcalico.org/v3
kind: GlobalAlert
metadata:
  name: pci-data-transfer
spec:
  description: "Data transfer by ${source_namespace}/${source_name} to Internet detected (${sum} bytes)"
  severity: 70
  period: 60m
  lookback: 60m
  dataSet: flows
  query: source_type=wep AND source_labels.labels="pci=true" dest_type=net
  aggregateBy: [source_namespace, source_name]
  field: bytes_in
  metric: sum
  condition: gte
  threshold: 1000000
```

### Above and beyond

For all global alert options, see [GlobalAlert]({{site.baseurl}}/{{page.version}}/reference/resources/globalalert)
To troubleshoot alerts, see [Troubleshooting]({{site.baseurl}}/{{page.version}}/maintenance/troubleshooting)

[flow]: ../logs/elastic/flow
[dns]: ../logs/elastic/dns
[audit logs]: ../logs/elastic/ee-audit
[45925]: https://www.exploit-db.com/exploits/45925