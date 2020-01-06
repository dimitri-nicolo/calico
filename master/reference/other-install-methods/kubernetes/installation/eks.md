---
title: Installing Calico Enterprise on Amazon EKS
canonical_url: https://docs.tigera.io/v2.3/getting-started/kubernetes/installation/other
---

## Overview

This guide covers installing {{site.prodname}} for policy enforcement on Amazon EKS.

## Before you begin

- Ensure that you have an EKS cluster with [platform version](https://docs.aws.amazon.com/eks/latest/userguide/platform-versions.html)
  at least eks.2 (for aggregated API server support).

- Ensure that your EKS nodes meet [system node requirements.](/{{page.version}}/getting-started/kubernetes/requirements)

- Ensure that you have the [credentials for the Tigera private registry](/{{page.version}}/getting-started/#obtain-the-private-registry-credentials)
  and a [license key](/{{page.version}}/getting-started/#obtain-a-license-key).

- To follow the TLS certificate and key creation instructions below you'll need openssl.

{% include {{page.version}}/load-docker.md yaml="calico" orchestrator="kubernetes" %}

{% include {{page.version}}/pull-secret.md %}

## <a name="install-cnx"></a>Installing {{site.prodname}} for policy only

> **Important**: At this time, we include steps for Kubernetes API datastore only. Should you wish
> to install {{site.prodname}} for policy only using the etcd datastore type, contact Tigera support.
{: .alert .alert-danger}

### <a name="install-ee-typha-nofed"></a>Installing {{site.prodname}} without federation

1. Download the {{site.prodname}} policy-only manifest for the Kubernetes API datastore with AWS VPC CNI plugin.

   ```bash
   curl \
   {{site.url}}/{{page.version}}/getting-started/kubernetes/installation/hosted/kubernetes-datastore/policy-only-ecs/1.7/calico-typha.yaml \
   -o calico.yaml
   ```

{% include {{page.version}}/cnx-cred-sed.md yaml="calico" %}

{% include {{page.version}}/config-typha.md %}

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
