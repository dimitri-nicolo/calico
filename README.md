# calico-k8sapiserver

k8s styled API server to interact with Calico resources.

To deploy, bring up k8s >=1.6

## Sample installation steps with kubeadm
```
1. kubeadm reset; rm -rf /var/etc
2. KUBE_HYPERKUBE_IMAGE=gcr.io/google_containers/hyperkube-amd64:v1.7.0-alpha.1 kubeadm init --config kubeadm.yaml
   Make sure to setup proxy-client certs.
   Example: proxy-client-cert-file: "/etc/kubernetes/pki/front-proxy-client.crt"
            proxy-client-key-file: "/etc/kubernetes/pki/front-proxy-client.key"
3. sudo cp /etc/kubernetes/admin.conf $HOME/
   sudo chown $(id -u):$(id -g) $HOME/admin.conf
   export KUBECONFIG=$HOME/admin.conf
4. kubectl apply -f http://docs.projectcalico.org/v2.1/getting-started/kubernetes/installation/hosted/kubeadm/1.6/calico.yaml
5.  kubectl taint nodes --all node-role.kubernetes.io/master-
6. kubectl create namespace calico
7. kubectl create -f artifacts/example/
8. kubectl create -f artifacts/policies/01-policy.yaml
.
.
.
```
