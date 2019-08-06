---
title: Installing Tigera Secure EE on Amazon EKS
canonical_url: https://docs.tigera.io/v2.3/getting-started/kubernetes/installation/other
---

## Overview

This guide covers installing {{site.prodname}} for policy enforcement on Amazon EKS.

## Before you begin

- Ensure that you have an EKS cluster with [platform version](https://docs.aws.amazon.com/eks/latest/userguide/platform-versions.html)
  at least eks.2 (for aggregated API server support).

- Ensure that you have the [credentials for the Tigera private registry](../../../getting-started/#obtain-the-private-registry-credentials)
  and a [license key](../../../getting-started/#obtain-a-license-key).

{% include {{page.version}}/load-docker.md yaml="calico" orchestrator="kubernetes" %}

{% include {{page.version}}/pull-secret.md %}

## <a name="install-cnx"></a>Installing {{site.prodname}} for policy only

> **Important**: At this time, we include steps for Kubernetes API datastore only. Should you wish
> to install {{site.prodname}} for policy only using the etcd datastore type, contact Tigera support.
{: .alert .alert-danger}

### <a name="install-ee-typha-nofed"></a>Installing {{site.prodname}} for policy only without federation, more than 50 nodes

1. Download the {{site.prodname}} policy-only manifest for the Kubernetes API datastore with AWS VPC CNI plugin.

   ```bash
   curl \
   {{site.url}}/{{page.version}}/getting-started/kubernetes/installation/hosted/kubernetes-datastore/policy-only-ecs/1.7/calico-typha.yaml \
   -o calico.yaml
   ```

{% include {{page.version}}/cnx-cred-sed.md yaml="calico" %}

1. Open the manifest in your favorite editor and modify the replica count in the
   `Deployment` named `calico-typha` to the desired number of replicas.

   ```yaml
   apiVersion: apps/v1beta1
   kind: Deployment
   metadata:
     name: calico-typha
     ...
   spec:
     ...
     replicas: <number of replicas>
   ```
   {: .no-select-button}

   We recommend at least one replica for every 200 nodes and no more than
   20 replicas. In production, we recommend a minimum of three replicas to reduce
   the impact of rolling upgrades and failures.

   > **Warning**: If you do not increase the replica
   > count from its default of `0` Felix will try to connect to Typha, find no
   > Typha instances to connect to, and fail to start.
   {: .alert .alert-danger}

1. Apply the manifest using the following command.

   ```bash
   kubectl apply -f calico.yaml
   ```

1. Continue to [Installing the {{site.prodname}} API Server](#installing-the-{{site.prodnamedash}}-api-server)

{% include {{page.version}}/cnx-api-install.md init="kubernetes" net="other" platform="eks" %}

1. Continue to [Applying your license key](#applying-your-license-key).

{% include {{page.version}}/apply-license.md platform="eks" cli="kubectl" %}

{% include {{page.version}}/cnx-monitor-install.md elasticsearch="operator" platform="eks" %}

1. Continue to [Installing the {{site.prodname}} Manager](#installing-the-{{site.prodnamedash}}-manager)

{% include {{page.version}}/cnx-mgr-install.md init="kubernetes" net="other" platform="eks" %}

{% include {{page.version}}/gs-next-steps.md %}
