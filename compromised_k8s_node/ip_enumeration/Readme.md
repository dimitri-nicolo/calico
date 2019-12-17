IP Enumeration Detection (Node)

Have a bare or empty pod on our tigera-internal(honeypod) namespace. By not having any ports exposed via NodePort/Services, the only way for this honeypod to be contacted is for a pod to be within the same Node and same subnet (192.168.X.0/32). We tighten the access even further by using a network policy to block all traffic into the honeypod. The only way for this honeypod to be access is by the running node. This enhance our detection to indicate that our Node is compromised and is enumerating the internal network.

Detection:
If the running node attempts to connect to the honeypod (other than healthchecks) we generate an alert.

* Pros
** Easy to setup
** No application exposed
* Cons
** Lack of information about the attack and attacker
** Bugged at the moment, see ticket


Customizations:
We can modify this Deployment manifest to a Daemonset so that every Node will have an instance of this honeypod.

