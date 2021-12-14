---
title: Detect and alert on performance issues
description: Enable machine learning to automatically alert you when clusters have performance issues. 
canonical_url: /visibility/performance-hotspots
---

> **Note**: This feature is **tech preview**. Tech preview features may be subject to significant changes 
> before they become GA.
{: .alert .alert-info}

### Big picture


Configure deployments of {{site.prodname}} to automatically detect performance issues
within your cluster and alert on them.

### Value


{{site.prodname}} includes a proprietary machine learning engine that analyzes patterns and indicates potential 
performance issues. **Performance hotspots** can include spikes in data transmission that can help you 
understand usage patterns, or also anomalous behavior. 
Machine learning adds another dimension to our proprietary anomaly detection feature so we can control, 
train, predict, and fine tune our product for threat defense and security. 
All you need to do is install a service on a cluster deployment, configure a few environment variables, 
and add alerts; the machine learning tool does all the work after that. If there are any performance “hotspots”, 
you will get alerts in the Manager UI **Alert List** page.


### Features

This how-to guide uses the following {{site.prodname}} features:
- **Performance Hotspots (PH)** service with anomaly detection variables.


### Concepts 

#### About performance hotspots

The {{site.prodname}} Performance Hotspots (PH) is a REST web-service that detects anomalous 
behavior for patterns and alerts on them. It uses the neural network from 
the {% include open-new-window.html text='GluonTS' url='https://github.com/awslabs/gluon-ts/' %} package, 
a Python toolkit for probabilistic time series modeling from the AWS Labs, built around Apache MXNet. 
The PH service can also work as a daemon service that periodically retrains the model and performs hotspot 
detection.

Based on Elasticsearch logs, the PH service performs these high-level tasks:
- Learns the normal behaviour and patterns of cluster nodes, pods, services, and other entities that 
send log records (applications, load balancers, databases, etc.). 
- Collects data from different cluster log fields (individual or aggregated) such 
as connections, bytes sent, latencies, and counters. 
- Learns time patterns (hourly, daily, or other) in field values. For example, there can be 
a peak in the connections to the authorization service in the morning, then a big data transmission 
when the database starts a backup operation, etc.

#### Trade offs: precision versus data retention

The PH service consumes {{site.prodname}} [flows]({{site.baseurl}}/visibility/elastic/flow), 
[l7]({{site.baseurl}}/visibility/elastic/l7),
and [dns]({{site.baseurl}}/visibility/elastic/dns) logs. The more data that is used 
in neural network training, the more precise the detection of performance issues. However, 
the more data the PH service ingests, the more costly it becomes in terms of data retention. 
The PH service default settings are configured for optimal detection quality. However, you can 
configure the amount of data that goes to the PH service, and the number of alerts.

#### How it works

The performance hotspots model uses these fields from the logs:
- `flows` log: `bytes_in`, `bytes_out`, `num_flows`, `num_flows_started`, `num_flows_completed`, 
`packets_in`, `packets_out`, `http_requests_allowed_in`, `http_requests_denied_in`, `num_process_names`, 
`num_process_ids`, `num_original_source_ips` 
- `dns` log: `latency_count`, `latency_mean`, `latency_max`
- `l7` log: `count`, `duration_mean`, `duration_max`, `bytes_in`, `bytes_out`


If some log is not presented, the model is not using fields
from this log. You don't have to do any additional configuration for this. You also can remove
some logs from the processing with the **PH_PROCESSED_LOGS** environment variable.

A model is trained to find patterns in these fields. It searches for the patterns 
for all fields. 

Performance hotspots can represent an abnormal behaviour of a single field or a set of fields.

**Note:** The model adds more data to the training each day. Potentially it improves the detection quality.
But if abnormal behaviour follows the pattern, the model can stop recognize it as an anomaly.


### How To

#### Install and configure the PH service

1. Download the manifest:
   
    - For the **management** or **standalone** cluster:   
    ```bash
    curl {{ "/manifests/ph/performance-hotspots-deployment.yaml" | absolute_url }} -O
    ```
    - For the **managed** cluster:   
    ```bash
    curl {{ "/manifests/ph/performance-hotspots-deployment-managed.yaml" | absolute_url }} -O
    ```

2. Configure the PH service environment variables using the deployment manifest (yaml).
   
   For **managed clusters**, `cluster_name` is required. For standalone and management clusters, it is optional.
   
Where:
- **CLUSTER_NAME** - Default: "cluster". 
Name of the cluster where the PH service will detect performance hotspots. Replace this value only for
managed clusters.
- **PH_PROCESSED_LOGS** - Default: "flows,l7,dns". 
The {{site.prodname}} logs used as the service input data. 
- **PH_MAX_DOCS** - Default: 2000000. 
Maximum number of records of individual logs used for training. The larger the number, the more 
precise the training models, but also the more data that is read from the Elasticsearch storage, 
and the longer the training.
- **PH_ANOMALY_VALUE_THRESHOLD** - Default: 10. Defines the anomaly score threshold. 
Increase this value if you want fewer alerts and less precision in hotspot detection; 
decrease if you want more alerts and greater precision in hotspot detection.
- **PH_VALUE_TO_MEAN_THRESHOLD** - Default: 60. Defines the log field value, compared to the mean of this value, 
across the training dataset. Increase this value if you want fewer alerts and less precision in hotspot detection; 
decrease if you want more alerts and greater precision in hotspot detection.

Example:
   
   ```yaml
   env:
    - name: CLUSTER_NAME
      value: "my-cluster"
    - name: PH_PROCESSED_LOGS
      value: "flows,dns"
   ```
   
3. Apply the manifest
   
    - For **standalone** and **management** cluster:   
    ```bash
    kubectl apply -f performance-hotspots-deployment.yaml
    ```
    - For **managed** cluster:   
    ```bash
    kubectl apply -f performance-hotspots-deployment-managed.yaml
    ```

4. Verify that the PH service is running.

   Use this command to check if pod is ready or not:
   ```bash
    kubectl get pods -n tigera-intrusion-detection -l k8s-app=performance-hotspots
    ```

### Above and beyond

- [Configure alerts]({{site.baseurl}}/visibility/alerts)
