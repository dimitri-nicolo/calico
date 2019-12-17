# Exposed Service

Have a [inetsim](https://www.inetsim.org/features.html) honeypod and have the pod simulate a HTTP server on port 8888. Two services is then applied to expose port 443 and 8888 to the honeypod. This will generate 2 entries into our DNS record: tigera-dashboard-internal-service.tigera-internal.svc.cluster.local(443) and tigera-dashboard-internal-debug.tigera-internal.svc.cluster.local(8888). This mimics a web application with a secure endpoint (443) and a debug instance. This entice the attacker in trying to connect to tigera-dashboard-internal-debug.tigera-internal.svc.cluster.local(8888). Since no pods should be accessing these services, we can indicate that a pod has been compromised and is attempting to move latterally.

## Detection:
If anyone talks to it (other than healthchecks) we create an alert. 

**Pros**
* Easy to setup (Dockerized already)

**Cons** 
* Only provide generic response
* Cannot determine if request is malicious


## Customizations:
* Inetsim is an internet service simulation application for SSH/HTTP/FTP/SMTP/POP/DNS and more. When contacted, the service will return a generic response for the requestor’s protocol.
* Target an actual running service instead of our honeypod service/namespace

