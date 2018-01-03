---
title: kubeadm Hosted Install
---

This document outlines how to install {{site.prodname}} on a cluster initialized with
[kubeadm](http://kubernetes.io/docs/getting-started-guides/kubeadm/).  {{site.prodname}}
is compatible with kubeadm-created clusters, as long as the [requirements](#requirements) are met.

{% include {{page.version}}/load-docker.md %}

## Installation

For {{site.prodname}} to be compatible with your kubeadm-created cluster:

1. Calico installations require kubeadm initialized clusters to have set a `--pod-network-cidr`
   which should match the value set by `CALICO_IPV4POOL_CIDR` in the manifest:

   ```yaml
   - name: CALICO_IPV4POOL_CIDR
     value: "192.168.0.0/16"
   ```

   This value can be changed to any CIDR that does not overlap the cluster's service CIDR.
   The manifests are currently setup to use the CIDR `192.168.0.0/16` so you would include
   `--pod-network-cidr=192.168.0.0/16` in the `kubeadm init` call (either as a parameter
   or in a config file).

1. To use CNX Manager, authentication needs to be set up on the cluster in a supported way.
   The [authentication documentation](../../../../reference/cnx/authentication) lists
   the supported methods and references the Kubernetes documentation for how to configure
   them.

   This [kubeadm config file](../cnx/demo-manifests/kubeadm.yaml) is an example that
   configures Google OIDC login using the email address as the username.

   Run kubeadm using the config file as follows.
   ```
   kubeadm init --config=kubeadm.yaml
   ```

1. Ensure kubeadm sets the Kubernetes API server up to support aggregation.  Recent
   versions of kubeadm should do this by default.  Please refer to the kubeadm
   documentation for full details on how to do this.

   The example kubeadm config file linked above does this explicitly.

You can create a cluster compatible with these manifests by following [the official kubeadm guide](http://kubernetes.io/docs/getting-started-guides/kubeadm/).

## Installing {{site.prodname}} with a Kubernetes-hosted etcd

Download the [calico with single node etcd manifest](1.7/calico.yaml) and
update it with the path to your private docker registry.
Substitute`mydockerregistry:5000` with the location of your docker registry.

1. Ensure your cluster meets the [requirements](#requirements) (or recreate it if not).

```
sed -i -e 's/<YOUR_PRIVATE_DOCKER_REGISTRY>/mydockerregistry:5000/g' calico.yaml
```

Then apply the manifest.

```shell
kubectl apply -f calico.yaml
```

To install {{site.prodname}}, configured to use an etcd that you have already set-up:

1. Ensure your cluster meets the [requirements](#requirements) (or recreate it if not).

2. Follow [the main etcd datastore instructions](../hosted).

Download the calico manifest and update it with the path to your private docker registry.  Substitute
`mydockerregistry:5000` with the location of your docker registry.

```
sed -i -e 's/<YOUR_PRIVATE_DOCKER_REGISTRY>/mydockerregistry:5000/g' calico.yaml
```

Then apply the manifests.

```shell
kubectl apply -f {{site.url}}/{{page.version}}/getting-started/kubernetes/installation/hosted/rbac-kdd.yaml
kubectl apply -f calico.yaml
```

>[Click here to view the above RBAC yaml directly.](../rbac-kdd.yaml)
>
>[Click here to view the Calico yaml.](../kubernetes-datastore/calico-networking/1.7/calico.yaml)

## Adding Tigera CNX

Now you've installed Calico with the enhanced CNX node agent, you're ready to
[add CNX Manager](../cnx/cnx).

## Using calicoctl in a kubeadm cluster

The simplest way to use calicoctl in kubeadm is by running it as a pod.
See [Installing calicoctl as a container](/{{page.version}}/usage/calicoctl/install#installing-calicoctl-as-a-container) for more information.
