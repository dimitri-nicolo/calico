---
title: Multi-cluster management overview
description: Simplify the management of multiple Calico Enterprise installations using a single management plane.
canonical_url: '/reference/mcm/overview'
---
## Big picture

Set up a single management plane to simplify administration of multiple {{site.prodname}} installations by connecting the individual clusters together through a centralized management cluster.

## Value

{{ site.prodname }} provides the ability to control multiple installations from a single, centralized instance of {{site.prodname}} Manager.

By configuring one of your {{site.prodname}} installation as the management cluster, you can connect it to one or more additional clusters. Each one of these additional clusters will be configured as a managed cluster. Once connected, you can toggle between clusters from the {{site.prodname}} Manager residing in the management cluster. This eliminates the overhead of having to access each cluster independently.

In addition, connecting multiple {{site.prodname}} clusters together means you only have to maintain a single Elasticsearch installation within the management cluster.

## Cluster types

| **Cluster types** | **Description**                                                                                                                                                                       |
| ----------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Management        | A cluster containing a shared management plane for managing all of your {{site.prodname}} clusters. Includes a centralized Elasticsearch for log storage across all of your clusters. |
| Managed           | A cluster that is managed by the shared management plane. Does not have its own Elasticsearch or {{site.prodname}} Manager.                                                           |
| Standalone        | A cluster that operates independently, is not connected to any other clusters, and has its own Elasticsearch and {{site.prodname}} Manager.                                           |

![Multi-cluster Overview]({{site.baseurl}}/images/mcm/mcm-overview.png)
{: .align-center}

## View multi-cluster management in {{site.prodname}} Manager

1. Log in to the {{site.prodname}} Manager. For more information on how to access the Manager, see [Configure access to the manager UI]({{site.baseurl}}/getting-started/cnx/access-the-manager).

1. In the right-hand side of the top navigation bar, you can toggle between the various clusters that are currently registered with the management plane. Clicking it displays a drop-down menu where you can pick the cluster you want to manage.

    ![Cluster Selection]({{site.baseurl}}/images/mcm/mcm-cluster-selection.png)
    {: .align-center}

    Selecting a cluster from the drop-down menu means all other views in the {{site.prodname}} Manager are specific to that cluster. For example, going to the Policies view displays Tiers and Policies defined within the selected cluster.
    
    By default, there is one cluster available in the selection drop down immediately after installation, the management cluster itself.

1. Examine the managed clusters view. This view allows you to see all currently registered managed clusters. From here you can edit each managed cluster and register additional managed clusters. Initially, this listing will be empty.

    ![Cluster Created]({{site.baseurl}}/images/mcm/mcm-cluster-created.png)
    {: .align-center}

> **Note**: The management cluster is not displayed in this view. It is not registered and cannot be edited or removed.
{: .alert .alert-info}

## Next steps

- [Multi-cluster management quickstart]({{site.baseurl}}/reference/mcm/quickstart)
