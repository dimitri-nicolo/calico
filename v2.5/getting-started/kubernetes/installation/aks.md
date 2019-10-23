---
title: Installing Tigera Secure EE on Azure AKS
redirect_from: latest/getting-started/kubernetes/installation/aks
canonical_url: https://docs.tigera.io/v2.5/getting-started/kubernetes/installation/aks
---

## Overview

This guide covers installing {{site.prodname}} for policy enforcement on Azure AKS.

> **Note**: AKS support on {{site.prodname}} is currently in technical preview
   and is suitable for non-production clusters only.
{: .alert .alert-info}

## Before you begin

- Create an AKS cluster with the following settings:

  - *Azure CNI plugin in transparent mode*. {{site.prodname}} requires Azure CNI plugin to be operating in transparent mode.

  - *No network policy*. This is to avoid conflicts between other network policy providers in the cluster and {{site.prodname}}.

- Ensure that your Azure account has sufficient IAM permissions. To apply the {{site.prodname}} manifests requires permissions to create Kubernetes `ClusterRoles` and `ClusterRoleBindings`. The easiest way to grant such permissions is to assign the "Azure Kubernetes Service Cluster Admin Role" to your user account. Please refer to [AKS access control](https://docs.microsoft.com/bs-latn-ba/azure/aks/control-kubeconfig-access).

- Ensure your cluster has sufficient RAM to install {{site.prodname}}.  The datastore and indexing components pre-allocate significant resources.  At least 10GB is required.

- Ensure that you have the [credentials for the Tigera private registry](../../../getting-started/#obtain-the-private-registry-credentials)
  and a [license key](../../../getting-started/#obtain-a-license-key).

- To follow the TLS certificate and key creation instructions below you'll need openssl.

{% include {{page.version}}/aks-work-around.md %}

{% include {{page.version}}/load-docker.md yaml="calico" orchestrator="kubernetes" %}

{% include {{page.version}}/pull-secret.md %}

### <a name="install-cnx"></a><a name="install-ee-typha-nofed"></a>Installing {{site.prodname}} without federation

1. Download the {{site.prodname}} policy-only manifest for the Kubernetes API datastore with Azure CNI plugin support.

   ```bash
   curl \
   {{site.url}}/{{page.version}}/manifests/aks/calico-typha.yaml \
   -o calico.yaml
   ```

{% include {{page.version}}/cnx-cred-sed.md yaml="calico" %}

{% include {{page.version}}/config-typha.md autoscale="true" %}

1. Apply the manifest using the following command.

   ```bash
   kubectl apply -f calico.yaml
   ```

1. Continue to [Installing the {{site.prodname}} API Server](#installing-the-{{site.prodnamedash}}-api-server)

{% include {{page.version}}/cnx-api-install.md init="kubernetes" net="other" platform="aks" %}

1. Continue to [Applying your license key](#applying-your-license-key).

{% include {{page.version}}/apply-license.md platform="aks" cli="kubectl" %}

{% include {{page.version}}/cnx-monitor-install.md elasticsearch="operator" platform="aks" %}

1. Continue to [Installing the {{site.prodname}} Manager](#installing-the-{{site.prodnamedash}}-manager)

{% include {{page.version}}/cnx-mgr-install.md init="kubernetes" net="other" platform="aks" %}

{% include {{page.version}}/gs-next-steps.md %}
