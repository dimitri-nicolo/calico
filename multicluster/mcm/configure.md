---
title: Configure Calico Enterprise for multi-cluster management, Kubernetes
description: Configure Calico Enterprise to manage clusters from a single management plane for Kubernetes.
canonical_url: '/multicluster/mcm/configure'
---

### Big picture

Install {{site.prodname}} multi-cluster management to manage clusters from a single management plane for Kubernetes.

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

**Supported cloud providers**

  GCP, GKE, AWS, EKS, AKS, RKE, and Azure

**Required**

- A [{{site.prodname}} license and pull secret]({{site.baseurl}}/getting-started/calico-enterprise)
- Two new Kubernetes clusters configured with `kubectl`.  
  For help, see [install Kubernetes]({{site.baseurl}}/getting-started/kubernetes/quickstart#install-kubernetes).
- A reachable, public IP address for the management cluster 

### How to

The following steps install multi-cluster management on two new Kubernetes clusters (a management cluster and a managed cluster). This is the simplest way to get started using multi-cluster management for non-production. As you move to production, you can exchange existing standalone {{site.prodname}} clusters to be the management cluster or the managed cluster, using [Change cluster types]({{site.baseurl}}/multicluster/mcm/change-cluster-type). 

- [Install Calico Enterprise](#install-calico-enterprise)
- [Turn a cluster into a management cluster](#turn-the-cluster-into-a-management-cluster)
- [Create an admin user and verify management cluster connection](#create-an-admin-user-and-verify-management-cluster-connection)
- [Add a managed cluster to the management cluster](#add-a-managed-cluster-to-the-management-cluster)
- [Install Calico Enterprise](#install-calico-enterprise-1)
- [Turn a cluster into a managed cluster](#turn-the-cluster-into-a-managed-cluster)
- [Provide permissions to view the managed cluster](#provide-permissions-to-view-the-managed-cluster)

#### Install Calico Enterprise
Follow these steps in the cluster you intend to use as the management cluster.

1. [Configure storage for {{site.prodname}}]({{site.baseurl}}/getting-started/create-storage).

1. Install the Tigera operator and custom resource definitions.

   ```bash
   kubectl create -f {{ "/manifests/tigera-operator.yaml" | absolute_url }}
   ```

1. Install the Prometheus operator and related custom resource definitions. The Prometheus operator will be used to deploy Prometheus server and Alertmanager to monitor {{site.prodname}} metrics.

   > **Note**: If you have an existing Prometheus operator in your cluster that you want to use, skip this step. To work with {{site.prodname}}, your Prometheus operator must be v0.30.0 or higher.
   {: .alert .alert-info}

   ```
   kubectl create -f {{ "/manifests/tigera-prometheus-operator.yaml" | absolute_url }}
   ```

1. Install your pull secret.

   ```bash
   kubectl create secret generic tigera-pull-secret \
       --from-file=.dockerconfigjson=<path/to/pull/secret> \
       --type=kubernetes.io/dockerconfigjson -n tigera-operator
   ```

1. Install the Tigera custom resources. For more information on configuration options available in this manifest, see [the installation reference]({{site.baseurl}}/reference/installation/api).

   ```bash
   kubectl create -f {{ "/manifests/custom-resources.yaml" | absolute_url }}
   ```

   You can now monitor progress with the following command:

   ```bash
   watch kubectl get tigerastatus
   ```

   Wait until the `apiserver` shows a status of `Available`, then proceed to the next section.

1. Install the Tigera license.

   ```bash
   kubectl create -f </path/to/license.yaml>
   ```

   Monitor progress with the following command:

   ```bash
   watch kubectl get tigerastatus
   ```

   When all components show a status of `Available`, proceed to the next section.

1. Secure {{site.prodname}} components on the management cluster with network policy

   ```bash
   kubectl create -f {{ "/manifests/tigera-policies.yaml" | absolute_url }}
   ```

#### Turn the cluster into a management cluster

To control managed clusters from your central management plane, you must ensure it is reachable for connections. The simplest way to get started (but not for production scenarios), is to configure a `NodePort` service to expose the management cluster. Note that the service must live within the `tigera-manager` namespace.

1.  Create a service to expose the management cluster. 
    The following example of a NodePort service may not be suitable for production and high availability. For options, see [Fine-tune multi-cluster management for production]({{site.baseurl}}/multicluster/mcm/fine-tune-deployment).
    Apply the following service manifest.
 
    ```bash
    kubectl create -f - <<EOF
    apiVersion: v1
    kind: Service
    metadata:
      name: tigera-manager-mcm
      namespace: tigera-manager
    spec:
      ports:
      - nodePort: 30449
        port: 9449
        protocol: TCP
        targetPort: 9449
      selector:
        k8s-app: tigera-manager
      type: NodePort
    EOF
    ```
 
1. Export the service port number, and the public IP or host of the management cluster. (Ex. "example.com:1234" or "10.0.0.10:1234".)
   ```bash
   export MANAGEMENT_CLUSTER_ADDR=<your-management-cluster-addr>
   ```
1. Apply the [ManagementCluster]({{site.baseurl}}/reference/installation/api#operator.tigera.io/v1.ManagementCluster) CR.

   ```bash
   kubectl apply -f - <<EOF
   apiVersion: operator.tigera.io/v1
   kind: ManagementCluster
   metadata:
     name: tigera-secure
   spec:
     address: $MANAGEMENT_CLUSTER_ADDR
   EOF
   ```

#### Create an admin user and verify management cluster connection

To access resources in a managed cluster from the {{site.prodname}} Manager within the management cluster, the logged-in user must have appropriate permissions defined in that managed cluster (clusterrole bindings). 

1. Create an admin user called, `mcm-user` in the default namespace with full permissions, by applying the following commands.

   ```bash
   kubectl create sa mcm-user
   kubectl create clusterrolebinding mcm-user-admin --serviceaccount=default:mcm-user --clusterrole=tigera-network-admin
   ```
1. Get the login token for your new admin user, and log in to {{site.prodname}} Manager.

   ```bash
   {% raw %}kubectl get secret $(kubectl get serviceaccount mcm-user -o jsonpath='{range .secrets[*]}{.name}{"\n"}{end}' | grep token) -o go-template='{{.data.token | base64decode}}' && echo{% endraw %}
   ```
   In the top right banner, your management cluster is displayed as the first entry in the cluster selection drop-down menu with the fixed name, `management cluster`.

   ![Cluster Created]({{site.baseurl}}/images/mcm/mcm-management-cluster.png)

You have successfully installed a management cluster.

#### Add a managed cluster to the management cluster

Choose a name for your managed cluster and then add it to your **management cluster**. The following commands will
create a manifest with the name of your managed cluster in your current directory.

1. First decide on the name for your managed cluster. We will re-use this value a few times throughout this page.
   ```bash
   export MANAGED_CLUSTER=my-managed-cluster
   ```

1. Add a managed cluster and save the manifest containing a [ManagementClusterConnection]({{site.baseurl}}/reference/installation/api#operator.tigera.io/v1.ManagementClusterConnection) and a Secret.
   ```bash
   kubectl -o jsonpath="{.spec.installationManifest}" > $MANAGED_CLUSTER.yaml create -f - <<EOF
   apiVersion: projectcalico.org/v3
   kind: ManagedCluster
   metadata:
     name: $MANAGED_CLUSTER
   EOF
   ```

Verify that the `managementClusterAddr` in the manifest is correct.

> **Tip**: Managed clusters can also be added from {{site.prodname}} Manager. From here you can see the connection status and switch to see data from other clusters by using the drop-down menu in the top right banner.
{: .alert .alert-info}

#### Install Calico Enterprise
Follow these steps in the cluster you intend to use as the managed cluster.

1. Install the Tigera operator and custom resource definitions.

   ```bash
   kubectl create -f {{ "/manifests/tigera-operator.yaml" | absolute_url }}
   ```

1. Install the Prometheus operator and related custom resource definitions. The Prometheus operator will be used to deploy Prometheus server and Alertmanager to monitor {{site.prodname}} metrics.

   > **Note**: If you have an existing Prometheus operator in your cluster that you want to use, skip this step. To work with {{site.prodname}}, your Prometheus operator must be v0.30.0 or higher.
   {: .alert .alert-info}

   ```
   kubectl create -f {{ "/manifests/tigera-prometheus-operator.yaml" | absolute_url }}
   ```

1. Install your pull secret.

   ```bash
   kubectl create secret generic tigera-pull-secret \
       --from-file=.dockerconfigjson=<path/to/pull/secret> \
       --type=kubernetes.io/dockerconfigjson -n tigera-operator
   ```

1. Download the Tigera custom resources. For more information on configuration options available in this manifest, see [the installation reference]({{site.baseurl}}/reference/installation/api).

   ```bash
   curl -O -L {{ "/manifests/custom-resources.yaml" | absolute_url }}
   ```

   Remove the `Manager` custom resource from the manifest file.

   ```yaml
   apiVersion: operator.tigera.io/v1
   kind: Manager
   metadata:
     name: tigera-secure
   spec:
     # Authentication configuration for accessing the Tigera manager.
     # Default is to use token-based authentication.
     auth:
       type: Token
   ```

   Remove the `LogStorage` custom resource from the manifest file.

   ```yaml
   apiVersion: operator.tigera.io/v1
   kind: LogStorage
   metadata:
     name: tigera-secure
   spec:
     nodes:
       count: 1
   ```
   Now apply the modified manifest.

   ```bash
   kubectl create -f ./custom-resources.yaml
   ```
1. Monitor progress with the following command:
   ```bash
   watch kubectl get tigerastatus
   ```
   Wait until the `apiserver` shows a status of `Available`, then go to the next step.

#### Turn the cluster into a managed cluster
1. Apply the manifest that you modified in the step, [Add a managed cluster to the management cluster](#add-a-managed-cluster-to-the-management-cluster).
   ```bash
   kubectl apply -f $MANAGED_CLUSTER.yaml
   ```
1. Monitor progress with the following command:
   ```bash
   watch kubectl get tigerastatus
   ```
   Wait until the `management-cluster-connection` and `tigera-compliance` show a status of `Available`.

1. Secure {{site.prodname}} on the managed cluster with network policy.

   ```bash
   kubectl create -f {{ "/manifests/tigera-policies-managed.yaml" | absolute_url }}
   ```

You have now successfully installed a managed cluster!


#### Provide permissions to view the managed cluster

To access resources belonging to a managed cluster from the {{site.prodname}} Manager UI, the service or user account used to log in must have appropriate permissions defined in the managed cluster.

Let's define admin-level permissions for the service account (`mcm-user`) we created to log in to the Manager UI. Run the following command against your managed cluster.

```bash
kubectl create clusterrolebinding mcm-user-admin --serviceaccount=default:mcm-user --clusterrole=tigera-network-admin
```

If you now access the Manager UI, you should see your managed cluster as an option in the cluster selection drop-down (top right banner). It will have the same name you inputted when adding the managed cluster in the UI. Once you select your managed cluster, you will be able to access all of the Manager UI features while connected to that cluster (e.g. Policies, Flow Visualizations, etc).

You have now successfully completed the setup for multi-cluster management.

### Next steps

- When you are ready to fine-tune your multi-cluster management deployment for production, see [Fine-tune multi-cluster management]({{site.baseurl}}/multicluster/mcm/fine-tune-deployment)
- To change an existing {{site.prodname}} standalone cluster to a management or managed cluster, see [Change cluster types]({{site.baseurl}}/multicluster/mcm/change-cluster-type)
