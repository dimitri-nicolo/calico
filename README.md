# calico-k8sapiserver

k8s styled API server to interact with Calico resources.

To deploy, bring up k8s >=1.6, preferably 1.7 since it comes with built-in aggregator.

Also, 1.7 is when the apiregistration.k8s.io api version goes beta.

## Sample installation steps with kubeadm with Calico in ETCD mode

`sudo make kubeadm` automates steps 1-7 of this, if kubeadm is installed and the
docker image (`make tigera/cnx-apiserver`) has been built.
```
1. kubeadm reset; rm -rf /var/etcd
2. KUBE_HYPERKUBE_IMAGE=gcr.io/google_containers/hyperkube-amd64:v1.7.0 kubeadm init --config kubeadm.yaml
   Make sure to setup proxy-client certs. Refer artifacts/misc/kubeadm.yaml
   Example: proxy-client-cert-file: "/etc/kubernetes/pki/front-proxy-client.crt"
            proxy-client-key-file: "/etc/kubernetes/pki/front-proxy-client.key"
3. sudo cp /etc/kubernetes/admin.conf $HOME/
   sudo chown $(id -u):$(id -g) $HOME/admin.conf
   export KUBECONFIG=$HOME/admin.conf
4. kubectl apply -f artifacts/misc/calico.yaml (this one has calico bringing up etcd 3.x backend)
5. kubectl taint nodes --all node-role.kubernetes.io/master-
6. kubectl create namespace calico
7. kubectl create -f artifacts/example/ <-- The set of manifests necessary to install Aggregated API Server
   Prior to this, make sure you have checked out calico-k8sapiserver and have run make clean;make
   This will create the docker image needed by the example/rc.yaml
   OR docker tar can be found in: https://drive.google.com/open?id=0B1QYlddBYM-ZWkoxVWNfcFJtbUU
   docker load -i calico-k8sapiserver-latest.tar.xz
8. kubectl create -f artifacts/policies/policy.yaml <-- Creating a NetworkPolicy
9. kubectl create -f artifacts/policies/tier.yaml <-- Creating a Tier
10. kubectl create -f artifacts/policies/globalpolicy.yaml <-- Creating a GlobalNetworkPolicy
.
.
.
```

## Sample installation steps with kubeadm with Calico in KDD mode
```
1. kubeadm reset; rm -rf /var/etcd
2. KUBE_HYPERKUBE_IMAGE=gcr.io/google_containers/hyperkube-amd64:v1.7.0 kubeadm init --config kubeadm.yaml
   Make sure to setup proxy-client certs. Refer artifacts/misc/kubeadm.yaml
   Example: proxy-client-cert-file: "/etc/kubernetes/pki/front-proxy-client.crt"
            proxy-client-key-file: "/etc/kubernetes/pki/front-proxy-client.key"
3. sudo cp /etc/kubernetes/admin.conf $HOME/
   sudo chown $(id -u):$(id -g) $HOME/admin.conf
   export KUBECONFIG=$HOME/admin.conf
4a. kubectl apply -f artifacts/misc/rbac-kdd.yaml
4b. kubectl apply -f artifacts/misc/kdd_calico.yaml
5. kubectl taint nodes --all node-role.kubernetes.io/master-
6. kubectl create namespace calico
7a. kubectl create -f artifacts/example/
7b. kubectl create -f artifacts/example_kdd
8. kubectl create -f artifacts/policies/policy.yaml <-- Creating a NetworkPolicy
9. kubectl create -f artifacts/policies/tier.yaml <-- Creating a Tier
10. kubectl create -f artifacts/policies/globalpolicy.yaml <-- Creating a GlobalNetworkPolicy
.
.
.
```

## Cleanup and Reset
```
1. kubectl delete -f ~/go/src/github.com/tigera/calico-k8sapiserver/artifacts/example/
2. kubectl delete -f http://docs.projectcalico.org/v2.1/getting-started/kubernetes/installation/hosted/kubeadm/1.6/calico.yaml
3. rm -rf /var/etcd/
4. Reload/Rebuild the new latest docker image for calico-k8sapiserver
5. kubectl apply -f http://docs.projectcalico.org/v2.1/getting-started/kubernetes/installation/hosted/kubeadm/1.6/calico.yaml
6. kubectl create -f ~/go/src/github.com/tigera/calico-k8sapiserver/artifacts/example/
```

## Cluster Access/Authentication

Kubernetes natively supports various Authentication strategies like:
Client Certificates
Bearer Tokens
Authenticating proxy
HTTP Basic Auth

An aggregated API Server will be able to delegate authentication, of any incoming request, to the master API Server/Aggregator.

The Authentication strategy being setup as part of the above demo installation process (through kubeadm) is based on Client Certificates/PKI.

In order to make curl requests against the Aggergated API, following steps can be followed:

