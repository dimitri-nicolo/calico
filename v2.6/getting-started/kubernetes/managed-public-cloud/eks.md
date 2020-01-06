---
title: Amazon Elastic Kubernetes Service (EKS)
redirect_from: latest/getting-started/kubernetes/managed-public-cloud/eks
---

### Big picture

Install {{ site.prodname }} in EKS managed Kubernetes service.

### Before you begin

- Ensure that you have an EKS cluster without Calico installed and with [platform version](https://docs.aws.amazon.com/eks/latest/userguide/platform-versions.html) at least eks.2 (for aggregated API server support).

- Ensure that you have the [credentials for the Tigera private registry](/{{page.version}}/getting-started/#obtain-the-private-registry-credentials) and a [license key](/{{page.version}}/getting-started/#obtain-a-license-key).

- If using a private registry, familiarize yourself with this guide on [using a private registry]({{site.baseurl}}/{{page.version}}/getting-started/private-registry).

### How to

1. [Install {{site.prodname}}](#install-calico-enterprise)
1. [Install the {{site.prodname}} license](#install-the-calico-enterprise-license)
1. [Secure {{site.prodname}} with network policy](#secure-calico-enterprise-with-network-policy)

#### Install {{site.prodname}}

1. [Configure a storage class for {{site.prodname}}.](/{{page.version}}/getting-started/create-storage)

1. Install the Tigera operators and custom resource definitions.

   ```
   kubectl create -f {{site.url}}/{{page.version}}/manifests/tigera-operator.yaml
   ```

1. Install your pull secret.

   If pulling images directly from `quay.io/tigera`, you will likely want to use the credentials provided to you by your Tigera support representative. If using a private registry, use your private registry credentials instead.

   ```
   kubectl create secret generic tigera-pull-secret \
       --from-file=.dockerconfigjson=<path/to/pull/secret> \
       --type=kubernetes.io/dockerconfigjson -n tigera-operator
   ```

1. Install the Tigera custom resources. For more information on configuration options available in this manifest, see [the installation reference](/{{page.version}}/reference/installation/api).

   ```
   kubectl create -f {{site.url}}/{{page.version}}/manifests/eks/custom-resources.yaml
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
