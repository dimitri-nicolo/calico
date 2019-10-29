---
title: Installing Tigera Secure EE on Docker Enterprise
canonical_url: https://docs.tigera.io/master/getting-started/kubernetes/self-managed-on-prem/docker-enterprise
---

### Big picture

Install Tigera Secure in a Docker Enterprise deployed Kubernetes cluster.

### Before you begin

- [Create a compatible Docker EE cluster](#create-a-compatible-docker-ee-cluster)
- [Gather the necessary resources](#gather-required-resources)

#### Create a compatible Docker EE cluster

- Ensure that your Docker Enterprise cluster meets the requirements by following the [Deploy Docker Enterprise on Linux](https://docs.docker.com/v17.09/datacenter/install/linux/) instructions and refering to [Install UCP for Production](https://docs.docker.com/ee/ucp/admin/install/). For a test environment, a minimum of 3 nodes is required. For a production environment, additional nodes should be deployed.
  - During the installation of UCP, the installation will require the following flag `--unmanaged-cni`. This tells UCP to not install the default Calico networking plugin.

- Refer to [Docker Reference Architecture: Docker EE Best Practices and Design Considerations](https://success.docker.com/article/docker-ee-best-practices) for details.

- Ensure that your Docker Enterprise cluster also meets the {{site.prodname}} [system requirements](../requirements).

#### Gather required resources

- Ensure that you have the [credentials for the Tigera private registry](../../../getting-started/#obtain-the-private-registry-credentials)
  and a [license key](../../../getting-started/#obtain-a-license-key).

- Download and install the UCP client bundle for accessing the cluster, instructions can be
  found at [Docker Universal Control Plane CLI-Based Access](https://docs.docker.com/ee/ucp/user-access/cli/).

- Install the Kubectl CLI tool. For more information please refer to [Install and Set Up kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/).

### How to

- [Install {{site.prodname}}](#install-tigera-secure-ee)
- [Install the {{site.prodname}} license](#install-the-tigera-secure-ee-license)
- [Secure {{site.prodname}} with network policy](#secure-tigera-secure-ee-with-network-policy)

#### Install {{site.prodname}}

1. [Configure a storage class for {{site.prodname}}.]()

1. Install Docker EE specific role bindings

   ```
   kubectl create -f {{site.url}}/master/manifests/docker-enterprise/bindings.yaml
   ```

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
   kubectl create -f {{site.url}}/master/manifests/custom-resources.yaml
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
