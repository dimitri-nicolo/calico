# Scenario: Compromised K8s Pod
This scenario covers the situation where the attacker is able to compromise a Pod within a K8s cluster. This pod can be a worker pod, privileged pod, or infrastructure pod in any Node. We assume there’s no network policy applied.

**The attacker is able to:**
* Reach all exposed services with Cluster IP, NodePort, etc within scope
* Reach externally and resources within VPC subnet (10.128.0.0/24) and K8s Nodes (10.128.0.0/24)
* Able to query the Kube DNS service for available services
* Get Token, Metadata, Credentials
* Access all data stored in container
* Access Host Mounted or connected resources
* [Privileged] Access underlying filesystem (Node)

**Honeypod Detections:**
* IP Enumuration
  * We can place a honeypod without exposing any services such that only neigboring pods can access the honeypod
* Exposed Service (Simulated)
  * We expose an inetsim HTTP service to the cluster such that it can be found via ClusterIP or DNS lookup
* Exposed Service (Nginx)
  * We expose an Nginx HTTP service to the cluster such that it can be found via ClusterIP or DNS lookup
* Vulnerable Service (Mysql)
  * We expose an empty Mysql service to the cluster such that it can be found via ClusterIP or DNS lookup

**Other Mitigations:**
* Proper Namespacing, RBAC
* Network Policy for VPC subnet and external
* Non-root and not use privileged pod access unless absolutely needed
* Barebone OS Container Image


