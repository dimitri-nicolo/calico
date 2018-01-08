---
title: kubeadm Hosted Install
---

This document outlines how to install {{site.prodname}} on a cluster initialized with
[kubeadm](http://kubernetes.io/docs/getting-started-guides/kubeadm/).  {{site.prodname}}
is compatible with kubeadm-created clusters, as long as the [requirements](#requirements) are met.

{% include {{page.version}}/load-docker.md %}

## Requirements

For {{site.prodname}} to be compatible with your kubeadm-created cluster:

* It must be running at least Kubernetes v1.8

* There should be no other CNI network configurations installed in /etc/cni/net.d (or equivalent directory)

* The kubeadm flag `--pod-network-cidr` must be set when creating the cluster with `kubeadm init`
  and the CIDR(s) specified with the flag must match {{site.prodname}}'s IP pools. The default
  IP pool configured in {{site.prodname}}'s manifests is `192.168.0.0/16`

* The CIDR specified with the kubeadm flag `--service-cidr` must not overlap with
  {{site.prodname}}'s IP pools

  * The default CIDR for `--service-cidr` is `10.96.0.0/12`

  * The default IP pool configured in {{site.prodname}}'s manifests is `192.168.0.0/16`

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
[add CNX Manager](../cnx/cnx).  Note that you can skip the step to [Configure
the aggregation
layer](https://kubernetes.io/docs/tasks/access-kubernetes-api/configure-aggregation-layer/),
because kubeadm already does that.

## Using calicoctl in a kubeadm cluster

The simplest way to use calicoctl in kubeadm is by running it as a pod.
See [Installing calicoctl as a container](/{{page.version}}/usage/calicoctl/install#installing-calicoctl-as-a-container) for more information.
