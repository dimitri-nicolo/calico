---
title: Standard
description: Install Calico Enterprise on a deployed Kubernetes cluster for on-premises deployments.
canonical_url: /getting-started/kubernetes/self-managed-on-prem/generic-install
---

### Big picture

Install {{site.prodname}} on a deployed Kubernetes cluster.

### Features

This how-to guide features the following {{site.prodname}} features:

- **operator.tigera.io installation APIs** 
- **{{site.prodname}} resources**

### Concepts

#### What's installed out of the box

When you install {{site.prodname}} on your cluster for the first time, you get the full {{site.prodname}} product with the following.

{% include content/default-install.md install="default" %}

### Before you begin

**Required**

- [License and pull secret to access Tigera private registry]({{site.baseurl}}/getting-started/calico-enterprise)

**Recommended**

- [Options for installing {{site.prodname}}]({{site.baseurl}}/getting-started/options-install)
- If you are installing {{site.prodname}} from a private registry, see [using a private registry]({{site.baseurl}}/getting-started/private-registry).

### How to

- [Install Calico Enterprise](#install-calico-enterprise)
- [Install Calico Enterprise license](#install-calico-enterprise-license)
- [Secure Calico Enterprise with network policy](#secure-calico-enterprise-with-network-policy)

#### Install {{site.prodname}}

1. [Configure storage for {{site.prodname}}]({{site.baseurl}}/getting-started/create-storage).

1. Install the Tigera operators and custom resource definitions.

   ```
   kubectl create -f {{ "/manifests/tigera-operator.yaml" | absolute_url }}
   ```

1. Install your pull secret.

   If pulling images directly from `quay.io/tigera`, you will likely want to use the credentials provided to you by your Tigera support representative. If using a private registry, use your private registry credentials.

   ```
   kubectl create secret generic tigera-pull-secret \
       --from-file=.dockerconfigjson=<path/to/pull/secret> \
       --type=kubernetes.io/dockerconfigjson -n tigera-operator
   ```
1. [Optional] If your cluster architecture requires any custom [{{site.prodname}} resources]({{site.baseurl}}/reference/resources) to function at startup, install them now using [calicoctl]({{site.baseurl}}/reference/calicoctl/overview).

1. Install the Tigera custom resources. For more information on configuration options available, see [the installation reference]({{site.baseurl}}/reference/installation/api).

   ```
   kubectl create -f {{ "/manifests/custom-resources.yaml" | absolute_url }}
   ```

   You can now monitor progress with the following command:

   ```
   watch kubectl get tigerastatus
   ```

   Wait until the `apiserver` shows a status of `Available`, then proceed to the next section.

#### Install {{site.prodname}} license

Install the {{site.prodname}} license provided to you by Tigera.

```
kubectl create -f </path/to/license.yaml>
```

You can now monitor progress with the following command:

```
watch kubectl get tigerastatus
```

When all components show a status of `Available`, continue to the next section.


#### Secure {{site.prodname}} with network policy

Install the following network policies to secure {{site.prodname}} component communications.

```
kubectl create -f {{ "/manifests/tigera-policies.yaml" | absolute_url }}
```

### Next steps

**Required**

- [Install and configure CLIs]({{site.baseurl}}/getting-started/clis/calicoctl/configure/kdd)

**Recommended**

- [Configure access to {{site.prodname}} Manager UI]({{site.baseurl}}/getting-started/cnx/access-the-manager)
- [Configure user authentication and log in]({{site.baseurl}}/getting-started/cnx/create-user-login)

**Recommended - Networking**

- If you are using the default BGP networking with full-mesh node-to-node peering with no encapsulation, go to [Configure BGP peering]({{site.baseurl}}/networking/bgp) to get traffic flowing between pods.
- If you are unsure about networking options, or want to implement encapsulation (overlay networking), see [Determine best networking option]({{site.baseurl}}/networking/determine-best-networking).

**Recommended - Security**

- [Get started with {{site.prodname}} tiered network policy]({{site.baseurl}}/security/tiered-policy)
