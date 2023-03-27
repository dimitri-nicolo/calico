# Scenario: Compromised Kubernetes Pod

This scenario covers the situation where the attacker is able to compromise a Pod within a K8s cluster. This pod can be a worker pod, privileged pod, or infrastructure pod in any Node. We assume there is no network policy applied.

The attacker is able to:

* Reach all exposed services with Cluster IP, NodePort, etc within scope
* Reach externally and resources within VPC subnet (10.128.0.0/24) and K8s Nodes (10.128.0.0/24)
* Able to query the Kube DNS service for available services
* Get Token, Metadata, Credentials
* Access all data stored in container
* Access Host Mounted or connected resources
* Privileged Access underlying filesystem (Node)

## Honeypod Detections

* IP Enumuration
  * We can place a honeypod without exposing any services such that only neighboring pods can access the honeypod
* Exposed Service (Simulated)
  * We expose an INetSim HTTP service to the cluster such that it can be found via ClusterIP or DNS lookup
* Exposed Service (Nginx)
  * We expose an Nginx HTTP service to the cluster such that it can be found via ClusterIP or DNS lookup
* Vulnerable Service (MySQL)
  * We expose an empty MySQL service to the cluster such that it can be found via ClusterIP or DNS lookup

## Other Mitigations

* Proper namespacing, RBAC
* Network Policy for VPC subnet and external
* Non-root and not use privileged pod access unless absolutely needed
* Barebone OS Container Image
