---
title: Installing Tigera Secure EE on Google GKE
redirect_from: latest/getting-started/kubernetes/installation/gke
canonical_url: https://docs.tigera.io/v2.3/getting-started/kubernetes/installation/other
---

## Overview

This guide covers installing {{site.prodname}} for policy enforcement on Google GKE.

## Before you begin

- Create a GKE cluster with the following settings:
 
  - [Intranode visibility](https://cloud.google.com/kubernetes-engine/docs/how-to/intranode-visibility) *enabled*.  This setting configures GKE to use the GKE CNI plugin, which is required.
  
  - Network policy *disabled*.  The Network Policy setting configures GKE to install Tigera Calico in a way that locks down its configuration, which prevents installation of {{site.prodname}}.
  
  - Istio *disabled*.  The Istio setting on the GKE cluster configures GKE to install Istio in a way that locks down its configuration, which prevents configuration of {{site.prodname}} application layer policy.  If you wish to use Istio in your cluster, follow [this GKE tutorial](https://cloud.google.com/kubernetes-engine/docs/tutorials/installing-istio), which explains how to install the open source version of Istio on GKE.

- Ensure that your Google account has sufficient IAM permissions.  To apply the {{site.prodname}} manifests requires permissions to create Kubernetes ClusterRoles and ClusterRoleBindings.  The easiest way to grant such permissions is to assign the "Kubernetes Engine Developer" IAM role to your user account as described in the [Creating Cloud IAM policies](https://cloud.google.com/kubernetes-engine/docs/how-to/iam) section of the GKE documentation.
  
  > **Tip**: By default, GCP users often have permissions to create basic Kubernetes resources (such as Pods and Services) but lack the permissions to create ClusterRoles and other admin resources.  Even if you can create basic resources, it's worth double checking that you can create admin resources before continuing.
  {: .alert .alert-success}

- Ensure your cluster has sufficient RAM to install {{site.prodname}}.  The datastore and indexing components pre-allocate significant resources.  At least 10GB is required.

- Ensure that you have the [credentials for the Tigera private registry](../../../getting-started/#obtain-the-private-registry-credentials)
  and a [license key](../../../getting-started/#obtain-a-license-key).
  
- To follow the TLS certificate and key creation instructions below you'll need openssl.

{% include {{page.version}}/load-docker.md yaml="calico" orchestrator="kubernetes" %}

{% include {{page.version}}/pull-secret.md %}

### <a name="install-cnx"></a><a name="install-ee-typha-nofed"></a>Installing {{site.prodname}} without federation

1. Download the {{site.prodname}} policy-only manifest for the Kubernetes API datastore with GKE CNI plugin support.

   ```bash
   curl \
   {{site.url}}/{{page.version}}/manifests/gke/calico-typha.yaml \
   -o calico.yaml
   ```

{% include {{page.version}}/cnx-cred-sed.md yaml="calico" %}

{% include {{page.version}}/config-typha.md autoscale="true" %}

1. Apply the manifest using the following command.

   ```bash
   kubectl apply -f calico.yaml
   ```

1. Continue to [Installing the {{site.prodname}} API Server](#installing-the-{{site.prodnamedash}}-api-server)

{% include {{page.version}}/cnx-api-install.md init="kubernetes" net="other" platform="gke" %}

1. Continue to [Applying your license key](#applying-your-license-key).

{% include {{page.version}}/apply-license.md platform="gke" cli="kubectl" %}

{% include {{page.version}}/cnx-monitor-install.md elasticsearch="operator" platform="gke" %}

1. Continue to [Installing the {{site.prodname}} Manager](#installing-the-{{site.prodnamedash}}-manager)

{% include {{page.version}}/cnx-mgr-install.md init="kubernetes" net="other" platform="gke" %}

{% include {{page.version}}/gs-next-steps.md %}
