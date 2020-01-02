# Vulnerable Service (mysql)

Have a mysql container serving an empty database on port 3306. The database contains a root account with no password. A services is then applied to expose port 3306 to the honeypod. This will generate an entry into our DNS record: tigera-internal-backend.tigera-internal.svc.cluster.local(3306). This attempts to provide a target for the attacker to try and gain access. Since no pods should be accessing these services, we can indicate that a pod has been compromised and is attempting to move latterally.

## Detection:
If anyone talks to it (other than healthchecks) we create an alert. 

## Alert:
* [Honeypod] Vulnerable Service (FTP) accessed by $\{source\_namespace\}\/$\{source\_name\_aggr\}
  * Detection of a Pod reaching our Honeypod FTP service

**Pros**
* Easy to setup 

**Cons** 
* Only provide generic response
* Cannot determine if request is malicious


## Customizations:
* place honeypod into target namespace instead of our honeypod service/namespace


