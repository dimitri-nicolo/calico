---
title: Quickstart for Tigera CNX on Kubernetes
---


### Overview

This quickstart gets you a single-host Kubernetes cluster with {{site.prodname}}
in approximately 10 minutes. You can use this cluster for testing and development.

To deploy a cluster suitable for production, refer to [Installation](/{{page.version}}/getting-started/kubernetes/installation/).


### Host requirements

- AMD64 processor
- 2CPU
- 4GB RAM
- 10GB free disk space
- Red Hat Enterprise Linux 7, CentOS 7, Ubuntu 16.04, or Debian 9
- [jq](https://stedolan.github.io/jq/download/)
- Internet access


### Before you begin

[Follow the Kubernetes instructions to install kubeadm](https://kubernetes.io/docs/setup/independent/install-kubeadm/){:target="_blank"}.

> **Note**: After installing kubeadm, do not power down or restart
the host. Instead, continue directly to the
[next section to create your cluster](#create-a-single-host-kubernetes-cluster).
{: .alert .alert-info}


### Create a single-host Kubernetes cluster
   
1. As a regular user with sudo privileges, open a terminal on the host that
   you installed kubeadm on.
   
1. Initialize the master using the following command.

   ```bash
   sudo kubeadm init --pod-network-cidr=192.168.0.0/16 --apiserver-cert-extra-sans=127.0.0.1
   ```

1. Execute the commands to configure kubectl as returned by
   `kubeadm init`. Most likely they will be as follows:

   ```bash
   mkdir -p $HOME/.kube
   sudo cp -i /etc/kubernetes/admin.conf $HOME/.kube/config
   sudo chown $(id -u):$(id -g) $HOME/.kube/config
   ```
   
1. Ensure that you have a local copy of the [`config.json` file with the private Tigera registry credentials](/{{page.version}}/getting-started/#obtain-the-private-registry-credentials),
   ideally in your current directory.
   
1. Download the installation script.
   
   ```bash
   curl --compressed {{site.url}}/{{page.version}}/getting-started/kubernetes/install-cnx.sh -O
   ```

1. Set the `install-cnx.sh` file to be executable.
   
   ```bash
   chmod +x install-cnx.sh
   ```

1. Use the following command to execute the script.

   ```
   ./install-cnx.sh -v {{page.version}}
   ```
 
1. Launch a browser and type `https://127.0.0.1:30003` in the address bar.

   > **Note**: Your browser may warn you of an insecure connection due to 
   > the self-signed certificate. Click past this warning to access the 
   > {{site.prodname}} web interface.
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
