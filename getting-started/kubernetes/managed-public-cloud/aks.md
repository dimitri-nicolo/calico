---
title: Microsoft Azure Kubernetes Service (AKS)
description: Enable Calico network policy in AKS.
---

### Big picture

Install {{site.prodname}} on an AKS managed Kubernetes cluster.

### Before you begin

- [Gather required resources](#gather-required-resources)
- [Create a compatible AKS cluster](#create-a-compatible-aks-cluster)
- If using a private registry, familiarize yourself with this guide on [using a private registry]({{site.baseurl}}/getting-started/private-registry).
- Review [network requirements]({{site.baseurl}}/getting-started/kubernetes/requirements#network-requirements) to ensure network access is properly configured for {{site.prodname}} components.

#### Gather required resources

- Ensure that your Azure account has IAM permissions to create Kubernetes ClusterRoles and ClusterRoleBindings. This is required for applying manifests. The easiest way to grant permissions is to assign the "Azure Kubernetes Service Cluster Admin Role" to your user account. For help, see [AKS access control](https://docs.microsoft.com/en-us/azure/aks/control-kubeconfig-access).
- Ensure that you have the [credentials for the Tigera private registry and a license key]({{site.baseurl}}/getting-started/calico-enterprise)

#### Create a compatible AKS cluster

Ensure that your AKS cluster meets the following requirements.

  - Azure CNI networking plugin is used with transparent mode
  - Network policy is not set. This is to avoid conflicts between other network policy providers in the cluster and {{site.prodname}}.

##### Using Azure Resource Manager (ARM) template

AKS clusters created using [ARM templates](https://azure.microsoft.com/en-us/resources/templates/?resourceType=Microsoft.Containerservice&term=AKS) can leverage native support for enabling Azure CNI plugin with transparent mode

> NOTE: ARM templates must create resources using [Microsoft.ContainerService apiVersion 2020-02-01](https://docs.microsoft.com/en-us/azure/templates/microsoft.containerservice/2020-02-01/managedclusters) or newer
{: .alert .alert-info}

1. Enable network mode using the `aks-preview` extension

   ```sh
   az extension add --name aks-preview
   az feature register -n AKSNetworkModePreview --namespace Microsoft.ContainerService
   az provider register -n Microsoft.ContainerService
   ```

2. Create cluster using `az deployment group create` with the values:

   ```json
   "networkProfile": {
    "networkPlugin": "azure",
    "networkMode": "transparent"
   }
   ```

##### Using Azure CLI

1. Create cluster using `az aks create` with the option `--network-plugin azure`

1. Create the following daemon set to update Azure CNI plugin so that it operates in transparent mode

   ```sh
   kubectl apply -f https://raw.githubusercontent.com/jonielsen/istioworkshop/master/03-TigeraSecure-Install/bridge-to-transparent.yaml
   ```

> **Note**: After the manifest is applied, nodes are rebooted one by one with the status, `SchedulingDisabled`. Wait until all nodes are in `Ready` status before continuing to the next step.
{: .alert .alert-info}

### How to

1. [Install {{site.prodname}}](#install-calico-enterprise)
1. [Install the {{site.prodname}} license](#install-the-calico-enterprise-license)
1. [Secure {{site.prodname}} with network policy](#secure-calico-enterprise-with-network-policy)

#### Install {{site.prodname}}

1. [Configure a storage class for {{site.prodname}}.]({{site.baseurl}}/getting-started/create-storage)

1. Install the Tigera operators and custom resource definitions.

   ```
   kubectl create -f {{ "/manifests/tigera-operator.yaml" | absolute_url }}
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
   kubectl create -f {{ "/manifests/aks/custom-resources.yaml" | absolute_url }}
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

> **Note**: The Calico network policy feature can only be enabled when the cluster is created. You can't enable Calico network policy on an existing AKS cluster.
{: .alert .alert-info}

The geeky details of what you get:
{% include geek-details.html details='Policy:Calico,IPAM:Azure,CNI:Azure,Overlay:No,Routing:VPC Native,Datastore:Kubernetes' %}

### Above and beyond

- [Video: Everything you need to know about Kubernetes networking on Azure](https://www.projectcalico.org/everything-you-need-to-know-about-kubernetes-networking-on-azure/)
- [Install calicoctl command line tool]({{ site.baseurl }}/getting-started/clis/calicoctl/install)
- [Get started with Kubernetes network policy]({{ site.baseurl }}/security/kubernetes-network-policy)
- [Get started with Calico network policy]({{ site.baseurl }}/security/calico-network-policy)
- [Enable default deny for Kubernetes pods]({{ site.baseurl }}/security/kubernetes-default-deny)
