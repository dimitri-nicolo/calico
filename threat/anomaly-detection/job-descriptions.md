---
title: Job descriptions
description: Summary of the anomaly detection jobs.
canonical_url: /threat/anomaly-detection/job-descriptions
---

The following anomaly detection jobs are included in {{site.prodname}}.

### IP Sweep detection
**Job ID**: `ip_sweep`

Looks for pods in your cluster that are sending packets to many destinations. This may indicate
an attacker has gained control of a pod and is gathering reconnaissance on what else they can reach. The job
compares pods both with other pods in their replica set, and with other pods in the cluster generally. 

### Port Scan detection
**Job ID**: `port_scan`

Looks for pods in your cluster that are sending packets to one destination on multiple ports. This may indicate
an attacker has gained control of a pod and is gathering reconnaissance on what else they can reach. The job
compares pods both with other pods in their replica set, and with other pods in the cluster generally.

### Inbound Service bytes anomaly 
**Job ID**: `bytes_in`

Looks for services that receive an anomalously high amount of data.  This could indicate a
denial of service attack, data exfiltrating, or other attacks. The job looks for services that are unusual
with respect to their replica set, and replica sets which are unusual with respect to the rest of the cluster.

### Outbound Service bytes anomaly 
**Job ID**: `bytes_out`

Looks for pods that send an anomalously high amount of data.  This could indicate a
denial of service attack, data exfiltrating, or other attacks. The job looks for pods that are unusual
with respect to their replica set, and replica sets which are unusual with respect to the rest of the cluster.

### Process restarts anomaly 
**Job ID**: `process_restarts`

Looks for pods with excessive number of the process restarts.  This could indicate problems with the processes, 
as resource problems or attacks. The job looks for pods that are unusual with respect to their process restart 
behavior.


