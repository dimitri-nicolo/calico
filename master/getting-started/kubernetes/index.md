---
title: Quickstart for Tigera CNX on Kubernetes
---


### Overview

This quickstart gets you a single-host Kubernetes cluster with {{site.prodname}} 
in approximately 30 minutes. You can use this cluster for testing and development.

To deploy a cluster suitable for production, refer to [Installation](https://docs.projectcalico.org/master/getting-started/kubernetes/installation/).


### Requirements

- AMD64 processor
- 2CPU
- 4GB RAM
- 10GB free disk space
- RedHat Enterprise Linux 7.x+, CentOS 7.x+, Ubuntu 16.04+, or Debian 8.x+
- A Google account for login


### Before you begin

[Follow the Kubernetes instructions to install kubeadm](https://kubernetes.io/docs/setup/independent/install-kubeadm/){:target="_blank"}.

> **Note**: After installing kubeadm, do not power down or restart
the host. Instead, continue directly to the 
[next section to create your cluster](#create-a-single-host-kubernetes-cluster).
{: .alert .alert-info}


### Create a single-host Kubernetes cluster

1. As a regular user with sudo privileges, open a terminal on the host that 
   you installed kubeadm on. 

1. Download the {{site.prodname}} images and add them to Docker.  Obtain the
   archives containing the images from your support representative, and then
   run the following commands to load them.

   ```
   docker load -i tigera_cnx-apiserver_{{site.data.versions[page.version].first.components["cnx-apiserver"].version}}.tar.xz
   docker load -i tigera_cnx-node_{{site.data.versions[page.version].first.components["cnx-node"].version}}.tar.xz
   docker load -i tigera_cnx-manager_{{site.data.versions[page.version].first.components["cnx-manager"].version}}.tar.xz
   ```

1. Update your package definitions and upgrade your existing packages.

   ```
   sudo apt-get update && sudo apt-get upgrade
   ```
   
1. [Create a Google project to use to login to CNX Manager](https://developers.google.com/identity/protocols/OpenIDConnect){:target="_blank"}.
   Set the redirect URIs to `http://127.0.0.1:30003` and `https://127.0.0.1:30003`.
   
1. Copy the OAuth client ID value.

1. Download the [kubeadm.yaml]({{site.baseurl}}/{{page.version}}/getting-started/kubernetes/installation/hosted/essentials/demo-manifests/kubeadm.yaml) file.

1. Open the kubeadm.yaml file in your favorite editor, replace `<fill-in-your-oauth-client-id-here>` 
   with the OAuth client ID value, then save and close the file.

1. Initialize the master using the following command.

   ```
   sudo kubeadm init --config kubeadm.yaml
   ```
   
1. Execute the following commands to configure kubectl (also returned by
   `kubeadm init`).

   ```
   mkdir -p $HOME/.kube
   sudo cp -i /etc/kubernetes/admin.conf $HOME/.kube/config
   sudo chown $(id -u):$(id -g) $HOME/.kube/config
   ```

1. Download the [`calico.yaml` manifest]({{site.baseurl}}/{{page.version}}/getting-started/kubernetes/installation/hosted/kubeadm/1.7/calico.yaml).

1. Since you have loaded the private {{site.prodname}} images locally, run the following command to remove `<YOUR_PRIVATE_DOCKER_REGISTRY>` from the path to the `cnx-node` image.

   ```
   sed -i -e 's/<YOUR_PRIVATE_DOCKER_REGISTRY>\///g' calico.yaml
   ```

1. Use the following command to apply the manifest.

   ```
   kubectl apply -f calico.yaml
   ```
   
   You should see the following output.

   ```
   configmap "calico-config" created
   daemonset "calico-etcd" created
   service "calico-etcd" created
   daemonset "calico-node" created
   deployment "calico-kube-controllers" created
   deployment "calico-policy-controller" created
   clusterrolebinding "calico-cni-plugin" created
   clusterrole "calico-cni-plugin" created
   serviceaccount "calico-cni-plugin" created
   clusterrolebinding "calico-kube-controllers" created
   clusterrole "calico-kube-controllers" created
   serviceaccount "calico-kube-controllers" created
   ```

1. [Download the `calico-cnx.yaml` manifest]({{site.baseurl}}/{{page.version}}/getting-started/kubernetes/installation/hosted/essentials/demo-manifests/calico-cnx.yaml).


1. As with the `calico.yaml` manifest, run the following command to remove `<YOUR_PRIVATE_DOCKER_REGISTRY>` from the path to the `cnx-node` image.

   ```
   sed -i -e 's/<YOUR_PRIVATE_DOCKER_REGISTRY>\///g' calico-cnx.yaml
   ```
   
1. Open `calico-cnx.yaml` in your favorite editor, replace `<fill-in-your-oauth-client-id-here>` with your OAuth client ID, save the file, and exit the editor. 

1. Use the following command to install the additional {{site.prodname}} components.

   ```
   kubectl apply -f calico-cnx.yaml
   ```
 
   You should see the following output.

   ```
   configmap "tigera-cnx-manager-config" created
   apiservice "v3.projectcalico.org" created
   clusterrolebinding "calico:system:auth-delegator" created
   rolebinding "calico-auth-reader" created
   replicationcontroller "cnx-apiserver" created
   serviceaccount "cnx-apiserver" created
   service "api" created
   deployment "tigera-cnx-manager" created
   service "tigera-cnx-manager" created
   ```
   
1. Remove the taints on the master so that pods can be scheduled on it.
   
   ```
   kubectl taint nodes --all node-role.kubernetes.io/master-
   ```

   It should return the following.

   ```
   node "<your-hostname>" untainted
   ```
   
1. Confirm that all of the pods are running with the following command.
   Some can only start after others, so it's OK to see a few restarts.

   ```
   watch kubectl get pods --all-namespaces
   ```
   
   Wait until each pod has the `STATUS` of `Running`.

   ```
   NAMESPACE     NAME                                       READY   STATUS    RESTARTS   AGE
   kube-system   calico-etcd-q4fcf                          1/1     Running   0          1h
   kube-system   calico-kube-controllers-797946f9d9-jrwx9   1/1     Running   0          1h
   kube-system   calico-node-mxzhw                          2/2     Running   0          1h
   kube-system   calico-server-8kmm8                        1/1     Running   0          1h
   kube-system   etcd-karen-virtualbox                      1/1     Running   0          1h
   kube-system   kube-apiserver-karen-virtualbox            1/1     Running   0          1h
   kube-system   kube-controller-manager-karen-virtualbox   1/1     Running   0          1h
   kube-system   kube-dns-545bc4bfd4-gxhpv                  3/3     Running   0          1h
   kube-system   kube-proxy-z5vq9                           1/1     Running   0          1h
   kube-system   kube-scheduler-karen-virtualbox            1/1     Running   0          1h
   kube-system   tigera-cnx-manager-web-558d896894-zvpmc    1/1     Running   0          1h
   ```

1. Press CTRL+C to exit `watch`.

1. Switch to a root shell.

   ```
   sudo -i
   ```

1. Scroll upward in your terminal to locate the `join` command
   returned by `kubeadm init`. Copy the `join` command, paste it
   in your shell prompt, and add `--skip-preflight-checks` to the end.
   
   **Syntax**:
   ```
   kubeadm join --token <token> <master-ip>:<master-port> \
   --discovery-token-ca-cert-hash sha256:<hash> \
   --skip-preflight-checks
   ```
   
   **Example**:
   ```
   kubeadm join --token eea8bd.4d282767b6b962ca 10.0.2.15:6443 \
   --discovery-token-ca-cert-hash sha256:0e6e73d52066326023432f417a566afad72667e6111d2236b69956b658773255
   --skip-preflight-checks
   ```
   
1. Exit the root shell.

   ```
   exit
   ```

1. Confirm that you now have a node in your cluster with the 
   following command.
   
   ```
   kubectl get nodes -o wide
   ```
   
   It should return something like the following.
   
   ```
   NAME             STATUS  ROLES   AGE  VERSION  EXTERNAL-IP  OS-IMAGE            KERNEL-VERSION     CONTAINER-RUNTIME
   <your-hostname>  Ready   master  1h   v1.8.x   <none>       Ubuntu 16.04.3 LTS  4.10.0-28-generic  docker://1.12.6
   ```
   
Congratulations! You now have a single-host Kubernetes cluster
equipped with {{site.prodname}}.

To access the {{site.prodname}} Manager web interface, navigate to `https://127.0.0.1:30003`.
If you're not running this demo on the same system you'll log in from,
instead run `kubectl proxy --port=8080` to provide the web application
with access to the Kubernetes API, and navigate to the domain name of
the system instead (still on port 30003).

You should be able to log in , but won't yet be able to see or edit resources.
To create some RBAC roles that allow full access to everyone, apply [this manifest]({{site.baseurl}}/{{page.version}}/getting-started/kubernetes/installation/hosted/essentials/demo-manifests/rbac-all.yaml).


### Next steps

**[Experiment with RBAC and the {{site.prodname}} Manager web interface]({{site.baseurl}}/{{page.version}}/reference/essentials/rbac-tiered-policies)**

**[Secure a simple two-tier application using the Kubernetes `NetworkPolicy` API](tutorials/simple-policy)**

**[Create a policy using more advanced policy features](tutorials/advanced-policy)**

**[Using the calicoctl CLI tool](https://docs.projectcalico.org/master/getting-started/kubernetes/tutorials/using-calicoctl)**
