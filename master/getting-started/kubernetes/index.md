---
title: Quickstart for Tigera Secure EE on Kubernetes
canonical_url: https://docs.tigera.io/v2.3/getting-started/kubernetes/
---

### Overview

This quickstart gets you a single-host Kubernetes cluster with {{site.prodname}}
in approximately 15 minutes. You can use this cluster for testing and
development.

To deploy a cluster suitable for production, refer to [Installation](installation).

### Host requirements

- AMD64 processor
- 2CPU
- 12GB RAM
- 50GB free disk space
- Ubuntu Server 16.04
- Internet access
- [Sufficient virtual memory](https://www.elastic.co/guide/en/elasticsearch/reference/current/vm-max-map-count.html){:target="_blank"}

### Before you begin

- Ensure that you have the following files in your current working directory:
  - [`config.json` containing the Tigera private registry credentials](/{{page.version}}/getting-started/#obtain-the-private-registry-credentials)
  - [`<customer-name>-license.yaml` containing your license key](/{{page.version}}/getting-started/#obtain-a-license-key)
<br><br>
- Ensure that {{site.prodname}} can manage `cali` and `tunl` interfaces on the host.
  If NetworkManager is present on the host, refer to
  [Configure NetworkManager](../../maintenance/troubleshooting#configure-networkmanager)
.

- [Follow the Kubernetes instructions to install kubeadm](https://kubernetes.io/docs/setup/independent/install-kubeadm/){:target="_blank"}.

   > **Note**: After installing kubeadm, do not power down or restart
   the host. Instead, continue directly to the
   [next section to create your cluster](#create-a-single-host-kubernetes-cluster).
   {: .alert .alert-info}

### Create a single-host Kubernetes cluster

1. As a regular user with sudo privileges, open a terminal on the host that
   you installed kubeadm on.

1. Initialize the master using the following command.

   ```bash
   sudo kubeadm init --pod-network-cidr=192.168.0.0/16 \
   --apiserver-cert-extra-sans=127.0.0.1
   ```

   > **Note**: If 192.168.0.0/16 is already in use within your network you must select a different pod network
   > CIDR, replacing 192.168.0.0/16 in the above command as well as in any manifests applied below.
   {: .alert .alert-info}

1. Execute the following commands to configure kubectl (also returned by
   `kubeadm init`).

   ```bash
   mkdir -p $HOME/.kube
   sudo cp -i /etc/kubernetes/admin.conf $HOME/.kube/config
   sudo chown $(id -u):$(id -g) $HOME/.kube/config
   ```
1. Download the installation script.

   ```bash
   curl --compressed \
   {{site.url}}/{{page.version}}/getting-started/kubernetes/install-cnx.sh -O
   ```

1. Set the `install-cnx.sh` file to be executable.

   ```bash
   chmod +x install-cnx.sh
   ```
1. Use the following command to execute the script, replacing `<customer-name>`
   with your customer name.

   **Command**
   ```bash
   ./install-cnx.sh -l <customer-name>-license.yaml
   ```

   **Example**
   ```bash
   ./install-cnx.sh -l awesome-corp-license.yaml
   ```
   {: .no-select-button}

1. Save the username and password printed out at the end of a successful install.

1. Launch a browser and type `https://127.0.0.1:30003` in the address bar.

   > **Note**: Your browser will warn you of an insecure connection due to
   > the self-signed certificate. Click past this warning to access the
   > {{site.prodname}} Manager.
   {: .alert .alert-info}

1. In the **Login** and **Password** box enter the username and password
   printed at the end of the install.
   Then click **Sign In**.

Congratulations! You now have a single-host Kubernetes cluster
equipped with {{site.prodname}}.

### Next steps

**[Experiment with OIDC authentication strategy](/{{page.version}}/reference/cnx/authentication)**

**[Experiment with non-admin users and the {{site.prodname}} manager](/{{page.version}}/reference/cnx/rbac-tiered-policies)**

**[Enable audit logs for Kubernetes `NetworkPolicy` and send them to Elasticsearch](/{{page.version}}/security/logs/elastic/ee-audit#enabling-auditing-for-other-resources)**

**[Secure a simple application using the Kubernetes `NetworkPolicy` API]({{site.url}}/{{page.version}}/security/simple-policy)**

**[Control ingress and egress traffic using the Kubernetes `NetworkPolicy` API]({{site.url}}/{{page.version}}/security/advanced-policy)**

**[Create a user interface that shows blocked and allowed connections in real time]({{site.url}}/{{page.version}}/security/stars-policy/)**

**[Install and configure calicoctl](/{{page.version}}/getting-started/calicoctl/install)**
