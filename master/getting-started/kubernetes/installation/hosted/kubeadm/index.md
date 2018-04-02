---
title: kubeadm Hosted Install
canonical_url: 'https://docs.projectcalico.org/v3.0/getting-started/kubernetes/installation/hosted/kubeadm/'
---

This document outlines how to install {{site.prodname}} on a cluster initialized with 
[kubeadm](http://kubernetes.io/docs/getting-started-guides/kubeadm/).  

{% include {{page.version}}/load-docker-intro.md %}

{% include {{page.version}}/load-docker-our-reg.md yaml="calico" %}

{% include {{page.version}}/load-docker.md yaml="calico" orchestrator="kubernetes" %}

## Installing {{site.prodname}} with a Kubernetes-hosted etcd

As a non-production quick start, to install {{site.prodname}} with a single-node dedicated etcd cluster,
running as a Kubernetes pod:

1. If you have an existing cluster, ensure that it meets the {{site.prodname}} [system requirements](../../../requirements). Otherwise, you can create a cluster compatible with these manifests by following [the official kubeadm guide](http://kubernetes.io/docs/getting-started-guides/kubeadm/).

2. Apply the single-node etcd manifest:
   
   ```shell
   kubectl apply -f {{site.url}}/{{page.version}}/getting-started/kubernetes/installation/hosted/kubeadm/1.7/calico.yaml
   ```
   
   > **Note**: You can also 
   > [view the YAML in your browser](1.7/calico.yaml){:target="_blank"}.
   {: .alert .alert-info}

1. Continue to [Installing the CNX Manager](#installing-the-cnx-manager)

## Installing with an existing etcd datastore

To install {{site.prodname}}, configured to use an etcd that you have already set-up:

1. Ensure your cluster meets the {{site.prodname}} [system requirements](../../../requirements) (or recreate it if not).

2. Follow [the main etcd datastore instructions](../hosted). 

1. Continue to [Installing the CNX Manager](#installing-the-cnx-manager)

## Kubernetes API datastore

To install {{site.prodname}}, configured to use the Kubernetes API as its sole data source:

1. Ensure your cluster meets the {{site.prodname}} [system requirements](../../../requirements) (or recreate it if not).

2. Follow [the main Kubernetes datastore instructions](../kubernetes-datastore).

1. Continue to [Installing the CNX Manager](#installing-the-cnx-manager)

## Installing the CNX Manager

1. [Open cnx-etcd.yaml in a new tab](../cnx/1.7/cnx-etcd.yaml){:target="_blank"}.

1. Copy the contents, paste them into a new file, and save the file as cnx.yaml.
   This is what subsequent instructions will refer to.
   
{% include {{page.version}}/cnx-mgr-install.md %}

{% include {{page.version}}/cnx-monitor-install.md %}

{% include {{page.version}}/gs-next-steps.md %}
