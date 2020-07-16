---
title: Install Calico Enterprise on a Rancher Kubernetes Engine cluster
description: Install Calico Enterprise on a Rancher Kubernetes Engine cluster.
canonical_url: '/getting-started/kubernetes/index'
---

### Big picture

Install {{side.prodname}} as the required CNI for networking and/or network policy on Rancher-deployed clusters.

### Before you begin

**Required**

- A compatible {% include open-new-window.html text='Rancher Kubernetes Engine cluster' url='https://rancher.com/docs/rke/latest/en/' %}
  - Configure your cluster with a {% include open-new-window.html text='Cluster Config File' url='https://rancher.com/docs/rancher/v2.x/en/cluster-provisioning/rke-clusters/options/#cluster-config-file' %} and specify {% include open-new-window.html text='no network plugin' url='https://rancher.com/docs/rke/latest/en/config-options/add-ons/network-plugins/' %} by setting `plugin: none` under `network` in your configuration file.

- RKE cluster meets the [{{side.prodname}} requirements](../requirements)

- [Credentials for the Tigera private registry and a license key](../../../getting-started/calico-enterprise).

- A `kubectl` environment with access to your cluster
  - Use {% include open-new-window.html text='Rancher kubectl Shell' url='https://rancher.com/docs/rancher/v2.x/en/cluster-admin/cluster-access/kubectl/' %} for access
  - Ensure you have the {% include open-new-window.html text='Kubeconfig file that was generated when you created the cluster' url='https://rancher.com/docs/rke/latest/en/installation/#save-your-files' %}.

- If using a Kubeconfig file locally, {% include open-new-window.html text='install and set up the Kubectl CLI tool' url='https://kubernetes.io/docs/tasks/tools/install-kubectl/' %}.

### How to

The geeky details of what you get:
{% include geek-details.html details='Policy:Calico,IPAM:Calico,CNI:Calico,Overlay:IPIP,Routing:BGP,Datastore:Kubernetes' %}

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

- The default networking uses IP in IP encapsulation with BPG routing. For all networking options, see [Determine best networking option]({{site.baseurl}}/networking/determine-best-networking).

**Recommended - Security**

- [Get started with {{site.prodname}} tiered network policy]({{site.baseurl}}/security/tiered-policy)
