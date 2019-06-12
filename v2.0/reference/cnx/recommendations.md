---
title: Deploying CNX Manager
---

This document contains recommendations and limits for deploying CNX Manager in production, particularly at high scale.

## Preventing CNX Manager lockouts

We recommend creating network policy to make it harder to accidentally lock yourselves out of CNX Manager.  Such policy should be created in a low ordered tier (i.e., one that applies early), and take the following form.

- An ingress policy to allow https (port 443) access to CNX Manager from wherever users will need to access CNX Manager.

- An ingress policy to allow access to the CNX API Servers from the Kubernetes API Servers on ports 443 and 5443.

- Egress policy to allow CNX API Server access to the Kubernetes API Server and Calico datastore.


## Maximum number of browser sessions

Large numbers of services and pod churn combine to make kube-proxy issue a very large amount of iptables updates.  This negatively impacts the responsiveness of changes to those services and also that of other iptables users such as CNX.

Due to the way the Kubernetes API Server interacts with an extension API Server, a single CNX API Server is only able to handle 250 concurrent connections. That in turn directly limits the number of concurrent CNX Manager browser sessions that can be served by a single CNX API Server. The number of tiers times the number of concurrent CNX Manager users (browser sessions) must not exceed 100.  For example, 10 tiers and 10 sessions.

One can stretch this limit by increasing the number of CNX API Server replicas in the deployment. Example:

```
apiVersion: v1
kind: Deployment
metadata:
  name: cnx-apiserver
  namespace: kube-system
  labels:
    apiserver: "true"
  spec:
    replicas: 3
```

kube-apiserver will also need to be enabled with the flag `--enable-aggregator-routing=true` for the connection sharing to take place. A restart of the kube-apiserver will be required for the update to take effect. Example:

```
spec:
  containers:
    - command:
      - kube-apiserver
      - --enable-aggregator-routing=true 
```

The number of replicas best suited for a given deployment can be based on the following asymptotic formula.

```
2st <= 250*r 
```
where
- `s` is the number of expected concurrent user sessions
- `t` is the number of tiers in the deployment
- `r` is the number of replicas

For example, if you had 40 tiers and 10 concurrent users, you would need four replicas for full functionality.

