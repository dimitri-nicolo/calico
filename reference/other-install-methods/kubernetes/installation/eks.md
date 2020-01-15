---
title: Installing Calico Enterprise on Amazon EKS
canonical_url: /getting-started/kubernetes/installation/other
---

## Overview

This guide covers installing {{site.prodname}} for policy enforcement on Amazon EKS.

## Before you begin

- Ensure that you have an EKS cluster with [platform version](https://docs.aws.amazon.com/eks/latest/userguide/platform-versions.html)
  at least eks.2 (for aggregated API server support).

- Ensure that your EKS nodes meet [system node requirements.]({{site.baseurl}}/getting-started/kubernetes/requirements)

- Ensure that you have the [credentials for the Tigera private registry]({{site.baseurl}}/getting-started/calico-enterprise#obtain-the-private-registry-credentials)
  and a [license key]({{site.baseurl}}/getting-started/calico-enterprise#obtain-a-license-key).

- To follow the TLS certificate and key creation instructions below you'll need openssl.

{% include content/load-docker.md yaml="calico" orchestrator="kubernetes" %}

{% include content/pull-secret.md %}

## <a name="install-cnx"></a>Installing {{site.prodname}} for policy only

> **Important**: At this time, we include steps for Kubernetes API datastore only. Should you wish
> to install {{site.prodname}} for policy only using the etcd datastore type, contact Tigera support.
{: .alert .alert-danger}

### <a name="install-ee-typha-nofed"></a>Installing {{site.prodname}} without federation

1. Download the {{site.prodname}} policy-only manifest for the Kubernetes API datastore with AWS VPC CNI plugin.

   ```bash
   curl \
   {{ "/getting-started/kubernetes/installation/hosted/kubernetes-datastore/policy-only-ecs/1.7/calico-typha.yaml" | absolute_url }} \
   -o calico.yaml
   ```

{% include content/cnx-cred-sed.md yaml="calico" %}

{% include content/config-typha.md %}

1. Apply the manifest using the following command.

   ```bash
   kubectl apply -f calico.yaml
   ```

1. Continue to [Installing the {{site.prodname}} API Server](#installing-the-{{site.prodnamedash}}-api-server)

{% include content/cnx-api-install.md init="kubernetes" net="other" platform="eks" %}

1. Continue to [Applying your license key](#applying-your-license-key).

{% include content/apply-license.md platform="eks" cli="kubectl" %}

{% include content/cnx-monitor-install.md elasticsearch="operator" platform="eks" %}

1. Continue to [Installing the {{site.prodname}} Manager](#installing-the-{{site.prodnamedash}}-manager)

{% include content/cnx-mgr-install.md init="kubernetes" net="other" platform="eks" %}

{% include content/gs-next-steps.md %}
