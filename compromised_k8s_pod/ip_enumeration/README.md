# IP Enumeration Detection

Have a bare or empty pod on our tigera-internal(honeypod) namespace. By not having any ports exposed via NodePort/Services, the only way for this honeypod to be contacted is for a pod to be within the same Node and same subnet (192.168.X.0/32). This indicate that the pod is behaving suspisously as no pods should be contacting the honeypod, and theres a good chance that the pod is compromised and is enumerating the network. 

## Detection:
If any pod attempts to connect to the honeypod (other than healthchecks) we generate an alert.

## Alerts:
* `[Honeypod] Pod Subnet IP Enumeration by ${source_namespace}/${source_name_aggr}`
  * Detection of a Pod possibly running a IP scan on its subnet and reaching the Honeypod
* `[Honeypod] Pod Subnet Port Scan by ${source_namespace}/${source_name_aggr}`
  * Detection of a Pod possibly running a port scan on the Honeypod

**Pros**
* Easy to setup
* No application exposed

**Cons**
* Lack of information about the attack and attacker


## Customizations:
We can modify this Deployment manifest to a Daemonset so that every Node will have an instance of this honeypod.
We can add Network Policy to focus only on specific pods (frontend, public facing) 

