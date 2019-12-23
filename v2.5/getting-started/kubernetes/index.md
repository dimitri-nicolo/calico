---
title: Quickstart for Tigera Secure EE on Kubernetes
canonical_url: https://docs.tigera.io/v2.3/getting-started/kubernetes/
---

## Overview

This quickstart gets you a single-host Kubernetes cluster with {{site.tseeprodname}}
in approximately 15 minutes. You can use this cluster for testing and
development.

To deploy a cluster suitable for production, refer to [Installation](installation).

## Create a Kubernetes cluster

### Host requirements

- AMD64 processor
- 2CPU
- 12GB RAM
- 50GB free disk space
- Ubuntu Server 16.04
- Internet access
- [Sufficient virtual memory](https://www.elastic.co/guide/en/elasticsearch/reference/current/vm-max-map-count.html){:target="_blank"}

### Create a single-host Kubernetes cluster

1. [Follow the Kubernetes instructions to install kubeadm](https://kubernetes.io/docs/setup/independent/install-kubeadm/){:target="_blank"}.

1. As a regular user with sudo privileges, open a terminal on the host that
   you installed kubeadm on.

1. Initialize the master using the following command.

   ```bash
   sudo kubeadm init --pod-network-cidr=192.168.0.0/16 \
   --apiserver-cert-extra-sans=127.0.0.1
   ```

   > **Note**: If 192.168.0.0/16 is already in use within your network you must select a different pod network
   > CIDR, replacing 192.168.0.0/16 in the above command as well as in any manifests applied below.
   {: .alert .alert-info}

1. Execute the following commands to configure kubectl (also returned by
   `kubeadm init`).

   ```bash
   mkdir -p $HOME/.kube
   sudo cp -i /etc/kubernetes/admin.conf $HOME/.kube/config
   sudo chown $(id -u):$(id -g) $HOME/.kube/config
   ```

1. Remove master taint in order to allow kubernetes to schedule pods on the master node.

   ```bash
   kubectl taint nodes --all node-role.kubernetes.io/master-
   ```

## Install helm and tiller

- [Install the helm](https://helm.sh/docs/using_helm/#install-helm) binary. For
  example:

  ```bash
  curl -O https://get.helm.sh/helm-v2.14.3-linux-amd64.tar.gz
  tar xzvf helm-v2.14.3-linux-amd64.tar.gz
  sudo install linux-amd64/helm /usr/bin/
  ```

- Verify `helm` is installed:

  ```bash
  helm version
  ```

- Install tiller.

  ```bash
  kubectl create clusterrolebinding tiller-cluster-rule --clusterrole=cluster-admin --serviceaccount=kube-system:tiller
  kubectl create serviceaccount --namespace kube-system tiller
  helm init --net-host --service-account tiller \
    --override "spec.template.spec.tolerations[0].effect=NoSchedule" \
    --override "spec.template.spec.tolerations[0].key=node.kubernetes.io/not-ready" \
    --override "spec.template.spec.tolerations[0].operator=Exists"
  ```

>**Alert**: This creates an insecure installion of tiller and should not be used
           for production networks.
{: .alert .alert-info}

{% include {{page.version}}/helm-install.md method="quick" %}

### Next steps

**[Experiment with OIDC authentication strategy](/{{page.version}}/reference/cnx/authentication)**

**[Experiment with non-admin users and the {{site.tseeprodname}} manager](/{{page.version}}/reference/cnx/rbac-tiered-policies)**

**[Enable audit logs for Kubernetes `NetworkPolicy` and send them to Elasticsearch](/{{page.version}}/security/logs/elastic/ee-audit#enabling-auditing-for-other-resources)**

**[Secure a simple application using the Kubernetes `NetworkPolicy` API]({{site.url}}/{{page.version}}/security/simple-policy)**

**[Control ingress and egress traffic using the Kubernetes `NetworkPolicy` API]({{site.url}}/{{page.version}}/security/advanced-policy)**

**[Create a user interface that shows blocked and allowed connections in real time]({{site.url}}/{{page.version}}/security/stars-policy/)**

**[Install and configure calicoctl](/{{page.version}}/getting-started/calicoctl/install)**
