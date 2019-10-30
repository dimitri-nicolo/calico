---
title: Microsoft Azure Kubernetes Service (AKS)
---

### Big picture

Install {{site.prodname}} on an AKS managed Kubernetes cluster.

> **Note**: AKS support for {{site.prodname}} is currently in technical preview
   and is suitable for non-production clusters only.
{: .alert .alert-info}

### Before you begin

- [Create a compatible AKS cluster](#create-a-compatible-aks-cluster)
- [Gather the necessary resources](#gather-required-resources)
- [Enable transparent mode](#enable-transparent-mode)

#### Create a compatible AKS cluster

Ensure that your AKS cluster that meets the following requirements.

   - *Azure CNI networking is used*. The cluster must be started with the option `--network-plugin azure`.

   - *Network policy is disabled*. This is to avoid conflicts between other network policy providers in the cluster and {{site.prodname}}.

#### Gather required resources

- Ensure that your Azure account has IAM permissions to create Kubernetes ClusterRoles and ClusterRoleBindings. This is required for applying manifests. The easiest way to grant permissions is to assign the "Azure Kubernetes Service Cluster Admin Role" to your user account. For help, see [AKS access control](https://docs.microsoft.com/bs-latn-ba/azure/aks/control-kubeconfig-access).


- Ensure that you have the [credentials for the Tigera private registry](/{{page.version}}/getting-started/#obtain-the-private-registry-credentials) and a [license key](/{{page.version}}/getting-started/#obtain-a-license-key).

#### Enable transparent mode

> **Note**: In a future release, AKS will support cluster creation with the Azure CNI plugin in transparent mode. Until then, follow these manual steps to work around this limitation.
{: .alert .alert-info}

1. Create the following daemon set to update the Azure CNI plugin so that it operates in transparent mode.

   ```
   kubectl apply -f https://raw.githubusercontent.com/jonielsen/istioworkshop/master/03-TigeraSecure-Install/bridge-to-transparent.yaml
   ```
> **Note**: After the manifest is applied, nodes are rebooted one by one with the status, `SchedulingDisabled`. Wait until all nodes are in `Ready` status before continuing to the next step.
{: .alert .alert-info}

### How to

1. [Install {{site.prodname}}](#install-tigera-secure-ee)
1. [Install the {{site.prodname}} license](#install-the-tigera-secure-ee-license)
1. [Secure {{site.prodname}} with network policy](#secure-tigera-secure-ee-with-network-policy)

#### Install {{site.prodname}}

1. [Configure a storage class for {{site.prodname}}.]()

1. Install the Tigera operators and custom resource definitions.

   ```
   kubectl create -f {{site.url}}/master/manifests/tigera-operator.yaml
   ```

1. Install your pull secret.

   ```
   kubectl create secret generic tigera-pull-secret \
       --from-file=.dockerconfigjson=<path/to/pull/secret> \
       --type=kubernetes.io/dockerconfigjson -n tigera-operator
   ```

1. Install the Tigera custom resources. For more information on configuration options available in this manifest, see [the installation reference](/{{page.version}}/reference/installation/api).

   ```
   kubectl create -f {{site.url}}/master/manifests/aks/custom-resources.yaml
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
kubectl create -f {{site.url}}/master/manifests/tigera-policies.yaml
```

### Above and beyond

- [Access the manager UI](/{{page.version}}/getting-started/access-the-manager)
- [Get started with Kubernetes network policy]({{site.url}}/{{page.version}}/security/kubernetes-network-policy)
- [Get started with Calico network policy]({{site.url}}/{{page.version}}/security/calico-network-policy)
- [Enable default deny for Kubernetes pods]({{site.url}}/{{page.version}}/security/kubernetes-default-deny)
