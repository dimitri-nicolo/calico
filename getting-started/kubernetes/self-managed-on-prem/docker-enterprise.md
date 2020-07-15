---
title: Install Calico Enterprise on Docker Enterprise
description: Install Calico Enterprise using Docker EE for self-managed on-premises deployments.
canonical_url: /getting-started/kubernetes/self-managed-on-prem/docker-enterprise
---

### Big picture

Install Calico Enterprise in a Docker Enterprise deployed Kubernetes cluster.

### Before you begin

- [Create a compatible Docker EE cluster](#create-a-compatible-docker-ee-cluster)
- [Gather the necessary resources](#gather-required-resources)
- If using a private registry, familiarize yourself with this guide on [using a private registry]({{site.baseurl}}/getting-started/private-registry).

#### Create a compatible Docker EE cluster

- Ensure that you have a compatible [Docker Enterprise](https://docs.docker.com/ee/) installation on Linux, and refer to [Install UCP for Production](https://docs.docker.com/ee/ucp/admin/install/). For a test environment, a minimum of 3 nodes is required. For a production environment, additional nodes should be deployed.
  - During the installation of UCP, the installation will require the following flag `--unmanaged-cni`. This tells UCP to not install the default Calico networking plugin.

- Refer to [Docker Reference Architecture: Docker EE Best Practices and Design Considerations](https://success.docker.com/article/docker-ee-best-practices) for details.

- Ensure that your Docker Enterprise cluster also meets the {{site.prodname}} [system requirements](../requirements).

#### Gather required resources

- Ensure that you have the [credentials for the Tigera private registry and a license key](../../../getting-started/calico-enterprise)

- Download and install the UCP client bundle for accessing the cluster, instructions can be
  found at [Docker Universal Control Plane CLI-Based Access](https://docs.docker.com/ee/ucp/user-access/cli/).

- Install the Kubectl CLI tool. For more information please refer to [Install and Set Up kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/).

### How to

- [Install {{site.prodname}}](#install-calico-enterprise)
- [Install the {{site.prodname}} license](#install-the-calico-enterprise-license)
- [Secure {{site.prodname}} with network policy](#secure-calico-enterprise-with-network-policy)

#### Install {{site.prodname}}

1. [Configure a storage class for {{site.prodname}}.]({{site.baseurl}}/getting-started/create-storage)

1. Install Docker EE specific role bindings

   ```
   kubectl create -f {{ "/manifests/docker-enterprise/bindings.yaml" | absolute_url }}
   ```

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

   If pulling images directly from `quay.io/tigera`, you will likely want to use the credentials provided to you by your Tigera support representative. If using a private registry, use your private registry credentials instead.

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

#### Secure {{site.prodname}} with network policy

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
