{% unless include.upgrade %}
## Before you begin

- Ensure that you have a Kubernetes cluster that meets the {{site.prodname}}
  [system requirements](/{{page.version}}/getting-started/kubernetes/requirements). If you don't, follow the steps in
  [Using kubeadm to create a cluster](http://kubernetes.io/docs/getting-started-guides/kubeadm/).

- Ensure that you have the [credentials for the Tigera private registry](/{{page.version}}/getting-started/#obtain-the-private-registry-credentials)
  and a [license key](/{{page.version}}/getting-started/#obtain-a-license-key).
{% endunless %}

{% include content/load-docker.md yaml="calico" orchestrator="kubernetes" upgrade=include.upgrade %}

{% unless include.upgrade %}
{% include content/pull-secret.md %}
{% endunless %}

## <a name="install-cnx"></a>Installing {{site.prodname}} for policy and networking

### Selecting your cluster configuration

The procedure differs according to whether or not you want to [federate clusters](/{{page.version}}/networking/federation/index)
and your datastore type. Refer to the section that matches your configuration.

- **Without federation**:
   - [etcd datastore](#installing-without-federation-using-etcd)
   - [Kubernetes API datastore](#installing-without-federation-using-kubernetes-api-datastore)

- **With federation**:
   - [etcd datastore](#installing-with-federation-using-etcd)
   - [Kubernetes API datastore](#installing-with-federation-using-kubernetes-api-datastore)

### Installing without federation, using etcd

1. Download the {{site.prodname}} networking manifest for etcd.

   ```bash
   curl \
   {{ "/getting-started/kubernetes/installation/hosted/calico.yaml" | absolute_url }} \
   -O
   ```

1. In the `ConfigMap` named `calico-config`, set the value of
   `etcd_endpoints` to the IP address and port of your etcd server.

   > **Tip**: You can specify more than one using commas as delimiters.
   {: .alert .alert-success}

{% include content/cnx-pod-cidr-sed.md yaml="calico" %}

{% include content/cnx-cred-sed.md yaml="calico" %}

1. Apply the manifest using the following command.

   ```bash
   kubectl apply -f calico.yaml
   ```

1. Continue to [Installing the {{site.prodname}} API Server](#installing-the-{{site.prodnamedash}}-api-server)


### Installing without federation, using Kubernetes API datastore

1. Ensure that the Kubernetes controller manager has the following flags
   set: <br>
   `--cluster-cidr=<your-pod-cidr>` and `--allocate-node-cidrs=true`.

   > **Tip**: On kubeadm, you can pass `--pod-network-cidr=<your-pod-cidr>`
   > to kubeadm to set both Kubernetes controller flags.
   {: .alert .alert-success}

1. Download the {{site.prodname}} networking manifest for the Kubernetes API datastore.

   ```bash
   curl \
   {{ "/getting-started/kubernetes/installation/hosted/kubernetes-datastore/calico-networking/typha/calico.yaml" | absolute_url }} \
   -O
   ```

{% include content/cnx-pod-cidr-sed.md yaml="calico" %}

{% include content/config-typha.md %}

1. Apply the manifest.

   ```bash
   kubectl apply -f calico.yaml
   ```

1. Continue to [Installing the {{site.prodname}} API Server](#installing-the-{{site.prodnamedash}}-api-server)


### Installing with federation, using etcd

The following procedure describes how to install {{site.prodname}} on a single cluster that uses an
etcd datastore (the [local cluster](/{{page.version}}/networking/federation/index#terminology)).

**Prerequisite**: Complete the steps in [Creating kubeconfig files](/{{page.version}}/networking/federation/kubeconfig)
for each [remote cluster](/{{page.version}}/networking/federation/index#terminology). Ensure that the
[local cluster](/{{page.version}}/networking/federation/index#terminology) can access all of the necessary `kubeconfig` files.

1. Access the local cluster using a `kubeconfig` with administrative privileges.

1. Create a secret containing the `kubeconfig` files for all of the remote clusters that
   the local cluster should federate with. A command to achieve this follows. Adjust the `--from-file`
   flags to include all of the kubeconfig files you created in [Creating kubeconfig files](/{{page.version}}/networking/federation/kubeconfig).

   > **Tip**: We recommend naming this secret `tigera-federation-remotecluster` as shown below
   > to make the rest of the procedure easier to follow.
   {: .alert .alert-success}

   ```bash
   kubectl create secret generic tigera-federation-remotecluster \
   --from-file=kubeconfig-rem-cluster-1 --from-file=kubeconfig-rem-cluster-2 \
   --namespace=kube-system
   ```

1. Download the {{site.prodname}} networking manifest for etcd.

   ```bash
   curl \
   {{ "/getting-started/kubernetes/installation/hosted/federation/calico.yaml" | absolute_url }} \
   -O
   ```

{% include content/cnx-pod-cidr-sed.md yaml="calico" %}

1. In the `ConfigMap` named `calico-config`, set the value of
   `etcd_endpoints` to the IP address and port of your etcd server.

   > **Tip**: You can specify more than one using commas as delimiters.
   {: .alert .alert-success}

{% include content/config-typha.md %}

{% include content/cnx-pod-cidr-sed.md yaml="calico" %}

{% include content/cnx-cred-sed.md yaml="calico" %}

1. Apply the manifest using the following command.

   ```bash
   kubectl apply -f calico.yaml
   ```

1. Continue to [Installing the {{site.prodname}} API Server](#installing-the-{{site.prodnamedash}}-api-server)


### Installing with federation, using Kubernetes API datastore

The following procedure describes how to install {{site.prodname}} on a single cluster that uses the
Kubernetes API datastore (the [local cluster](/{{page.version}}/networking/federation/index#terminology)).

**Prerequisite**: Complete the steps in [Creating kubeconfig files](/{{page.version}}/networking/federation/kubeconfig)
for each [remote cluster](/{{page.version}}/networking/federation/index#terminology). Ensure that the
[local cluster](/{{page.version}}/networking/federation/index#terminology) can access all of the necessary `kubeconfig` files.

1. Access the local cluster using a `kubeconfig` with administrative privileges.

1. Create a secret containing the `kubeconfig` files for all of the remote clusters that
   the local cluster should federate with. A command to achieve this follows. Adjust the `--from-file`
   flags to include all of the kubeconfig files you created in [Creating kubeconfig files](/{{page.version}}/networking/federation/kubeconfig).

   > **Tip**: We recommend naming this secret `tigera-federation-remotecluster` as shown below to
   > make the rest of the procedure easier to follow.
   {: .alert .alert-success}

   ```bash
   kubectl create secret generic tigera-federation-remotecluster \
   --from-file=kubeconfig-rem-cluster-1 --from-file=kubeconfig-rem-cluster-2 \
   --namespace=kube-system
   ```

1. Ensure that the Kubernetes controller manager has the following flags set:<br>
   - `--cluster-cidr=<cidr>`: Ensure that the `<cidr>` value matches or includes the IPV4 pool
     (`CALICO_IPV4POOL_CIDR`) in the manifest and does not overlap with the IPV4 pools of any other
     federated clusters. Example: `--cluster-cidr=192.168.0.0/16`
   - `--allocate-node-cidrs=true`

   > **Tip**: On kubeadm, you can pass `--pod-network-cidr=<cidr>`
   > to kubeadm to set both Kubernetes controller flags.
   {: .alert .alert-success}

1. Download the {{site.prodname}} networking manifest for the Kubernetes API datastore.

   ```bash
   curl \
   {{ "/getting-started/kubernetes/installation/hosted/kubernetes-datastore/calico-networking/federation/calico.yaml" | absolute_url }} \
   -O
   ```

{% include content/config-typha.md %}

{% include content/cnx-pod-cidr-sed.md yaml="calico" %}

{% include content/cnx-cred-sed.md yaml="calico" %}

1. Apply the manifest.

   ```bash
   kubectl apply -f calico.yaml
   ```

1. Continue to [Installing the {{site.prodname}} API Server](#installing-the-{{site.prodnamedash}}-api-server)

{% include content/cnx-api-install.md init="kubernetes" net="calico" upgrade=include.upgrade %}

1. Continue to [Applying your license key](#applying-your-license-key).

{% include content/apply-license.md cli="kubectl" %}

{% unless include.upgrade %}
1. Continue to [Installing the {{site.prodname}} Manager](#installing-the-{{site.prodnamedash}}-manager)

{% include content/cnx-mgr-install.md init="kubernetes" %}

{% include content/gs-next-steps.md %}
{% endunless %}
