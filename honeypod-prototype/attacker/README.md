# Attacker Pod

The Attacker Pod is used to test the functionality of the Honeypod. This is done by running nmap in a loop, scanning the pod's subnet and probing all ports on reachable pods. The pod will then curl the exposed service `tigera-dashboard-internal-service.tigera-internal.svc.cluster.local`, `tigera-dashboard-internal-debug.tigera-internal.svc.cluster.local`, `tigera-internal-backend.tigera-internal.svc.cluster.local` and run a nmap MySQL brute-forcer on `tigera-internal-backend.tigera-internal.svc.cluster.local`. These will trigger the Alerts for Honeypod deployments.

**This container should not be included into a Honeypod deployment**.

## Installation

* `kubectl create secret -n tigera-internal generic tigera-pull-secret --from-file=.dockerconfigjson=<PATH/TO/PULL/SECRET> -n default --type=kubernetes.io/dockerconfigjson`.
* Apply image to a cluster: `kubectl apply -f attacker.yaml`.

### Contents

* curl
* dig
* nmap
