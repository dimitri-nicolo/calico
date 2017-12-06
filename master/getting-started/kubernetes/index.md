---
title: Quickstart for Tigera CNX on Kubernetes
---


### Overview

This quickstart gets you a single-host Kubernetes cluster with {{site.prodname}} (with Calico in etcd mode)
in approximately 30 minutes.  You must run the cluster on a linux system with a
web browser (possibly a VM).  You can use this cluster for testing and development.

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

1. Initialize the master using the following command and configure kubectl.

   ```
   sudo kubeadm init --pod-network-cidr 192.168.0.0/16
   ```
   Execute the commands to configure kubectl as returned by
   `kubeadm init`. Most likely they will be as follows:
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
   daemonset "cnx-node" created
   deployment "calico-kube-controllers" created
   deployment "calico-policy-controller" created
   clusterrolebinding "calico-cni-plugin" created
   clusterrole "calico-cni-plugin" created
   serviceaccount "calico-cni-plugin" created
   clusterrolebinding "calico-kube-controllers" created
   clusterrole "calico-kube-controllers" created
   serviceaccount "calico-kube-controllers" created
   ```

1. Considering its a single node cluster, remove the taints on the master so that pods can be scheduled on it.

   ```
   kubectl taint nodes --all node-role.kubernetes.io/master-
   ```

   It should return the following.

   ```
   node "<your-hostname>" untainted
   ```

   > **Note**: At this point, you should have a fully functional kubernetes cluster with calico. Confirm that you now have the node(s) `STATUS` as `Ready` using the command: `kubectl get nodes`
   {: .alert .alert-info}

   Now, continuing onto setting up the CNX manager(ui) and the CNX apiserver.

1. Set TLS for CNX manager. For simplicity, we will use kubeadm generated kube-apiserver cert and key for cnx-manager as well. 
   Create the tls Secret named `cnx-manager-tls`.
   ```
   kubectl create secret generic cnx-manager-tls --from-file=cert=/etc/kubernetes/pki/apiserver.crt \
   --from-file=key=/etc/kubernetes/pki/apiserver.key -n kube-system
   ```

1. [Download the `cnx-etcd.yaml` manifest]({{site.baseurl}}/{{page.version}}/getting-started/kubernetes/installation/hosted/essentials/1.7/cnx-etcd.yaml).

1. As with the `calico.yaml` manifest, run the following command to remove `<YOUR_PRIVATE_DOCKER_REGISTRY>` from the path to the `cnx-node` image.

   ```
   sed -i -e 's/<YOUR_PRIVATE_DOCKER_REGISTRY>\///g' cnx-etcd.yaml
   ```

1. Use the following command to install the additional {{site.prodname}} components.

   ```
   kubectl apply -f cnx-etcd.yaml
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
   > **Note**: At this point, you should have the CNX services up and running. Confirm that pods have their `STATUS` as `Running` using the command: `watch kubectl get pods --all-namespaces`. Some can only start after others, so it's OK to see a few restarts. Press CTRL+C to exit `watch`.
   {: .alert .alert-info}

1. Setup a basic-auth user to login through the UI.
   
   Create the basic auth csv file to be used by the kube apiserver and lets register `jane` as our user. Each line in the file is of the form 'password,user-name,user-id'
   ```
   echo 'welc0me,jane,1' > /etc/kubernetes/pki/basic_auth.csv
   ```
   Then, lets set the path in kube-apiserver.yaml
   ```
   sed -i "/- kube-apiserver/a\    - --basic-auth-file=/etc/kubernetes/pki/basic_auth.csv" \
   /etc/kubernetes/manifests/kube-apiserver.yaml
   ```

   > **Note**: We created the basic_auth.csv under /etc/kubernetes/pki because that volume is mounted by default on the kube-apiserver pod with a kubeadm installation.
   {: .alert .alert-info}

1. Set up CORS on the kube apiserver to allow API accesses from the UI
   ```
   sed -i "/- kube-apiserver/a\    - --cors-allowed-origins=\"https://*\"" \
   /etc/kubernetes/manifests/kube-apiserver.yaml
   ```

1. You should now be able to log in as `jane`, but won't yet be able to see or edit resources. Bind `jane` with the `cluster-admin` role to give full access to all resources.
   ```
   kubectl create clusterrolebinding permissive-binding \
   --clusterrole=cluster-admin \
   --user=jane
   ```

Congratulations! You now have a single-host Kubernetes cluster
equipped with {{site.prodname}}.

To access the {{site.prodname}} Manager web interface, navigate to `https://127.0.0.1:30003`.
> **Note**: As part of these instruction we are dealing with self-signed certificates for CNX/K8s services. So, you might need to proceed explicitly from the browser. For example, if on Chrome and things do not go as planned, take a look at the Inspect (right-click Inspect) Console logs to get tip-offs.
{: .alert .alert-info}

### Next steps
**[Experiment with OIDC authentication strategy]({{site.baseurl}}/{{page.version}}/reference/essentials/authentication)**

**[Experiment with non-admin users and the web manager]({{site.baseurl}}/{{page.version}}/reference/essentials/non-admin-workflows)**

**[Secure a simple two-tier application using the Kubernetes `NetworkPolicy` API](tutorials/simple-policy)**

**[Create a policy using more advanced policy features](tutorials/advanced-policy)**

**[Using the calicoctl CLI tool](https://docs.projectcalico.org/master/getting-started/kubernetes/tutorials/using-calicoctl)**
