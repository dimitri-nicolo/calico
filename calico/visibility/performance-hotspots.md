---
title: Detect and alert on performance anomalies
description: Enable machine learning to automatically alert you when clusters have performance issues. 
canonical_url: /visibility/performance-hotspots
---

>**Note**: This feature is tech preview. Tech preview features may be subject to significant changes before they become GA.
{: .alert .alert-info}

### Big picture

Automatically detect performance issues within your cluster and alert on them.

### Value

{{site.prodname}} includes an anomaly detection engine that analyzes patterns and indicates potential performance issues. Performance issues can include spikes in data transmission, and anomalous degradation in network communication that may impact application workloads. To enable the anomaly detection engine, all you need to do is install anomaly detection within your cluster. If there are any performance anomalies, you will automatically get alerts in the Manager UI.

### Features

This how-to guide uses the following {{site.prodname}} features:
- **Anomaly detection** 

### Concepts 

#### About performance anomalies

{{site.prodname}} anomaly detection detects anomalous behavior for patterns and 
alerts on them. This feature allows you to proactively determine if there is an issue (or not), and potentially 
resolve problems before service levels are compromised. Anomaly detection uses {{site.prodname}} Elasticsearch logs 
([flows]({{site.baseurl}}/visibility/elastic/flow) logs, [L7]({{site.baseurl}}/visibility/elastic/l7) logs, and [DNS]({{site.baseurl}}/visibility/elastic/dns) logs) to learn behavior of cluster nodes, pods, services, and other 
entities that send log records (applications, load balancers, databases, etc.).

Root causes of cluster performance anomalies are numerous, for example:

**Applications and microservices**
- Bugs in applications and microservices
- Underprovisioned applications (replicas)

**Network and infrastructure**
- Process or CPU overload due to surges in traffic 

#### About anomaly detection modeling

Anomaly detection uses a neural network and probabilistic time series modeling to automatically 
identify performance anomalies associated with workloads in your cluster. It can also work as a daemon that 
periodically retrains the model and performs anomaly detection. Anomaly detecton performs these 
high-level tasks:

- Learns the normal behaviour and patterns of cluster nodes, pods, services, and other entities that send log records 
(applications, load balancers, databases, etc.).
- Collects data from different cluster log fields (individual or aggregated) such as connections, 
bytes sent, latencies, and counters.
- Learns the time patterns (hourly, daily, or other) in field values. For example, there can be a peak 
in the connections to an authorization service in the morning, then a big data transmission when 
the database starts a backup operation.

For a list of performance anomaly detectors that are enabled by default, see [Anomaly detection reference]({{site.baseurl}}/reference/anomaly-detection/all-detectors#performance-anomaly-detectors).

### FAQ

**Do I need to configure anomaly values?**

>Anomalies are preconfigured with reasonable defaults that are optimized for performance and appropriate frequency and severity. You should not need to change the values unless you are getting too many alerts. 

**Will alerts be unusually high until the engine learns to distinguish normal from anomalous behavior?**

>Yes. If you have data for a cluster running several hours, that could result in an unusually high alert rate.

**If the engine detects more than one anomaly (ex. one in `flow` logs, and `DNS` logs), will I get separate alerts?**

>You can see several `suspicious_records` from different logs in the same alert if those records are presented in 
> the same time interval. Suspicious records in alert are grouped by log names.

**Where does anomaly detection run on multi-cluster management (mcm) deployments?**

>Anomaly detection runs on the management cluster, but users on managed clusters can enable/disable alerts. 

### Before you begin

**Supported** 

- Kubernetes/kubeadm, OpenShift, AWS/kOps, RKE, EKS, TKG, AKS, GKE, Windows

### How To

- [Install anomaly detection](#install-anomaly-detection)
- [Monitor anomalies and alerts](#monitor-anomalies-and-alerts)
- [Disable anomaly detectors](#disable-anomaly-detectors)

#### Install anomaly detection

{% include /content/anomaly-detection/install-common.md %}

#### Monitor anomalies and alerts

{% include /content/anomaly-detection/monitor.md %}

#### Disable anomaly detectors

{% include /content/anomaly-detection/disable.md %}

### Above and beyond

- [Anomaly detection reference]({{site.baseurl}}/reference/anomaly-detection/all-detectors)
- [Global Alert reference]({{site.baseurl}}/reference/resources/globalalert)
- [Detect and alert on security anomalies]({{site.baseurl}}/threat/security-anomalies)
