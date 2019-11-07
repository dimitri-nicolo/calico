---
title: Scheduling to well-known nodes
redirect_from: latest/security/comms/reduce-nodes
canonical_url: https://docs.tigera.io/v2.3/usage/reduce-nodes
---

The following {{site.prodname}} components must accept connections on
a configurable port. By default, these components can be scheduled to any agent node.
Therefore, any given node may have the required ports open. To reduce the number of
nodes with the ports open to a subset of the total, consider
[scheduling these components to well-known nodes](https://kubernetes.io/docs/concepts/configuration/assign-pod-node/).

| Component                    | Default port        | Notes                                                                                                                                |
|------------------------------|---------------------|--------------------------------------------------------------------------------------------------------------------------------------|
| {{site.prodname}} API Server | TCP 8080 and 5443   |                                                                                                                                      |
| {{site.prodname}} Manager    | TCP 30003 and 9443  |                                                                                                                                      |
| Prometheus                   | TCP 9090            |                                                                                                                                      |
| Alertmanager                 | TCP 9093            |                                                                                                                                      |
| Typha                        | TCP 5473            | Deployed on Kubernetes clusters with more than 50 nodes that use the Kubernetes API datastore. Also required for federated identity. |
