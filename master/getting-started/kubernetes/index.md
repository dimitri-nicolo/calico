---
title: Quickstart for Tigera CNX on Kubernetes
---


### Overview

This quickstart gets you a single-host Kubernetes cluster with {{site.prodname}}
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

   > **Note**: If you are on a multi-node cluster, you will need to join all of the non-master nodes to the cluster using the `kubeadm join` command. Scroll upward in your terminal to locate the `join` command as returned by `kubeadm init`. Copy and paste it in each of the node's shell prompt.
   {: .alert .alert-info}

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

   > **Note**: At this point, you should have a fully functional kubernetes cluster with calico. Confirm that you now have the node(s) `STATUS` as `Ready` using the
   following command: `kubectl get nodes`
   {: .alert .alert-info}

   Now, continuing onto setting up the CNX manager(ui) and the CNX apiserver.
  
1. Considering its a single node cluster, remove the taints on the master so that pods can be scheduled on it.

   ```
   kubectl taint nodes --all node-role.kubernetes.io/master-
   ```

   It should return the following.

   ```
   node "<your-hostname>" untainted
   ```

1. Generate TLS credentials - i.e. a web server certificate and key - for the
   CNX Manager.

   See
   [Certificates](https://kubernetes.io/docs/concepts/cluster-administration/certificates/)
   for various ways of generating TLS credentials.  As both its Common Name and
   a Subject Alternative Name, the certificate must have the host name (or IP
   address) that browsers will use to access the CNX Manager.  In a single-node
   test deployment this can be just `127.0.0.1`, but in a real deployment it
   should be a planned host name that maps to the `cnx-manager` Service.

   For the sake of this quick start, i.e. just to see CNX working, you can use
   [this test
   certificate]({{site.baseurl}}/{{page.version}}/getting-started/kubernetes/installation/hosted/essentials/demo-manifests/cert)
   and
   [key]({{site.baseurl}}/{{page.version}}/getting-started/kubernetes/installation/hosted/essentials/demo-manifests/key).

1. Store those credentials as `cert` and `key` in a Secret named
   `cnx-manager-tls`.  For example:

   ```
   kubectl create secret generic cnx-manager-tls --from-file=cert=/path/to/certificate --from-file=key=/path/to/key -n kube-system
   ```

1. [Download the `calico-cnx.yaml` manifest]({{site.baseurl}}/{{page.version}}/getting-started/kubernetes/installation/hosted/essentials/demo-manifests/calico-cnx.yaml).

1. As with the `calico.yaml` manifest, run the following command to remove `<YOUR_PRIVATE_DOCKER_REGISTRY>` from the path to the `cnx-node` image.

   ```
   sed -i -e 's/<YOUR_PRIVATE_DOCKER_REGISTRY>\///g' calico-cnx.yaml
   ```

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
   kube-system   cnx-node-mxzhw                             2/2     Running   0          1h
   kube-system   cnx-apiserver-8kmm8                        1/1     Running   0          1h
   kube-system   etcd-<hostname>                            1/1     Running   0          1h
   kube-system   kube-apiserver-<hostname>                  1/1     Running   0          1h
   kube-system   kube-controller-manager-<hostname>         1/1     Running   0          1h
   kube-system   kube-dns-545bc4bfd4-gxhpv                  3/3     Running   0          1h
   kube-system   kube-proxy-z5vq9                           1/1     Running   0          1h
   kube-system   kube-scheduler-<hostname>                  1/1     Running   0          1h
   kube-system   cnx-manager-558d896894-zvpmc               1/1     Running   0          1h
   ```
   Press CTRL+C to exit `watch`.

1. Setup a basic-auth user for the purposes of logging in and then navigating the UI.
   
   Create the csv file and register `jane` as the user with the Kubernetes cluster. Each line is 'password,user-name,user-id'
   ```
   echo 'welc0me,jane,1' > /etc/kubernetes/pki/basic_auth.csv
   ```
   Now, lets set the path in kube-apiserver.yaml
   ```
   sed -i "/- kube-apiserver/a\    - --basic-auth-file=/etc/kubernetes/pki/basic_auth.csv" \
   /etc/kubernetes/manifests/kube-apiserver.yaml
   ```

   >**Note: We created the basic_auth.csv under /etc/kubernetes/pki since that volume is mounted by default on the kube-apiserver pod with a kubeadm installation.
   {: .alert .alert-info}

1. Set up CORS on the API Server to allow API access from the UI
   ```
   sed -i "/- kube-apiserver/a\    - --cors-allowed-origins=\"https://*\"" \
   /etc/kubernetes/manifests/kube-apiserver.yaml
   ```

1. You should now be able to log in as Jane, but won't yet be able to see or edit resources. Bind `jane` with the `cluster-admin` role to give full access.
   ```
   kubectl create clusterrolebinding permissive-binding \
   --clusterrole=cluster-admin \
   --user=jane
   ```

Congratulations! You now have a single-host Kubernetes cluster
equipped with {{site.prodname}}.

To access the {{site.prodname}} Manager web interface, navigate to `https://127.0.0.1:30003`.
>**Note: As part of these instruction we are dealing with self-signed certificates for both the cnx-manager and the cnx-apiserver. So, you will need to proceed explicitly from the browser. If on Chrome and things dont go as planned, take a look at the Inspect logs to get tip-offs.
Example: The case of `Failed to fetch namespaces` while navigating through Chrome. If you open up the `Console` tab under Chrome Inspect (right-click Inspect) window, you may see `net::ERR_INSECURE_RESPONSE`. Click on the api link associated to the message and explicitly proceed ahead.
{: .alert .alert-info}

### Next steps
**[Experiment with OIDC authentication strategy] (WIP add the link with detailed instructions)**

**[Experiment with RBAC and the {{site.prodname}} Manager web interface]({{site.baseurl}}/{{page.version}}/reference/essentials/rbac-tiered-policies)**

**[Secure a simple two-tier application using the Kubernetes `NetworkPolicy` API](tutorials/simple-policy)**

**[Create a policy using more advanced policy features](tutorials/advanced-policy)**

**[Using the calicoctl CLI tool](https://docs.projectcalico.org/master/getting-started/kubernetes/tutorials/using-calicoctl)**