1. Open /etc/kubernetes/admin.conf
The file contains the bits of information that kubectl uses in order to make authorized requests against the API Server.

The file container client-certficate and client-key in base64 encoded format.

Copy the value of client-certificate-data in a file , say client-crt.
Run `base64 decode client-crt > client.crt`

Copy the value of client-key-data in a file, say client-key
Run `base64 decode client-key > client.key`

The curl command expects the client certificate to be presented in PEM format.

Generate the PEM file using the command:
`cat client.crt client.key > client.includesprivatekey.pem`

OR

use the helper script artifacts/misc/admin_conf_parser.py to generate /var/tmp/client.includesprivatekey.pem and use it in the
argument to the curl.

2. Find the API Server Certificate Authority info. This is used to verify the certificate response coming in from the Server.

The CA can be found under /etc/kubernetes/pki/apiserver.crt

3. Run the curl command against a K8s resource:

`curl --cacert /etc/kubernetes/pki/apiserver.crt --cert-type pem --cert client.includesprivatekey.pem https://10.0.2.15:6443/api/v1/nodes`

The API Server address can be found from the above admin.conf file as well.

The API Server command/flags used for running can be found under /etc/kubernetes/manifest/

## API Examples
```
Follows native Kubernetes REST semantics.
Tiers - APIVersion: projectcalico.org/v3 Kind: Tier
1. Listing Tiers: https://10.0.2.15:6443/apis/projectcalico.org/v3/tiers
2. Getting a Tier: https://10.0.2.15:6443/apis/projectcalico.org/v3/tiers/net-sec
3. Posting a Tier: -XPOST -d @tier.yaml  -H "Content-type:application/yaml"  https://10.0.2.15:6443/apis/projectcalico.org/v3/tiers

NetworkPolicies - APIVersion: projectcalico.org/v3 Kind: NetworkPolicy
4. Listing networkpolicies across namespaces: https://10.0.2.15:6443/apis/projectcalico.org/v3/networkpolicies
5. Listing networkpolicy from a given namespace (belonging to default tier): https://10.0.2.15:6443/apis/projectcalico.org/v3/namespaces/default/networkpolicies 
^ NOTE: NetworkPolicy list will also include Core NetworkPolicies. Core NetworkPolicy names will be prepended with "knp."
^ NOTE: When fieldSelector not specified it defaults to "default" on the server side.
6. Watching networkpolicies in the default namespace: https://10.0.2.15:6443/apis/projectcalico.org/v3/namespaces/default/networkpolicies?watch=true
7. Selecting networkpolicies in the default namespace belonging to "net-sec": https://10.0.2.15:6443/apis/projectcalico.org/v3/namespaces/default/networkpolicies?fieldSelector=spec.tier=net-sec
8. Select networkpolicies based on Tier and watch at the same time: https://10.0.2.15:6443/apis/projectcalico.org/v3/namespaces/default/networkpolicies?watch=true&fieldSelector=spec.tier=net-sec
9. Create networkpolicies: -XPOST -d @policy.yaml -H "Content-type:application/yaml" https://10.0.2.15:6443/apis/projectcalico.org/v3/namespaces/default/networkpolicies

GlobalNetworkPolicies - APIVersion: projectcalico.org/v3 Kind: GlobalNetworkPolicy
10. Listing globalnetworkpolicies: https://10.0.2.15:6443/apis/projectcalico.org/v3/globalnetworkpolicies
11. Watching globalnetworkpolicies belonging to default tier: https://10.0.2.15:6443/apis/projectcalico.org/v3/globalnetworkpolicies?watch=true
12. Selecting globalnetworkpolicies belonging to Tier1: https://10.0.2.15:6443/apis/projectcalico.org/v3/globalnetworkpolicies?fieldSelector=spec.tier=Tier1
13. Create globalnetworkpolicies: -XPOST -d @policy.yaml -H "Content-type:application/yaml" https://10.0.2.15:6443/apis/projectcalico.org/v3/globalnetworkpolicies

Core/K8s NetworkPolicies - APIVersion: networking.k8s.io/v1 Kind: NetworkPolicy
14. Create core networkpolicies: -XPOST -d @policy.yaml -H "Content-type:application/yaml" https://10.0.2.15:6443/apis/networking.k8s.io/v1/networkpolicies
NOTE: Use above endpoint for CREATE, UPDATE and DELETE on core networkpolicies.

Listing Namespaces - APIVersion: v1 Kind: Namespace
15. List K8s Namespaces:https://10.0.2.15:6443/api/v1/namespaces
``` 

## Testing
The integration tests/functional verification tests can be run via the `fv`/`fv-kdd` Makefile target,
e.g.:

    $ make fv

The unit tests can be run via `ut` Makefile target,
e.g.:

    $ make ut

All of the tests can be run via `test` Makefile target,
e.g.:

    $ make test

