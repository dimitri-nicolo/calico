---
title: Installing the Federated Services Controller
---

## Before you begin

It is assumed you have a Kubernetes cluster with a {{site.prodname}} installation that is suitable for Federated Endpoint
Identity. In addition, you should have the {{site.prodname}} license applied and the {{site.prodname}} Docker images 
available.
 
Refer to the [Installing {{site.prodname}} on Kubernetes](./index) guide for more details on installation of {{site.prodname}}.

It is only necessary to install the Federated Services Controller if you intend to use [Federated Services](/{{page.version}}/usage/federation/services-controller)
as your cross-cluster service discovery mechanism.

## Installing the Federated Services Controller

To install the controller, download, edit and apply the manifest appropriate to the datastore type selected for your
{{site.prodname}} installation.

1. Download the controller manifest

   To download the {{site.prodname}} Federated Services Controller manifest for etcd datastore:
   
   ```bash
   curl \
   {{site.url}}/{{page.version}}/getting-started/kubernetes/installation/federation-controller.yaml \
   -O
   ```

   To download the {{site.prodname}} Federated Services Controller manifest for the Kubernetes API datastore:
   
   ```bash
   curl \
   {{site.url}}/{{page.version}}/getting-started/kubernetes/installation/hosted/kubernetes-datastore/federation-controller.yaml \
   -O
   ```

1. Edit the manifest to volume mount any required secrets

   If you have already configured some Remote Cluster Configuration resources, and have created some secrets to store the
   remote cluster keys etc., then you will need to volume mount the secrets into the controller container.

   If you have not yet created any secrets, the  manifest can be applied unchanged. Without configuring any Remote Cluster 
   Configuration resources, the Federated Services Controller is able to federate services within your local cluster.
   This may be sufficient for some initial testing to familiarize yourself with the controller.

   The process of adding Remote Cluster Configuration resources and mounting secrets is discussed in more details in the 
   [Configuring a Remote Cluster for Federation](/{{page.version}}/usage/federation/configure-rcc) guide.

   > **Warning**: If you are upgrading from a previous release and previously had secrets mounted in for 
   > federation, then failure to include these secrets in this manifest will result in loss of federation
   > functionality, which may include loss of service between clusters.
   {: .alert .alert-danger}

1. Apply the manifest

   ```bash
   kubectl apply -f federation-controller.yaml
   ```

## More information and next steps

Refer to the following guides for more details on configuration options for Federation:
- [Configuring a Remote Cluster for Federation](/{{page.version}}/usage/federation/configure-rcc)
- [Configuring a federated service](/{{page.version}}/usage/federation/services-controller)
