---
title: Amazon Elastic Kubernetes Service (EKS)
---

### Big picture

Install {{ site.prodname }} in EKS managed Kubernetes service.

### Before you begin

Ensure that you have an EKS cluster without Calico installed and with [platform version](https://docs.aws.amazon.com/eks/latest/userguide/platform-versions.html) at least eks.2 (for aggregated API server support).

### How to

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

1. Install the Tigera custom resources.

   ```
   kubectl create -f {{site.url}}/master/manifests/eks/custom-resources.yaml
   ```

   You can now monitor progress with the following command:

   ```
   watch kubectl get tigerastatus
   ```

   Wait until the `apiserver` is showing a status of `Available`, then proceed to the next section.

#### Install the {{site.prodname}} license

In order to use {{site.prodname}}, you must install the license provided to you by Tigera.

```
kubectl create -f </path/to/license.yaml>
```

You can now monitor progress with the following command:

```
watch kubectl get tigerastatus
```

When it shows all components with status `Available`, proceed to the next section.


#### Secure Tigera Secure EE with network policy

To secure the components which make up Tigera Secure EE, install the following set of network policies.

```
kubectl create -f {{site.url}}/master/manifests/tigera-policies.yaml
```

### Above and beyond

- [Install calicoctl command line tool]({{site.url}}/{{page.version}}/getting-started/calicoctl/install)
- [Get started with Kubernetes network policy]({{site.url}}/{{page.version}}/security/kubernetes-network-policy)
- [Get started with Calico network policy]({{site.url}}/{{page.version}}/security/calico-network-policy)
