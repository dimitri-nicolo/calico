# Scenario: Compromised K8s Node
This scenario covers the situation where the attacker is able to compromise a Node within a K8s cluster. This Node can be a worker node, infrastructure node, or master node in increasing severity. We assume there’s no network policy applied.

**The attacker is able to:**
* Reach all running pods and nodes within the K8s (Not via Kubectl)
  * Does not matter if these pods are exposed via a service or not
* Reach all other K8s Nodes/VMs/Hosts within the VPC subnet
* Access all mounted/available cloud resources (ElasticSearch local data, storage)
* Get Token, Metadata, Credentials
* Access to Containerization Engine (Docker)
  * Pull/Run/Exec into containers(K8s Pods)
* [Root] Alter underlying OS settings
* [Root] Access all containers data
* [Root] Basically Game Over

**Honeypod:**
* IP Enumuration

**Mitigations:**
* Firewall and Network Policy between K8s
* Barebone OS running K8s
* Unique certificate SSH access, rotated often
* Block metadata service (Maybe no, DNS IP is also 169.254.169.254 here)

