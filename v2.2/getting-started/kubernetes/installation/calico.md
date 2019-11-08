---
title: Installing Calico Enterprise for policy and networking
canonical_url: https://docs.tigera.io/v2.3/getting-started/kubernetes/installation/calico
---

## Before you begin

- Ensure that you have a Kubernetes cluster that meets the {{site.prodname}}
  [system requirements](../requirements). If you don't, follow the steps in
  [Using kubeadm to create a cluster](http://kubernetes.io/docs/getting-started-guides/kubeadm/).

- Ensure that you have the [credentials for the Tigera private registry](../../../getting-started/#obtain-the-private-registry-credentials)
  and a [license key](../../../getting-started/#obtain-a-license-key).

{% include {{page.version}}/load-docker.md yaml="calico" orchestrator="kubernetes" %}

## <a name="install-cnx"></a>Installing {{site.prodname}} for policy and networking

### Selecting your cluster configuration

The procedure differs according to whether or not you want to [federate clusters](../../../usage/federation/index)
and your datastore type. Refer to the section that matches your configuration.

- **Without federation**:
   - [etcd datastore](#installing-without-federation-using-etcd)
   - [Kubernetes API datastore, 50 nodes or less](#installing-without-federation-using-kubernetes-api-datastore-50-nodes-or-less)
   - [Kubernetes API datastore, more than 50 nodes](#installing-without-federation-using-kubernetes-api-datastore-50-nodes-or-less)

- **With federation**:
   - [etcd datastore](#installing-with-federation-using-etcd)
   - [Kubernetes API datastore](#installing-with-federation-using-kubernetes-api-datastore)

> **Note**: {{site.prodname}} networking with the Kubernetes API datastore
> is beta because it does not yet support {{site.prodname}} IPAM. It uses
> `host-local` IPAM with Kubernetes pod CIDR assignments instead.
{: .alert .alert-info}

### Installing without federation, using etcd

1. If your cluster has RBAC enabled, issue the following command to
   configure the roles and bindings that {{site.prodname}} requires.

   ```
   kubectl apply -f \
   {{site.url}}/{{page.version}}/getting-started/kubernetes/installation/rbac.yaml
   ```
   > **Note**: You can also [view the manifest in your browser](rbac.yaml){:target="_blank"}.
   {: .alert .alert-info}

1. Download the {{site.prodname}} networking manifest for etcd.

   ```bash
   curl \
   {{site.url}}/{{page.version}}/getting-started/kubernetes/installation/hosted/calico.yaml \
   -O
   ```

1. In the `ConfigMap` named `calico-config`, set the value of
   `etcd_endpoints` to the IP address and port of your etcd server.

   > **Tip**: You can specify more than one using commas as delimiters.
   {: .alert .alert-success}

{% include {{page.version}}/cnx-cred-sed.md yaml="calico" %}

1. Apply the manifest using the following command.

   ```bash
   kubectl apply -f calico.yaml
   ```

1. Continue to [Applying your license key](#applying-your-license-key).

### Installing without federation, using Kubernetes API datastore, 50 nodes or less

1. Ensure that the Kubernetes controller manager has the following flags
   set: <br>
   `--cluster-cidr=192.168.0.0/16` and `--allocate-node-cidrs=true`.

   > **Tip**: On kubeadm, you can pass `--pod-network-cidr=192.168.0.0/16`
   > to kubeadm to set both Kubernetes controller flags.
   {: .alert .alert-success}

1. If your cluster has RBAC enabled, issue the following command to
   configure the roles and bindings that {{site.prodname}} requires.

   ```
   kubectl apply -f \
   {{site.url}}/{{page.version}}/getting-started/kubernetes/installation/hosted/rbac-kdd.yaml
   ```
   > **Note**: You can also
   > [view the manifest in your browser](hosted/rbac-kdd.yaml){:target="_blank"}.
   {: .alert .alert-info}

1. Download the {{site.prodname}} networking manifest for the Kubernetes API datastore.

   ```bash
   curl \
   {{site.url}}/{{page.version}}/getting-started/kubernetes/installation/hosted/kubernetes-datastore/calico-networking/1.7/calico.yaml \
   -O
   ```

{% include {{page.version}}/cnx-cred-sed.md yaml="calico" %}

1. Apply the manifest.

   ```bash
   kubectl apply -f calico.yaml
   ```

1. Continue to [Applying your license key](#applying-your-license-key).

### Installing without federation, using Kubernetes API datastore, more than 50 nodes

1. Ensure that the Kubernetes controller manager has the following flags set:<br>
   `--cluster-cidr=192.168.0.0/16` and `--allocate-node-cidrs=true`.

   > **Tip**: On kubeadm, you can pass `--pod-network-cidr=192.168.0.0/16`
   > to kubeadm to set both Kubernetes controller flags.
   {: .alert .alert-success}

1. If your cluster has RBAC enabled, issue the following command to
   configure the roles and bindings that {{site.prodname}} requires.

   ```
   kubectl apply -f \
   {{site.url}}/{{page.version}}/getting-started/kubernetes/installation/hosted/rbac-kdd.yaml
   ```
   > **Note**: You can also
   > [view the manifest in your browser](hosted/rbac-kdd.yaml){:target="_blank"}.
   {: .alert .alert-info}

1. Download the {{site.prodname}} networking manifest for the Kubernetes API datastore.

   ```bash
   curl \
   {{site.url}}/{{page.version}}/getting-started/kubernetes/installation/hosted/kubernetes-datastore/calico-networking/typha/calico.yaml \
   -O
   ```

1. Modify the replica count in the`Deployment` named `calico-typha`
   to the desired number of replicas.

   ```
   apiVersion: apps/v1beta1
   kind: Deployment
   metadata:
     name: calico-typha
     ...
   spec:
     ...
     replicas: <number of replicas>
   ```

   We recommend at least one replica for every 200 nodes and no more than
   20 replicas. In production, we recommend a minimum of three replicas to reduce
   the impact of rolling upgrades and failures.

   > **Warning**: If you do not increase the replica
   > count from its default of `0` Felix will try to connect to Typha, find no
   > Typha instances to connect to, and fail to start.
   {: .alert .alert-danger}

{% include {{page.version}}/cnx-cred-sed.md yaml="calico" %}

1. Apply the manifest.

   ```bash
   kubectl apply -f calico.yaml
   ```

1. Continue to [Applying your license key](#applying-your-license-key).


### Installing with federation, using etcd

The following procedure describes how to install {{site.prodname}} on a single cluster that uses an
etcd datastore (the [local cluster](../../../usage/federation/index#terminology)).

**Prerequisite**: Complete the steps in [Creating kubeconfig files](../../../usage/federation/kubeconfig)
for each [remote cluster](../../../usage/federation/index#terminology). Ensure that the
[local cluster](../../../usage/federation/index#terminology) can access all of the necessary `kubeconfig` files.

1. Access the local cluster using a `kubeconfig` with administrative privileges.

1. Create a secret containing the `kubeconfig` files for all of the remote clusters that
   the local cluster should federate with. A command to achieve this follows. Adjust the `--from-file`
   flags to include all of the kubeconfig files you created in [Creating kubeconfig files](../../../usage/federation/kubeconfig).

   > **Tip**: We recommend naming this secret `tigera-federation-remotecluster` as shown below
   > to make the rest of the procedure easier to follow.
   {: .alert .alert-success}

   ```bash
   kubectl create secret generic tigera-federation-remotecluster \
   --from-file=kubeconfig-rem-cluster-1 --from-file=kubeconfig-rem-cluster-2 \
   --namespace=kube-system
   ```

1. If the local cluster has RBAC enabled, issue the following command to
   configure the roles and bindings that {{site.prodname}} requires on the local cluster.

   ```
   kubectl apply -f \
   {{site.url}}/{{page.version}}/getting-started/kubernetes/installation/rbac-etcd-typha.yaml
   ```

   > **Note**: You can also
   > [view the manifest in your browser](rbac-etcd-typha.yaml){:target="_blank"}.
   {: .alert .alert-info}

1. Download the {{site.prodname}} networking manifest for etcd.

   ```bash
   curl \
   {{site.url}}/{{page.version}}/getting-started/kubernetes/installation/hosted/federation/calico.yaml \
   -O
   ```

1. In the `ConfigMap` named `calico-config`, set the value of
      `etcd_endpoints` to the IP address and port of your etcd server.

   > **Tip**: You can specify more than one using commas as delimiters.
   {: .alert .alert-success}

1. Modify the replica count in the `Deployment` named `calico-typha`
   to the desired number of replicas.

   ```
   apiVersion: apps/v1beta1
   kind: Deployment
   metadata:
     name: calico-typha
     ...
   spec:
     ...
     replicas: <number of replicas>
   ```

   We recommend at least one replica for every 200 nodes and no more than
   20 replicas. In production, we recommend a minimum of three replicas to reduce
   the impact of rolling upgrades and failures.

   > **Warning**: If you do not increase the replica
   > count from its default of `0` Felix will try to connect to Typha, find no
   > Typha instances to connect to, and fail to start.
   {: .alert .alert-danger}

{% include {{page.version}}/cnx-pod-cidr-sed.md yaml="calico" %}

{% include {{page.version}}/cnx-cred-sed.md yaml="calico" %}

1. Apply the manifest.

   ```bash
   kubectl apply -f calico.yaml
   ```

1. Continue to [Applying your license key](#applying-your-license-key).


### Installing with federation, using Kubernetes API datastore

The following procedure describes how to install {{site.prodname}} on a single cluster that uses the
Kubernetes API datastore (the [local cluster](../../../usage/federation/index#terminology)).

**Prerequisite**: Complete the steps in [Creating kubeconfig files](../../../usage/federation/kubeconfig)
for each [remote cluster](../../../usage/federation/index#terminology). Ensure that the
[local cluster](../../../usage/federation/index#terminology) can access all of the necessary `kubeconfig` files.

1. Access the local cluster using a `kubeconfig` with administrative privileges.

1. Create a secret containing the `kubeconfig` files for all of the remote clusters that
   the local cluster should federate with. A command to achieve this follows. Adjust the `--from-file`
   flags to include all of the kubeconfig files you created in [Creating kubeconfig files](../../../usage/federation/kubeconfig).

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

1. If the local cluster has RBAC enabled, issue the following command to
   configure the roles and bindings that {{site.prodname}} requires.

   ```
   kubectl apply -f \
   {{site.url}}/{{page.version}}/getting-started/kubernetes/installation/hosted/rbac-kdd.yaml
   ```

   > **Note**: You can also
   > [view the manifest in your browser](hosted/rbac-kdd.yaml){:target="_blank"}.
   {: .alert .alert-info}

1. Download the {{site.prodname}} networking manifest for the Kubernetes API datastore.

   ```bash
   curl \
   {{site.url}}/{{page.version}}/getting-started/kubernetes/installation/hosted/kubernetes-datastore/calico-networking/federation/calico.yaml \
   -O
   ```

1. Open the manifest in your favorite editor and modify the replica count in the
   `Deployment` named `calico-typha` to the desired number of replicas.

   ```
   apiVersion: apps/v1beta1
   kind: Deployment
   metadata:
     name: calico-typha
     ...
   spec:
     ...
     replicas: <number of replicas>
   ```

   We recommend at least one replica for every 200 nodes and no more than
   20 replicas. In production, we recommend a minimum of three replicas to reduce
   the impact of rolling upgrades and failures.

   > **Warning**: If you do not increase the replica
   > count from its default of `0` Felix will try to connect to Typha, find no
   > Typha instances to connect to, and fail to start.
   {: .alert .alert-danger}

{% include {{page.version}}/cnx-pod-cidr-sed.md yaml="calico" %}

{% include {{page.version}}/cnx-cred-sed.md yaml="calico" %}

1. Apply the manifest.

   ```bash
   kubectl apply -f calico.yaml
   ```

1. Continue to [Applying your license key](#applying-your-license-key).


{% include {{page.version}}/apply-license.md %}

{% include {{page.version}}/cnx-mgr-install.md init="kubernetes" net="calico" %}

1. For production installs, we recommend using your own Elasticsearch cluster. If you are performing a
   production install, do not complete any more steps on this page. Instead, refer to
   [Using your own Elasticsearch for logs](byo-elasticsearch) for the final steps.

   For demonstration or proof of concept installs, you can use the bundled
   [Elasticsearch operator](https://github.com/upmc-enterprises/elasticsearch-operator). Continue to the
   next step to complete a demonstration or proof of concept install.

   > **Important**: The bundled Elasticsearch operator does not provide reliable persistent storage
   of logs or authenticate access to Kibana.
   {: .alert .alert-danger}

{% include {{page.version}}/cnx-monitor-install.md elasticsearch="operator"%}

{% include {{page.version}}/gs-next-steps.md %}
