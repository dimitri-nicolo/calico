---
title: Quickstart for Tigera CNX on Kubernetes
---


### Overview

This quickstart gets you a single-host Kubernetes cluster with {{site.prodname}}
in approximately 30 minutes. You can use this cluster for testing and development.

To deploy a cluster suitable for production, refer to [Installation](/{{page.version}}/getting-started/kubernetes/installation/).


### Requirements

- AMD64 processor
- 2CPU
- 4GB RAM
- 10GB free disk space
- RedHat Enterprise Linux 7.x+, CentOS 7.x+, Ubuntu 16.04+, or Debian 8.x+


### Before you begin

[Follow the Kubernetes instructions to install kubeadm](https://kubernetes.io/docs/setup/independent/install-kubeadm/){:target="_blank"}.

> **Note**: After installing kubeadm, do not power down or restart
the host. Instead, continue directly to the
[next section to create your cluster](#create-a-single-host-kubernetes-cluster).
{: .alert .alert-info}


### Create a single-host Kubernetes cluster

1. Ensure that you have the [`config.json` file with the private Tigera registry credentials](/{{page.version}}/getting-started/#obtain-the-private-registry-credentials).
   
1. As a regular user with sudo privileges, open a terminal on the host that
   you installed kubeadm on.
   
1. Open the `~/.docker/config.json` file in your favorite editor.

   ```bash
   vi ~/.docker/config.json
   ```
   
1. If the file did not already exist, paste in the entire contents of the
   `config.json` file from Tigera. Otherwise, just add an entry for `"quay.io"`
   under `"auths"` and paste in the `"auth"` and `"email"` lines.

1. Save and close the file.

1. Use the following commands to pull the {{site.prodname}} images.

   ```bash
   docker pull {{site.imageNames["cnxApiserver"]}}:{{site.data.versions[page.version].first.title}}
   docker pull {{site.imageNames["cnxQueryserver"]}}:{{site.data.versions[page.version].first.title}}
   docker pull {{site.imageNames["cnxManager"]}}:{{site.data.versions[page.version].first.title}}
   docker pull {{site.imageNames["cnxManagerProxy"]}}:{{site.data.versions[page.version].first.title}}
   docker pull {{site.imageNames["node"]}}:{{site.data.versions[page.version].first.title}}
   ```

1. Initialize the master using the following command.

   ```
   sudo kubeadm init --pod-network-cidr=192.168.0.0/16 --apiserver-cert-extra-sans=127.0.0.1
   ```

1. Execute the commands to configure kubectl as returned by
   `kubeadm init`. Most likely they will be as follows:

   ```
   mkdir -p $HOME/.kube
   sudo cp -i /etc/kubernetes/admin.conf $HOME/.kube/config
   sudo chown $(id -u):$(id -g) $HOME/.kube/config
   ```

1. Use the following command to create a file `basic_auth.csv` containing
   a set of credentials.

   ```
   sudo sh -c "echo 'welc0me,jane,1' > /etc/kubernetes/pki/basic_auth.csv"
   ```

1. Add a reference to the `basic_auth.csv` file in `kube-apiserver.yaml`.

   ```
   sudo sed -i \
   "/- kube-apiserver/a\    - --basic-auth-file=/etc/kubernetes/pki/basic_auth.csv" \
   /etc/kubernetes/manifests/kube-apiserver.yaml
   ```

   > **Note**: We created the basic_auth.csv under /etc/kubernetes/pki because that volume is
   mounted by default on the kube-apiserver pod with a kubeadm installation.
   {: .alert .alert-info}

1. Restart kube-apiserver to pick up new settings:

   ```
   sudo systemctl restart kubelet
   ```

1. Bind `jane` with the `cluster-admin` role so that she can access any
   resources after logging in.

   ```
   kubectl create clusterrolebinding permissive-binding \
   --clusterrole=cluster-admin \
   --user=jane
   ```

   `kubectl` should return the following.

   ```
   clusterrolebinding "permissive-binding" created
   ```

1. Install {{site.prodname}} and a single node etcd with the following command.

   ```
   kubectl apply -f \
   {{site.url}}/{{page.version}}/getting-started/kubernetes/installation/hosted/kubeadm/1.7/calico.yaml
   ```
   
   > **Note**: You can also 
   > [view the YAML in a new tab]({{site.baseurl}}/{{page.version}}/getting-started/kubernetes/installation/hosted/kubeadm/1.7/calico.yaml){:target="_blank"}.
   {: .alert .alert-info}

   You should see the following output.

   ```
   configmap "calico-config" created
   daemonset "calico-etcd" created
   service "calico-etcd" created
   daemonset "{{site.noderunning}}" created
   deployment "calico-kube-controllers" created
   clusterrolebinding "calico-cni-plugin" created
   clusterrole "calico-cni-plugin" created
   serviceaccount "calico-cni-plugin" created
   clusterrolebinding "calico-kube-controllers" created
   clusterrole "calico-kube-controllers" created
   serviceaccount "calico-kube-controllers" created
   ```

1. Remove the taints on the master so that you can schedule pods
   on it.

   ```
   kubectl taint nodes --all node-role.kubernetes.io/master-
   ```

   It should return the following.

   ```
   node "<your-hostname>" untainted
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

1. Use the following command on the master node to create a secret
   called `cnx-manager-tls` containing the self-signed Kubernetes API server
   certificate and key generated by kubeadm. We can use these to achieve
   TLS-encrypted communications with the {{site.prodname}} Manager.

   ```
   sudo -E kubectl create secret generic cnx-manager-tls --from-file=cert=/etc/kubernetes/pki/apiserver.crt \
   --from-file=key=/etc/kubernetes/pki/apiserver.key -n kube-system
   ```

   `kubectl` should return the following.

   ```
   secret "cnx-manager-tls" created
   ```
   
1. Install {{site.prodname}} and a single node etcd with the following command.

   ```
   kubectl apply -f \
   {{site.url}}/{{page.version}}/getting-started/kubernetes/installation/hosted/cnx/1.7/cnx-etcd.yaml
   ```
   
   > **Note**: You can also 
   > [view the YAML in a new tab]({{site.baseurl}}/{{page.version}}/getting-started/kubernetes/installation/hosted/cnx/1.7/cnx-etcd.yaml){:target="_blank"}.
   {: .alert .alert-info}

   You should see the following output.

   ```
   configmap "tigera-cnx-manager-config" created
   secret "cnx-apiserver-certs" created
   service "cnx-manager" created
   apiservice "v3.projectcalico.org" created
   clusterrolebinding "calico:system:auth-delegator" created
   rolebinding "calico-auth-reader" created
   serviceaccount "cnx-apiserver" created
   serviceaccount "cnx-manager" created
   service "api" created
   deployment "cnx-apiserver" created
   deployment "cnx-manager" created
   ```

1. Confirm that all of the pods are running with the following command.

   ```
   watch kubectl get pods --all-namespaces
   ```

   Wait until each pod has the `STATUS` of `Running`.

   ```
   NAMESPACE    NAME                                       READY  STATUS   RESTARTS  AGE
   kube-system  calico-etcd-x2482                          1/1    Running  0         12m
   kube-system  calico-kube-controllers-6ff88bf6d4-tgtzb   1/1    Running  0         12m
   kube-system  {{site.noderunning}}-24h85                             2/2    Running  0         12m
   kube-system  cnx-apiserver-m5bt9                        1/1    Running  0         4m
   kube-system  cnx-manager-68cf9fd767-szzl2               1/1    Running  0         2m
   kube-system  etcd-jbaker-virtualbox                     1/1    Running  0         16m
   kube-system  kube-apiserver-jbaker-virtualbox           1/1    Running  0         16m
   kube-system  kube-controller-manager-jbaker-virtualbox  1/1    Running  0         16m
   kube-system  kube-dns-545bc4bfd4-67qqp                  3/3    Running  0         15m
   kube-system  kube-proxy-8fzp2                           1/1    Running  0         15m
   kube-system  kube-scheduler-jbaker-virtualbox           1/1    Running  0         15m
   ```

1. Press CTRL+C to exit `watch`.

1. Optionally add network policy to ensure requests to CNX are permitted.  By default Kubernetes doesn't
   install any network policy, and therefore CNX Manager is accessible, but it is easy to
   unintentionally block it when editing policy so this is a recommended step.  Download the
   [cnx-poolicy.yaml]({{site.baseurl}}/{{page.version}}/getting-started/kubernetes/installation/hosted/cnx/1.7/cnx-policy.yaml)
   file and apply it.  This step is an example of applying CNX Tiered Security Policy using
   the `kubectl` command line tool.

   ```
   kubectl apply -f cnx-policy.yaml
   ```

1. Launch a browser and type `https://127.0.0.1:30003` in the address bar.

   > **Note**: If your browser is accessing a remote CNX installation via ssh tunnelling, make sure ssh tunnel has been setup correctly for port 30003.
   {: .alert .alert-info}

   > **Note**: If you didn't configure your browser, above, to trust the
   cluster CA certificate, the browser may warn you of an insecure
   connection.  Click past the warning.
   {: .alert .alert-info}

1. Type **jane** in the **Login** box and **welc0me** in the **Password** box.
   Then click **Sign In**.

Congratulations! You now have a single-host Kubernetes cluster
equipped with {{site.prodname}}.

### Next steps
**[Experiment with OIDC authentication strategy](/{{page.version}}/reference/cnx/authentication)**

**[Experiment with non-admin users and the {{site.prodname}} manager](/{{page.version}}/reference/cnx/rbac-tiered-policies)**

**[Secure a simple application using the Kubernetes `NetworkPolicy` API](tutorials/simple-policy)**

**[Control ingress and egress traffic using the Kubernetes `NetworkPolicy` API](tutorials/advanced-policy)**

**[Create a user interface that shows blocked and allowed connections in real time](tutorials/stars-policy/)**

**[Install and configure calicoctl](/{{page.version}}/usage/calicoctl/install)**
