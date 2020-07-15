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
- If using a private registry, familiarize yourself with this guide on [using a private registry]({{site.baseurl}}/getting-started/private-registry)

#### Create a compatible Rancher cluster

- Ensure that your {% include open-new-window.html text='Rancher Kubernetes Engine cluster' url='https://rancher.com/docs/rke/latest/en/' %} meets the [requirements](../requirements) for {{site.prodname}}.
  - Configure your cluster with a {% include open-new-window.html text='Cluster Config File' url='https://rancher.com/docs/rancher/v2.x/en/cluster-provisioning/rke-clusters/options/#cluster-config-file' %} and specify {% include open-new-window.html text='no network plugin' url='https://rancher.com/docs/rke/latest/en/config-options/add-ons/network-plugins/' %} by setting `plugin: none` under `network` in your configuration file.

#### Gather required resources

- Ensure that you have the [credentials for the Tigera private registry and a license key](../../../getting-started/calico-enterprise).

- Ensure you have a kubectl environment with access to your cluster
  - Use {% include open-new-window.html text='Rancher kubectl Shell' url='https://rancher.com/docs/rancher/v2.x/en/cluster-admin/cluster-access/kubectl/' %} for access
  - Ensure you have the {% include open-new-window.html text='Kubeconfig file that was generated when you created the cluster' url='https://rancher.com/docs/rke/latest/en/installation/#save-your-files' %}.

- If using a Kubeconfig file locally, {% include open-new-window.html text='install and set up the Kubectl CLI tool' url='https://kubernetes.io/docs/tasks/tools/install-kubectl/' %}.

### How to

- [Install {{site.prodname}}](#install-calico-enterprise)
- [Install the {{site.prodname}} license](#install-the-calico-enterprise-license)
- [Secure {{site.prodname}} components with network policy](#secure-calico-enterprise-components-with-network-policy)


#### Install {{site.prodname}}

1. [Configure a storage class for {{site.prodname}}.]({{site.baseurl}}/getting-started/create-storage).

1. Install the Tigera operator and custom resource definitions.

   ```
   kubectl create -f {{ "/manifests/tigera-operator.yaml" | absolute_url }}
   ```

1. (Optional) If you have an existing Prometheus operator in your cluster that you want to use, continue to the next step. Otherwise, install the Prometheus operator and related custom resource definitions with the command below. The Prometheus operator will be used to deploy Prometheus server and Alertmanager to monitor {{site.prodname}} metrics.

   ```
   kubectl create -f {{ "/manifests/tigera-prometheus-operator.yaml" | absolute_url }}
   ```

   > **Note**: If you plan to use your own Prometheus operator with {{site.prodname}}, please ensure it is v0.30.0 or higher.
   {: .alert .alert-info}

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

### Next steps

**Recommended**

- [Configure access to {{site.prodname}} Manager UI]({{site.baseurl}}/getting-started/cnx/access-the-manager)
- [Configure user authentication and log in]({{site.baseurl}}/getting-started/cnx/create-user-login)

**Recommended - Networking**

- If you are using the default BGP networking with full-mesh node-to-node peering with no encapsulation, go to [Configure BGP peering]({{site.baseurl}}/networking/bgp) to get traffic flowing between pods.
- If you are unsure about networking options, or want to implement encapsulation (overlay networking), see [Determine best networking option]({{site.baseurl}}/networking/determine-best-networking).

**Recommended - Security**

- [Get started with {{site.prodname}} tiered network policy]({{site.baseurl}}/security/tiered-policy)
