# Attacker Pod
The Attacker Pod is used to test the functionality of the Honeypod. This is done by running nmap in a loop, scanning the pod's subnet and probing all ports on reachable pods. The pod will then curl the exposed service "tigera-dashboard-internal-service.tigera-internal.svc.cluster.local", "tigera-dashboard-internal-debug.tigera-internal.svc.cluster.local", "tigera-internal-backend.tigera-internal.svc.cluster.local" and run a nmap Mysql bruteforcer on "tigera-internal-backend.tigera-internal.svc.cluster.local". These will trigger the Alerts for Honeypod deployments.

**This container should not be included into a Honeypod deployment**

**Manual Installation**
* Navigate to build dir
* Run `docker build -t attacker-pod .`
* Tag image `docker tag attacker-pod gcr.io/tigera-security-research/attacker-pod:1.0.0`
* Push image `docker push gcr.io/tigera-security-research/attacker-pod:1.0.0`
* Apply image to k8s `kubectl apply -f attacker.yaml`

**Installation**
* `kubectl create secret -n tigera-internal generic tigera-pull-secret \
    --from-file=.dockerconfigjson=<PATH/TO/PULL/SECRET> -n default --type=kubernetes.io/dockerconfigjson`
* Apply image to k8s `kubectl apply -f attacker.yaml`

**Contents:**
* Nmap
* Curl
* Dig


