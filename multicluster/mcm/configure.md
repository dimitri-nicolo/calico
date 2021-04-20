---
title: Configure Calico Enterprise for multi-cluster management
description: Configure Calico Enterprise to manage clusters from a single management plane.
canonical_url: '/multicluster/mcm/configure'
---

### Big picture

Install {{site.prodname}} multi-cluster management to manage clusters from a single management plane.

### Value

Managing standalone clusters and multiple instances of Elasticsearch is not onerous when you first install {{site.prodname}}. But as you move to production with 300+ clusters, it is not scalable; you need centralized cluster management and log storage. With {{site.prodname}} multi-cluster management, you can securely connect multiple clusters from different cloud providers in a single management plane, and control user access using RBAC. This architecture also supports federation of network policy resources across clusters, and lays the foundation for a “single pane of glass.” 

### Features

This how-to guide uses the following {{site.prodname}} features:

- **Installation API** with `ManagementCluster` resource
- **Installation API** with `ManagementClusterConnection` resource
- {{site.prodname}} Manager user interface

### Concepts

#### Cluster types

The standard {{site.prodname}} installation is a standalone cluster. For multi-cluster management, you install {{site.prodname}} on two types of clusters: a **management cluster**, and **managed clusters**. Note that you can do everything on a managed cluster that you can on a standalone cluster.

![Cluster Selection]({{site.baseurl}}/images/mcm/mcm-clusters.png)


| **Cluster types**           | **Description**                                              |
| ------------------ | ------------------------------------------------------------ |
| Management  | Provides a single management plane with a centralized Elasticsearch for managing multiple managed clusters. You should have a single management cluster connected to all of your managed clusters. |
| Managed  | A cluster managed by a centralized management plane with a shared Elasticsearch. Because a managed cluster sends log data to the central Elasticsearch, it is not fully operational until it is connected to the management plane. Access control to each managed cluster’s log data can be configured individually. |

After installation, you control your managed clusters in the {{site.prodname}} Manager UI. 

![Cluster Selection]({{site.baseurl}}/images/mcm/mcm-cluster-selection.png)

#### User authentication and authorization

Multi-cluster management provides a single source for authorization across managed clusters. The default authentication method for user access to the management cluster is [Token authentication]({{site.baseurl}}/getting-started/cnx/authentication-quickstart). You define user access to managed clusters using Kubernetes RBAC roles and cluster roles. For example, you can define access to specific log types (DNS, flow, audit) and specific clusters. 

### Before you begin...

**Supported**

- Kubernetes on-premises
- OpenShift
- GKE, EKS, and AKS

**Required**

- A [{{site.prodname}} license and pull secret]({{site.baseurl}}/getting-started/calico-enterprise)
- Two new Kubernetes clusters configured with `kubectl`.  
  For help, see [install Kubernetes]({{site.baseurl}}/getting-started/kubernetes/quickstart#install-kubernetes).
- A reachable, public IP address for the management cluster 

### How to

To get started with a non-production deployment, follow these steps to set up a management cluster, with a new managed cluster.

#### Set up the management cluster
Follow these steps in the cluster you intend to use as the management cluster.

{% tabs %}
Kubernetes, Openshift, GKE, EKS, AKS
<label:Kubernetes,active:true>
<%
{% include content/install-generic.md clusterType="management" %}

#### Set up a managed cluster
Follow these steps in the cluster you intend to use as the managed cluster.

{% include content/install-generic.md clusterType="managed" %}
%>

<label: GKE>
<%
{% include content/install-gke.md clusterType="management" %}

#### Set up a managed cluster
Follow these steps in the cluster you intend to use as the managed cluster.

{% include content/install-gke.md clusterType="managed" %}
%>

<label: EKS>
<%
{% include content/install-eks.md clusterType="management" %}

#### Set up a managed cluster
Follow these steps in the cluster you intend to use as the managed cluster.

{% include content/install-eks.md clusterType="managed" %}
%>

<label: AKS>
<%
{% include content/install-aks.md clusterType="management" %}

#### Set up a managed cluster
Follow these steps in the cluster you intend to use as the managed cluster.

{% include content/install-aks.md clusterType="managed" %}
%>

<label: Openshift>
<%
{% include content/install-openshift.md clusterType="management" %}

#### Set up a managed cluster
Follow these steps in the cluster you intend to use as the managed cluster.

{% include content/install-openshift.md clusterType="managed" %}
%>

{% endtabs %}


If you now access the Manager UI, you should see your managed cluster as an option in the cluster selection drop-down (top right banner). It will have the same name you provided when adding the managed cluster in the UI. Once you select your managed cluster, you will be able to access all of the Manager UI features while connected to that cluster (e.g. Policies, Flow Visualizations, etc).


You have now successfully completed the setup for multi-cluster management.

### Next steps

- When you are ready to fine-tune your multi-cluster management deployment for production, see [Fine-tune multi-cluster management]({{site.baseurl}}/multicluster/mcm/fine-tune-deployment)
- To change an existing {{site.prodname}} standalone cluster to a management or managed cluster, see [Change cluster types]({{site.baseurl}}/multicluster/mcm/change-cluster-type)
