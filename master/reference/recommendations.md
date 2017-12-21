---
title: Deployment Recommendations and limits
---

This document contains recommendations and limits for deploying CNX in production, particularly at high scale.
It supplements the basic [system requirements]({{site.baseurl}}/{{page.version}}/reference/requirements).

It supplements the Kubernetes [building large clusters](https://kubernetes.io/docs/admin/cluster-large/)
guide and the OpenShift [cluster limits](https://docs.openshift.com/container-platform/3.7/scaling_performance/cluster_limits.html)
guide.

## Cluster size based recommendations

- Use BGP route reflectors for clusters of more than 50 nodes.  A multi-AS design peering to L3 switches
  is an alternative where available, such as on premesis.
- If using the Kubernetes Datastore Driver with more than 50 nodes, use the Typha fan-out agent.

## Limitations

- The number of tiers times the number of concurrent CNX Manager users (browser sessions) must not
  exceed 100.  For example, 10 tiers and 10 sessions.
- Large numbers of services and pod churn combine to make kube-proxy issue a very large amount of iptables
  updates.  This negatively impacts the responsiveness of changes to those services and also that of other
  iptables users such as CNX.
