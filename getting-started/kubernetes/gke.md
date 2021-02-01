---
title: Google Kubernetes Engine (GKE)
description: Enable Calico network policy in GKE.
canonical_url: '/getting-started/kubernetes/gke'
---

### Big picture

Install {{site.prodname}} on a GKE managed Kubernetes cluster.

### Before you begin

- [Create a compatible GKE cluster](#create-a-compatible-gke-cluster)
- [Gather the necessary resources](#gather-required-resources)
- If using a private registry, familiarize yourself with this guide on [using a private registry]({{site.baseurl}}/getting-started/private-registry)
- Review [network requirements]({{site.baseurl}}/getting-started/kubernetes/requirements#network-requirements) to ensure network access is properly configured for {{site.prodname}} components

#### Create a compatible GKE cluster

Ensure that your GKE cluster that meets the following requirements:

  - *Version `1.18.x` or newer

  - *{% include open-new-window.html text='Intranode visibility' url='https://cloud.google.com/kubernetes-engine/docs/how-to/intranode-visibility' %} is enabled*.  This setting configures GKE to use the GKE CNI plugin, which is required.

  - *Network policy is disabled*. This avoids conflicts between other network policy providers in the cluster and {{site.prodname}}.

  - *Istio disabled*. The Istio setting on the GKE cluster prevents configuration of {{site.prodname}} application layer policy. To use Istio in your cluster, follow {% include open-new-window.html text='this GKE tutorial' url='https://cloud.google.com/kubernetes-engine/docs/tutorials/installing-istio' %} to install the open source version of Istio on GKE.

  - *Master access to port 5443*. The GKE master must be able to access the {{site.prodname}} API server, which runs with host networking on port 5443.  For multi-zone clusters and clusters with the "master IP range" configured, you will need to add a GCP firewall rule to allow access to that port from the master.  For example, you could add a network tag to your nodes and then refer to that tag in a firewall rule, or allow based on your cluster's node CIDR.

#### Gather required resources

- Ensure that your Google account has sufficient IAM permissions.  To apply the {{site.prodname}} manifests requires permissions to create Kubernetes ClusterRoles and ClusterRoleBindings.  The easiest way to grant such permissions is to assign the "Kubernetes Engine Developer" IAM role to your user account as described in the {% include open-new-window.html text='Creating Cloud IAM policies' url='https://cloud.google.com/kubernetes-engine/docs/how-to/iam' %} section of the GKE documentation.

> **Tip**: By default, GCP users often have permissions to create basic Kubernetes resources (such as Pods and Services) but lack the permissions to create ClusterRoles and other admin resources.  Even if you can create basic resources, it's worth verifying that you can create admin resources before continuing.

- Ensure that you have the [credentials for the Tigera private registry]({{site.baseurl}}/getting-started/calico-enterprise#get-private-registry-credentials-and-license-key) and a [license key]({{site.baseurl}}/getting-started/calico-enterprise#get-private-registry-credentials-and-license-key).

### How to

1. [Install {{site.prodname}}](#install-calico-enterprise)
1. [Install the {{site.prodname}} license](#install-the-calico-enterprise-license)
1. [Secure {{site.prodname}} with network policy](#secure-calico-enterprise-with-network-policy)

#### Install {{site.prodname}}

1. [Configure a storage class for {{site.prodname}}.]({{site.baseurl}}/getting-started/create-storage)

1. Install the Tigera operator and custom resource definitions.

   ```
   kubectl create -f {{ "/manifests/tigera-operator.yaml" | absolute_url }}
   ```

1. Install the Prometheus operator and related custom resource definitions. The Prometheus operator will be used to deploy Prometheus server and Alertmanager to monitor {{site.prodname}} metrics.

   > **Note**: If you have an existing Prometheus operator in your cluster that you want to use, skip this step. To work with {{site.prodname}}, your Prometheus operator must be v0.30.0 or higher.
   {: .alert .alert-info}

   ```
   kubectl create -f {{ "/manifests/tigera-prometheus-operator.yaml" | absolute_url }}
   ```

1. Install your pull secret.

   If pulling images directly from `quay.io/tigera`, you will likely want to use the credentials provided to you by your Tigera support representative. If using a private registry, use your private registry credentials instead.

   ```
   kubectl create secret generic tigera-pull-secret \
       --type=kubernetes.io/dockerconfigjson -n tigera-operator \
       --from-file=.dockerconfigjson=<path/to/pull/secret>
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


#### Secure {{site.prodname}} with network policy

To secure {{site.prodname}} component communications, install the following set of network policies.

```
kubectl create -f {{ "/manifests/tigera-policies.yaml" | absolute_url }}
```

The geeky details of what you get:
{% include geek-details.html details='Policy:Calico,IPAM:Host Local,CNI:Calico,Overlay:No,Routing:VPC Native,Datastore:Kubernetes' %}

### Next steps

- [Configure access to {{site.prodname}} Manager]({{site.baseurl}}/getting-started/cnx/access-the-manager)
- {% include open-new-window.html text='Video: Everything you need to know about Kubernetes networking on Google cloud' url='https://www.projectcalico.org/everything-you-need-to-know-about-kubernetes-networking-on-google-cloud/' %}
- [Get started with Kubernetes network policy]({{ site.baseurl }}/security/kubernetes-network-policy)
- [Get started with {{site.prodname}} network policy]({{ site.baseurl }}/security/calico-enterprise-policy)
- [Enable default deny for Kubernetes pods]({{ site.baseurl }}/security/kubernetes-default-deny)
