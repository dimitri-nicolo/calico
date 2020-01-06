---
title: kubeadm Hosted Install
canonical_url: https://docs.tigera.io/v2.3/getting-started/kubernetes/installation/
---

This document outlines how to install {{site.tseeprodname}} on a cluster initialized with
[kubeadm](http://kubernetes.io/docs/getting-started-guides/kubeadm/).

## Requirements

For {{site.tseeprodname}} to be compatible with your kubeadm-created cluster:

* It must be running at least Kubernetes `v1.8.0`

* There should be no other CNI network configurations installed in /etc/cni/net.d (or equivalent directory)

* The kubeadm flag `--pod-network-cidr` must be set when creating the cluster with `kubeadm init`
  and the CIDR(s) specified with the flag must match {{site.tseeprodname}}'s IP pools. The default
  IP pool configured in {{site.tseeprodname}}'s manifests is `192.168.0.0/16`

* The CIDR specified with the kubeadm flag `--service-cidr` must not overlap with
  {{site.tseeprodname}}'s IP pools

  * The default CIDR for `--service-cidr` is `10.96.0.0/12`

  * The default IP pool configured in {{site.tseeprodname}}'s manifests is `192.168.0.0/16`

{% include {{page.version}}/cnx-k8s-apiserver-requirements.md %}

Note that kubeadm enables the aggregation layer by default.

You can create a cluster compatible with these manifests by following [the official kubeadm guide](http://kubernetes.io/docs/getting-started-guides/kubeadm/).

{% include {{page.version}}/load-docker.md %}

## Installing {{site.tseeprodname}}

### Installing {{site.tseeprodname}} with a Kubernetes-hosted etcd

As a non-production quick start, to install Calico with a single-node dedicated etcd cluster, running as a Kubernetes pod:

1. [Open calico.yaml in a new tab](1.7/calico.yaml){:target="_blank"}.

1. Copy the contents, paste them into a new file, and save the file as calico.yaml.

{% include {{page.version}}/cnx-cred-sed.md %}

   > **Note**: Refer to [Configuration options](../index#configuration-options) for additional
   > settings that can be modified in the manifest.
   {: .alert .alert-info}

1. Then apply the manifest.

   ```shell
   kubectl apply -f calico.yaml
   ```
1. Continue to [Installing the CNX Manager](#installing-the-cnx-manager)

### Installing {{site.tseeprodname}} with an existing etcd datastore

To install {{site.tseeprodname}}, configured to use an etcd that you have already set-up:

1. Ensure your cluster meets the [requirements](#requirements) (or recreate it if not).

2. Follow [the main etcd datastore instructions](../hosted).

1. Continue to [Installing the CNX Manager](#installing-the-cnx-manager)

### Installing {{site.tseeprodname}} with the Kubernetes API datastore

To install {{site.tseeprodname}}, configured to use the Kubernetes API as its sole data source:

1. Ensure your cluster meets the [requirements](#requirements) (or recreate it if not).

2. Follow [the main Kubernetes datastore instructions](../kubernetes-datastore).

1. Continue to [Installing the CNX Manager](#installing-the-cnx-manager)

## Installing the CNX Manager

1. [Open cnx-etcd.yaml in a new tab](../cnx/1.7/cnx-etcd.yaml){:target="_blank"}.

1. Copy the contents, paste them into a new file, and save the file as cnx.yaml.
   This is what subsequent instructions will refer to.

{% include {{page.version}}/cnx-mgr-install.md %}

{% include {{page.version}}/gs-next-steps.md %}
