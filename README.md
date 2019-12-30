# Honeypod
The basic idea of Honeypod is to place canary pods or resources within a K8s cluster such that all defined and valid resources will never attempt to access or make connections to the Honeypods. If any resources reaches these Honeypods, we can automatically assume the connection is suspicious at minimum and that the source resource may be been compromised.

Honeypod may be used to detect resources enumeration, privilege escalation, data exfiltration, denial of service and vulnerability exploitation attempts. 


## Default Naming
* Pod:  “tigera-internal-\*”, where \* is the type number
* Namespace: “tigera-internal”
* Exposed services: 
  * (Vulnerable) “tigera-dashboard-internal-debug”
  * (Unreachable) “tigera-dashboard-internal-service”
* Global Network Policies: “tigera-internal-\*”
* Tier: “tigera-internal”
* Clusterolebinding: tigera-internal-binding
* Clusterole: tigera-internal-role

## Demo Installation (only compromised pod scenarios)
0. Ensure Calico Enterprise version 2.6+ is installed
1. Kubectl apply -f honeypod\_sample\_setup.yaml

## Installation
0. Ensure Calico Enterprise version 2.6+ is installed
1. Kubectl apply -f common/common.yaml
2. Navigate to relevant scenarios folder and apply the YAMLs (Modify naming if needed)

## Testing
To test, run the 'attacker' pod on the k8s cluster, the pod will periodically nmap/scan the network 

## Scenarios
We currently have several Honeypod deployments that can be used to detect different scenarios:

### Compromised Pod
* IP Enumeration
  * By not setting a service, the pod can only be reach locally (adjecant pods within same subnet).
* Exposed Service (inetsim)
  * Setup a simulator service pod that is exposed as a HTTP service. The pod can be discovered via clusterip or DNS lookup.
* Exposed Service (nginx)
  * Expose a nginx service that serves a generic page. The pod can be discovered via clusterip or DNS lookup.
* Vulnerable Service (mysql)
  * Expose a SQL service that contains an empty database with easy (root, no password) access. The pod can be discovered via clusterip or DNS lookup.

### Compromised Node
* IP Enumeration
  * By not setting a service and adding strict network policy, the pod can only be reach by the hosting Node.
