# Honeypod
The basic idea of Honeypod is to place canary pods or resources within a K8s cluster such that all defined and valid resources will never attempt to access or make connections to the Honeypods. If any resources reaches these Honeypods, we can automatically assume the connection is suspicious at minimum and that the source resource may be been compromised.
Honeypod may be used to detect resources enumeration, privilege escalation, data exfiltration, denial of service and vulnerability exploitation attempts. 


## Default Naming
* Pod:  “tigera-internal-\*”, where \* is the type number
* Namespace: “tigera-internal”
* Exposed services: 
** (Vulnerable) “tigera-dashboard-internal-debug”
** (Unreachable) “tigera-dashboard-internal-service”
* Global Network Policies: “tigera-internal-\*”
* Tier: “tigera-internal”
* Clusterolebinding: tigera-internal-binding
* Clusterole: tigera-internal-role

## Installation
0. Ensure Calico Enterprise version 2.6+ is installed
1. Kubectl apply -f common/\*.yaml
2. Navigate to relevant detection folder and apply the YAMLs (Modify Naming if needed)
3. To test, use the 'attacker' pod to create an entrypoint to the cluster (You will need to push pod into registry)
