# Attacker Pod
The Attacker Pod is used to test the functionality of the Honeypod. This is done by running nmap in a loop, scanning the pod's subnet and probing all ports on reachable pods.

**This container should not be included into a Honeypod deployment**

**Installation**
* Navigate to build dir
* Run 'docker build -t attacker-pod .'
* Tag image 'docker tag attacker-pod gcr.io/tigera-security-research/attacker-pod:1.0.0'
* Push image 'docker push gcr.io/tigera-security-research/attacker-pod:1.0.0'
* Apply image to k8s 'kubectl apply -f attacker.yaml'

**Contents:**
* Nmap
* Curl
* Dig


