---
title: Google Kubernetes Engine (GKE)
---

### Big picture

Install {{site.prodname}} on a GKE managed Kubernetes cluster.

### Before you begin

- [Create a compatible GKE cluster](#create-a-compatible-gke-cluster)
- [Gather the necessary resources](#gather-required-resources)

#### Create a compatible GKE cluster

Ensure that your GKE cluster that meets the following requirements:

  - *Version `v1.14.5-gke.5` or greater.* At the time of writing, `v1.14.5-gke.5` is available through the "Rapid" [release channel](https://cloud.google.com/kubernetes-engine/docs/concepts/release-channels).

  - *[Intranode visibility](https://cloud.google.com/kubernetes-engine/docs/how-to/intranode-visibility) is enabled*.  This setting configures GKE to use the GKE CNI plugin, which is required.

  - *Network policy is disabled*. This avoids conflicts between other network policy providers in the cluster and {{site.prodname}}.

  - *Istio disabled*. The Istio setting on the GKE cluster prevents configuration of {{site.prodname}} application layer policy. To use Istio in your cluster, follow [this GKE tutorial](https://cloud.google.com/kubernetes-engine/docs/tutorials/installing-istio) to install the open source version of Istio on GKE.

  - *Master access to port 5443*. The GKE master must be able to access the {{site.prodname}} API server, which runs with host networking on port 5443.  For multi-zone clusters and clusters with the "master IP range" configured, you will need to add a GCP firewall rule to allow access to that port from the master.  For example, you could add a network tag to your nodes and then refer to that tag in a firewall rule, or allow based on your cluster's node CIDR.

#### Gather required resources

- Ensure that your Google account has sufficient IAM permissions.  To apply the {{site.prodname}} manifests requires permissions to create Kubernetes ClusterRoles and ClusterRoleBindings.  The easiest way to grant such permissions is to assign the "Kubernetes Engine Developer" IAM role to your user account as described in the [Creating Cloud IAM policies](https://cloud.google.com/kubernetes-engine/docs/how-to/iam) section of the GKE documentation.

> **Tip**: By default, GCP users often have permissions to create basic Kubernetes resources (such as Pods and Services) but lack the permissions to create ClusterRoles and other admin resources.  Even if you can create basic resources, it's worth verifying that you can create admin resources before continuing.

- Ensure that you have the [credentials for the Tigera private registry](/{{page.version}}/getting-started/#obtain-the-private-registry-credentials) and a [license key](/{{page.version}}/getting-started/#obtain-a-license-key).

### How to

1. [Install {{site.prodname}}](#install-tigera-secure-ee)
1. [Install the {{site.prodname}} license](#install-the-tigera-secure-ee-license)
1. [Secure {{site.prodname}} with network policy](#secure-tigera-secure-ee-with-network-policy)

#### Install {{site.prodname}}

1. [Configure a storage class for {{site.prodname}}.]()

1. Install the Tigera operators and custom resource definitions.

   ```
   kubectl create -f {{site.url}}/{{page.version}}/manifests/tigera-operator.yaml
   ```

1. Install your pull secret.

   ```
   kubectl create secret generic tigera-pull-secret \
       --from-file=.dockerconfigjson=<path/to/pull/secret> \
       --type=kubernetes.io/dockerconfigjson -n tigera-operator
   ```

1. Install the Tigera custom resources. For more information on configuration options available in this manifest, see [the installation reference](/{{page.version}}/reference/installation/api).

   ```
   kubectl create -f {{site.url}}/{{page.version}}/manifests/custom-resources.yaml
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
kubectl create -f {{site.url}}/{{page.version}}/manifests/tigera-policies.yaml
```

### Above and beyond

- [Configure access to the manager UI](/{{page.version}}/getting-started/access-the-manager)
- [Get started with Kubernetes network policy]({{site.url}}/{{page.version}}/security/kubernetes-network-policy)
- [Get started with Calico network policy]({{site.url}}/{{page.version}}/security/calico-network-policy)
- [Enable default deny for Kubernetes pods]({{site.url}}/{{page.version}}/security/kubernetes-default-deny)
