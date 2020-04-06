---
title: Install Calico Enterprise on a Rancher Kubernetes Engine cluster
description: Install Calico Enterprise on a Rancher Kubernetes Engine cluster.
canonical_url: '/getting-started/kubernetes/index'
---

### Big picture

Install {{side.prodname}} as the required CNI for networking and/or network policy on Rancher-deployed clusters.

### Before you begin

- [Create a compatible Rancher Kubernetes Engine (RKE) cluster](#create-a-compatible-rancher-cluster)
- [Gather the necessary resources](#gather-required-resources)
- If using a private registry, familiarize yourself with this guide on [using a private registry]({{site.baseurl}}/getting-started/private-registry).

#### Create a compatible Rancher cluster

- Ensure that your [Rancher Kubernetes Engine cluster](https://rancher.com/docs/rke/latest/en/)
  meets the [requirements](../requirements) for {{site.prodname}}.
  - Configure your cluster with a [Cluster Config File](https://rancher.com/docs/rancher/v2.x/en/cluster-provisioning/rke-clusters/options/#cluster-config-file)
    and specify [no network plugin](https://rancher.com/docs/rke/latest/en/config-options/add-ons/network-plugins/) by setting
    `plugin: none` under `network` in your configuration file.

#### Gather required resources

- Ensure that you have the [credentials for the Tigera private registry and a license key](../../../getting-started/calico-enterprise).

- Ensure you have a kubectl environment with access to your cluster
  - Use [Rancher kubectl Shell](https://rancher.com/docs/rancher/v2.x/en/cluster-admin/cluster-access/kubectl/) for access
  - Ensure you have the [Kubeconfig file that was generated when you created the cluster](https://rancher.com/docs/rke/latest/en/installation/#save-your-files).

- If using a Kubeconfig file locally, [install and set up the Kubectl CLI tool](https://kubernetes.io/docs/tasks/tools/install-kubectl/).

### How to

- [Install {{site.prodname}}](#install-calico-enterprise)
- [Install the {{site.prodname}} license](#install-the-calico-enterprise-license)
- [Secure {{site.prodname}} components with network policy](#secure-calico-enterprise-components-with-network-policy)


#### Install {{site.prodname}}

1. [Configure a storage class for {{site.prodname}}.]({{site.baseurl}}/getting-started/create-storage)

1. Install the Tigera operators and custom resource definitions.

   ```
   kubectl create -f {{ "/manifests/tigera-operator.yaml" | absolute_url }}
   ```

1. Install your pull secret.

   ```
   kubectl create secret generic tigera-pull-secret \
       --from-file=.dockerconfigjson=<path/to/pull/secret> \
       --type=kubernetes.io/dockerconfigjson -n tigera-operator
   ```

1. Install any extra [Calico resources]({{site.baseurl}}/reference/resources) needed at cluster start using [calicoctl]({{site.baseurl}}/reference/calicoctl/overview).

1. Install the Tigera custom resources. For more information on configuration options available in this manifest, see [the installation reference]({{site.baseurl}}/reference/installation/api).

   ```
   kubectl create -f {{ "/manifests/custom-resources.yaml" | absolute_url }}
   ```

   You can now monitor progress with the following command:

   ```
   watch kubectl get tigerastatus
   ```

   Wait until the `apiserver` shows a status of `Available`, then proceed to the next section.

#### Install the {{site.prodname}} license

In order to use {{site.prodname}}, you must install the license provided to you by Tigera.

```
kubectl create -f </path/to/license.yaml>
```

You can now monitor progress with the following command:

```
watch kubectl get tigerastatus
```

When all components show a status of `Available`, proceed to the next section.

#### Secure {{site.prodname}} components with network policy

To secure {{site.prodname}} component communications, install the following set of network policies.

```
kubectl create -f {{ "/manifests/tigera-policies.yaml" | absolute_url }}
```

### Above and beyond

- [Configure access to Calico Enterprise Manager]({{site.baseurl}}/getting-started/cnx/access-the-manager)
- [Get started with Kubernetes network policy]({{site.baseurl}}/security/kubernetes-network-policy)
- [Get started with Calico network policy]({{site.baseurl}}/security/calico-network-policy)
- [Enable default deny for Kubernetes pods]({{site.baseurl}}/security/kubernetes-default-deny)
