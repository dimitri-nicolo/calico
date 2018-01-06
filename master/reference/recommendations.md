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

- Large numbers of services and pod churn combine to make kube-proxy issue a very large amount of iptables updates.  This negatively impacts the responsiveness of changes to those services and also that of other iptables users such as CNX.
- Due to the way the Kubernetes API Server interacts with an extension API Server, a single CNX API Server is only able to handle 250 concurrent connections.
- That in turn directly limits the no. of concurrent CNX Manager browser sessions, that can be served by a single CNX API Server. The number of tiers times the number of concurrent CNX Manager users (browser sessions) must not exceed 100.  For example, 10 tiers and 10 sessions.
- One can stretch this limit by increasing the no. of CNX API Server replicas in the deployment. Example:
   ```
   apiVersion: v1
   kind: Deployment
   metadata:
     name: cnx-apiserver
     namespace: kube-system
     labels:
       apiserver: "true"
   spec:
     replicas: 3 <<---
   .
   .
   ```
- kube-apiserver will also need to be enabled with the flag '--enable-aggregator-routing=true' for the connection sharing to take place. And a restart of the kube-apiserver will be required for the update to take effect. Example:
   ```
   .
   .
   spec:
     containers:
     - command:
       - kube-apiserver
       - --enable-aggregator-routing=true <<--
   .
   .
   ```
- The no. of replicas best suited for a given deployment can be based on the following asymtptotic formula:

   ```
   2st <= 250*r ; where

   's' no. of expected concurrent user sessions
   't' no. of tiers in the deployment
   'r' is the no. of replicas.
   ```
   So, if 't' were to be 40 tiers and 's' were to be 10 concurrent users, no. of replicas would need to be 4 for a fully functionaly deployment.

