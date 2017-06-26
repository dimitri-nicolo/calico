# calico-k8sapiserver

k8s styled API server to interact with Calico resources.

To deploy, bring up k8s >=1.6, preferably 1.7 (for now beta.1)

## Sample installation steps with kubeadm
```
1. kubeadm reset; rm -rf /var/etcd
2. KUBE_HYPERKUBE_IMAGE=gcr.io/google_containers/hyperkube-amd64:v1.7.0-beta.1 kubeadm init --config kubeadm.yaml
   Make sure to setup proxy-client certs. Refer artifacts/misc/kubeadm.yaml
   Example: proxy-client-cert-file: "/etc/kubernetes/pki/front-proxy-client.crt"
            proxy-client-key-file: "/etc/kubernetes/pki/front-proxy-client.key"
3. sudo cp /etc/kubernetes/admin.conf $HOME/
   sudo chown $(id -u):$(id -g) $HOME/admin.conf
   export KUBECONFIG=$HOME/admin.conf
4. kubectl apply -f http://docs.projectcalico.org/v2.1/getting-started/kubernetes/installation/hosted/kubeadm/1.6/calico.yaml
5. kubectl taint nodes --all node-role.kubernetes.io/master-
6. kubectl create namespace calico
7. kubectl create -f artifacts/example/ <-- The set of manifests necessary to install Aggregated API Server
   Prior to this, make sure you have checked out calico-k8sapiserver and have run make clean;make
   This will create the docker image needed by the example/rc.yaml
   Currently tested to be stable at 4ba7b5c1d394b5f94cfd5f2ef4f6d467cff4051b
8. kubectl create -f artifacts/policies/policy.yaml <-- Creating a Policy
9. kubectl create -f artifacts/policies/tier.yaml <-- Creating a Tier
.
.
.
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

2. Find the API Server Certificate Authority info. This is used to verify the certificate response coming in from the Server.

The CA can be found under /etc/kubernetes/pki/apiserver.crt

3. Run the curl command against a K8s resource:

`curl --cacert /etc/kubernetes/pki/apiserver.crt --cert-type pem --cert client.includesprivatekey.pem https://10.0.2.15:6443/api/v1/nodes`

The API Server address can be found from the above admin.conf file as well.

The API Server command/flags used for running can be found under /etc/kubernetes/manifest/
