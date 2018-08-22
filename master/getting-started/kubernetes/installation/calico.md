---
title: Installing CNX for policy and networking (recommended)
---

## Before you begin

- Ensure that you have a Kubernetes cluster that meets the {{site.prodname}}
  [system requirements](../requirements). If you don't, follow the steps in
  [Using kubeadm to create a cluster](http://kubernetes.io/docs/getting-started-guides/kubeadm/).

- If you are planning to federate multiple clusters using the Federated Endpoint Identity feature, please read the
  [Federated Identity Overview](../../../usage/federation/index) to understand additional installation requirements
  for each of your clusters.

- Ensure that you have the [private registry credentials](../../../getting-started/#obtain-the-private-registry-credentials)
  and a [license key](../../../getting-started/#obtain-a-license-key).

{% include {{page.version}}/load-docker-intro.md %}

{% include {{page.version}}/load-docker-our-reg.md yaml="calico" %}

{% include {{page.version}}/load-docker.md yaml="calico" orchestrator="kubernetes" %}

## <a name="install-cnx"></a>Installing {{site.prodname}} for policy and networking

### Selecting your datastore type and number of nodes

The procedure differs according to the type of datastore you want {{site.prodname}}
to use, the number of nodes and whether you will be using [federated endpoint identity](../../../usage/federation/index). Refer to the
section that matches your use case.

- [etcd datastore without federation](#installing-for-etcd-datastore-without-federation)

- [etcd datastore with federation](#installing-for-etcd-datastore-with-federation)

- [Kubernetes API datastore, 50 nodes or less and without federation](#50-nodes-or-less)

- [Kubernetes API datastore, more than 50 nodes and/or with federation](#installing-for-kubernetes-api-datastore-more-than-50-nodes-or-with-federation)

> **Note**: {{site.prodname}} networking with the Kubernetes API datastore
> is beta because it does not yet support {{site.prodname}} IPAM. It uses
> `host-local` IPAM with Kubernetes pod CIDR assignments instead.
{: .alert .alert-info}

### Installing for etcd datastore: without federation

1. If your cluster has RBAC enabled, issue the following command to
   configure the roles and bindings that {{site.prodname}} requires.

   ```
   kubectl apply -f \
   {{site.url}}/{{page.version}}/getting-started/kubernetes/installation/rbac.yaml
   ```
   > **Note**: You can also
   > [view the manifest in your browser](rbac.yaml){:target="_blank"}.
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

{% include {{page.version}}/cnx-pod-cidr-sed.md yaml="calico" %}

{% include {{page.version}}/cnx-cred-sed.md yaml="calico" %}

1. Apply the manifest using the following command.

   ```bash
   kubectl apply -f calico.yaml
   ```

1. Continue to [Applying your license key](#applying-your-license-key).

### Installing for etcd datastore: with federation

1. If your cluster has RBAC enabled, issue the following command to
   configure the roles and bindings that {{site.prodname}} requires.

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
   {{site.url}}/{{page.version}}/getting-started/kubernetes/installation/hosted/etcd-typha/calico.yaml \
   -O
   ```

1. In the `ConfigMap` named `calico-config`, set the value of
   `etcd_endpoints` to the IP address and port of your etcd server.

   > **Tip**: You can specify more than one using commas as delimiters.
   {: .alert .alert-success}

1. In the `ConfigMap` named `calico-config`, locate the `typha_service_name`,
   delete the `none` value, and replace it with `calico-typha`.

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

   > **Warning**: If you set `typha_service_name` without increasing the replica
   > count from its default of `0` Felix will try to connect to Typha, find no
   > Typha instances to connect to, and fail to start.
   {: .alert .alert-danger}

{% include {{page.version}}/cnx-pod-cidr-sed.md yaml="calico" %}

{% include {{page.version}}/cnx-cred-sed.md yaml="calico" %}

1. If you have created some secrets for Federation, then modify the manifest to mount the secrets into the
   container.  For details see [Configuring a Remote Cluster for Federation](/{{page.version}}/usage/federation/configure-rcc).

   > **Warning**: If you are upgrading from a previous release and previously had secrets mounted in for
   > federation, then failure to include these secrets in this manifest will result in loss of federation
   > functionality, which may include loss of service between clusters.
   {: .alert .alert-danger}

1. Apply the manifest using the following command.

   ```bash
   kubectl apply -f calico.yaml
   ```

1. Continue to [Applying your license key](#applying-your-license-key).

1. If you wish to enforce application layer policies and secure workload-to-workload
   communications with mutual TLS authentication, continue to [Enabling application layer policy](app-layer-policy) (optional).

### <a name="50-nodes-or-less"></a> Installing with the Kubernetes API datastore—50 nodes or less

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

{% include {{page.version}}/cnx-pod-cidr-sed.md yaml="calico" %}

{% include {{page.version}}/cnx-cred-sed.md yaml="calico" %}

1. Apply the manifest

   ```bash
   kubectl apply -f calico.yaml
   ```

   > **Note**: You can also [view the manifest in your browser](hosted/kubernetes-datastore/calico-networking/1.7/calico.yaml){:target="_blank"}.
   {: .alert .alert-info}


1. Continue to [Applying your license key](#applying-your-license-key).

1. If you wish to enforce application layer policies and secure workload-to-workload
   communications with mutual TLS authentication, continue to [Enabling application layer policy](app-layer-policy) (optional).

### Installing for Kubernetes API datastore: more than 50 nodes or with federation

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
   {{site.url}}/{{page.version}}/getting-started/kubernetes/installation/hosted/kubernetes-datastore/calico-networking/1.7/calico.yaml \
   -O
   ```

1. In the `ConfigMap` named `calico-config`, locate the `typha_service_name`,
   delete the `none` value, and replace it with `calico-typha`.

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

   > **Warning**: If you set `typha_service_name` without increasing the replica
   > count from its default of `0` Felix will try to connect to Typha, find no
   > Typha instances to connect to, and fail to start.
   {: .alert .alert-danger}

{% include {{page.version}}/cnx-pod-cidr-sed.md yaml="calico" %}

{% include {{page.version}}/cnx-cred-sed.md yaml="calico" %}

1. If you have created some secrets for Federation, then modify the manifest to mount the secrets into the
   container.  For details see [Configuring a Remote Cluster for Federation](/{{page.version}}/usage/federation/configure-rcc).

   > **Warning**: If you are upgrading from a previous release and previously had secrets mounted in for
   > federation, then failure to include these secrets in this manifest will result in loss of federation
   > functionality, which may include loss of service between clusters.
   {: .alert .alert-danger}

1. Apply the manifest.

   ```bash
   kubectl apply -f calico.yaml
   ```

1. Continue to [Applying your license key](#applying-your-license-key).

{% include {{page.version}}/apply-license.md %}

{% include {{page.version}}/cnx-mgr-install.md init="kubernetes" %}

{% include {{page.version}}/cnx-monitor-install.md %}

{% include {{page.version}}/gs-next-steps.md %}

1. If you wish to enforce application layer policies and secure workload-to-workload
   communications with mutual TLS authentication, continue to [Enabling application layer policy](app-layer-policy) (optional).
