---
title: Microsoft Azure Kubernetes Service (AKS)
description: Enable Calico network policy in AKS.
canonical_url: '/getting-started/kubernetes/aks'
---

### Big picture

Install {{site.prodname}} on an AKS managed Kubernetes cluster.

### Before you begin

- [Gather required resources](#gather-required-resources)
- [Create a compatible AKS cluster](#create-a-compatible-aks-cluster)
- If using a private registry, familiarize yourself with this guide on [using a private registry]({{site.baseurl}}/getting-started/private-registry)
- Review [network requirements]({{site.baseurl}}/getting-started/kubernetes/requirements#network-requirements) to ensure network access is properly configured for {{site.prodname}} components

#### Gather required resources

- Ensure that your Azure account has IAM permissions to create Kubernetes ClusterRoles and ClusterRoleBindings.
This is required for applying manifests. The easiest way to grant permissions is to assign the "Azure Kubernetes Service Cluster Admin Role" to your user account. For help, see {% include open-new-window.html text='AKS access control' url='https://docs.microsoft.com/en-us/azure/aks/control-kubeconfig-access' %}.
- Ensure that you have the [credentials for the Tigera private registry and a license key]({{site.baseurl}}/getting-started/calico-enterprise)

#### Create a compatible AKS cluster

Ensure that your AKS cluster meets the following requirements.

  - Azure CNI networking plugin is used with {% include open-new-window.html text='transparent mode' url='https://docs.microsoft.com/en-us/azure/aks/faq#what-is-azure-cni-transparent-mode-vs-bridge-mode' %}
  - Network policy is not set.
    This avoids conflicts between other network policy providers in the cluster and {{site.prodname}}.

**Using Azure CLI**

AKS cluster created using the {% include open-new-window.html text='Azure CLI' url='https://docs.microsoft.com/en-us/cli/azure/aks?view=azure-cli-latest' %} are created with transparent mode by default. Ensure cluster is started with the option `--network-plugin azure`

##### Using Azure Resource Manager (ARM) template

> **Note**: {% include open-new-window.html text='ARM templates' url='https://azure.microsoft.com/en-us/resources/templates/?resourceType=Microsoft.Containerservice&term=AKS' %} must create resources using {% include open-new-window.html text='Microsoft.ContainerService apiVersion 2020-02-01' url='https://docs.microsoft.com/en-us/azure/templates/microsoft.containerservice/2020-02-01/managedclusters' %} or newer
{: .alert .alert-info}

1. Enable network mode using the `aks-preview` extension.

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

### How to

1. [Install {{site.prodname}}](#install-calico-enterprise)
1. [Install the {{site.prodname}} license](#install-the-calico-enterprise-license)
1. [Secure {{site.prodname}} with network policy](#secure-calico-enterprise-with-network-policy)

{% include content/install-aks.md clusterType="standalone" %}

The geeky details of what you get:
{% include geek-details.html details='Policy:Calico,IPAM:Azure,CNI:Azure,Overlay:No,Routing:VPC Native,Datastore:Kubernetes' %}

### Next steps

- [Configure access to {{site.prodname}} Enterprise Manager]({{site.baseurl}}/getting-started/cnx/access-the-manager)
- {% include open-new-window.html text='Video: Everything you need to know about Kubernetes networking on Azure' url='https://www.projectcalico.org/everything-you-need-to-know-about-kubernetes-networking-on-azure/' %}
- [Get started with Kubernetes network policy]({{ site.baseurl }}/security/kubernetes-network-policy)
- [Get started with {{site.prodname}} network policy]({{ site.baseurl }}/security/calico-network-policy)
- [Enable default deny for Kubernetes pods]({{ site.baseurl }}/security/kubernetes-default-deny)
