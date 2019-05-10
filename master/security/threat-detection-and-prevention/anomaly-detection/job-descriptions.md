---
title: Job descriptions
canonical_url: https://docs.tigera.io/v2.4/security/threat-detection-and-prevention/anomaly-detection/job-descriptions
---

The following anomaly detection jobs are included in {{site.prodname}}

### Inbound connection spike
**Job ID**: `inbound_connection_spike`

Looks for pods that receive a higher than usual number of inbound flows.  This may indicate a denial of
service or other attack. The job looks for pods that are unusual with respect to their replica set,
and replica sets which are unusual with respect to the rest of the cluster.

### IP Sweep - External 
**Job ID**: `ip_sweep_external`

Looks for IPs outside the cluster that are sending packets to a large number of destinations within the
cluster. This may indicate an attacker gathering reconnaisance on the active IP addresses in your cluster.

### IP Sweep - Pods 
**Job ID**: `ip_sweep_pods`

Looks for pods in your cluster that are sending packets to a large number of destinations. This may indicate
an attacker has gained control of a pod and is gathering reconnaissance on what else they can reach. The job
compares pods both with other pods in their replica set, and with other pods in the cluster generally. 

### Outlier IP Activity - Pods 
**Job ID**: `pod_outlier_ip_activity`

Looks for pods that connect to anomalous or rare destination IP addresses. This may indicate a compromised pod
exfiltrating data, or contacting a malicious command-and-control server. 

### Port Scan - External
**Job ID**: `port_scan_external`

Looks for IPs outside the cluster that are sending packets to a large number of destination ports within the 
cluster. This may indicate an attacker gathering reconnaissance on which ports are accepting connections in
an attempt to find one they can exploit.

### Port Scan - Pods
**Job ID**: `port_scan_pods`

Looks for pods in your cluster that are sending packets to a large number of destination ports. This may indicate
an attacker has gained control of a pod and is gathering reconnaissance on what else they can reach. The job
compares pods both with other pods in their replica set, and with other pods in the cluster generally. 

### Service bytes anomaly 
**Job ID**: `service_bytes_anomaly`

Looks for pods that either send or receive an anomalously high amount of data.  This could indicate a
denial of service attack, data exfiltration, or other attacks. The job looks for pods that are unusual
with respect to their replica set, and replica sets which are unusual with respect to the rest of the cluster.
