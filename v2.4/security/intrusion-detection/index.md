---
title: Intrusion detection overview
canonical_url: https://docs.tigera.io/v2.3/usage/intrusion-detection
---

{{site.tseeprodname}} includes software that analyzes the logs of network communications
in your cluster ([flow logs]) to find anomalies that may indicate a compromise of
your network or cluster.  As it depends on flow logs, it is supported on Kubernets or OpenShift
clusters.

Intrusion detection jobs run in the Elasticsearch cluster that contains the flow
logs. You can configure the jobs to continuously analyze your logs as they are
added to the Elasticsearch cluster, or you can manually control the exact time
range you wish to analyze.


[flow logs]: ../logs/elastic/flow
