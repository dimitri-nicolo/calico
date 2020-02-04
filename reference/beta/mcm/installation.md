---
title: Set up multi-cluster management
description: Simplify the management of multiple Calico Enterprise installations using a single management plane.
---
>**Note**: Beta version
{: .alert .alert-info}

### Big picture

Set up a single management plane to simplify administration of multiple {{site.prodname}} installations by connecting the individual clusters together through a centralized management cluster.

### Value

{{ site.prodname }} provides the ability to control multiple {{site.prodname}} installations from a single, centralized instance of {{site.prodname}} Manager.

By configuring one of your Calio Enterprise installation as the management cluster, you can connect it to one or more additional clusters. Each one of these additional clusters will be configured as a managed cluster. Once connected, you can toggle between clusters from the {{site.prodname}} Manager residing in the management cluster. This eliminates the overhead of having to access each cluster independently.

In addition, connecting multiple {{site.prodname}} clusters together means you only have to maintain a single Elasticsearch installation within the management cluster.

### Features

This how-to guide uses the following {{site.prodname}} features:

- Installation API with ManagementClusterConnection
- Managed clusters user flow within the {{site.prodname}} Manager
- Kubernetes RBAC to individual permissions per cluster
- Querying flow logs across mulitple clusters within Kibana

### Concepts

#### Cluster types

| **Cluster types** | **Description**                                                                                                                                                                       |
| ----------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Management        | A cluster containing a shared management plane for managing all of your {{site.prodname}} clusters. Includes a centralized Elasticsearch for log storage across all of your clusters. |
| Managed           | A cluster that is managed by the shared management plane. Does not have its own Elasticsearch or {{site.prodname}} Manager.                                                           |
| Standalone        | A cluster that operates independently, is not connected to any other clusters, and has its own Elasticsearch and {{site.prodname}} Manager. All clusters prior to 2.7 are standalone. |

![Multi-cluster Overview]({{site.url}}/images/mcm/mcm-overview.png)

### Before you begin...

You will need multiple Kubernetes clusters to install {{site.prodname}} with different cluster types.

#### Requirements

- Each cluster has Kubernetes is up and running. For help, see [Install Kubernetes in the {{site.prodname}} Quickstart guide]({{site.baseurl}}/getting-started/kubernetes/quickstart#install-kubernetes).
- `kubectl` is installed on each cluster 
- A {{site.prodname}} license and docker registry config file

#### Limitations

Intrusion detection is currently not supported in a multi-cluster configuration.

### How To

#### Configure multi-cluster management using new clusters

**Install management cluster**

- [Install {{site.prodname}} on the management cluster](#install-siteprodname-on-the-management-cluster)
- [Install the {{site.prodname}} license on the management cluster](#install-the-siteprodname-license-on-the-management-cluster)
- [Secure {{site.prodname}} on the management cluster with network policy](#secure-siteprodname-on-the-management-cluster-with-network-policy)
- [Allow connections to the management plane](#allow-connections-to-the-management-plane)
- [View multi-cluster management in {{site.prodname}} Manager](#view-multi-cluster-management-in-siteprodname-manager)

**Install a managed cluster**

- [Add a managed cluster to the management plane](#add-a-managed-cluster-to-the-management-plane)
- [Install {{site.prodname}} on the managed cluster](#install-siteprodname-on-the-managed-cluster)
- [Install a {{site.prodname}} license on the managed cluster](#install-a-siteprodname-license-on-the-managed-cluster)
- [Secure {{site.prodname}} on the managed cluster with network policy](#secure-siteprodname-on-the-managed-cluster-with-network-policy)

**Post-installation**

- [Configure cross-cluster user permissions]()
- [Access log data in Kibana for a specific managed cluster]()
- [Configure permissions to access log data per managed cluster]()

#### Configure multi-cluster management using existing standalone clusters

- Convert a standalone cluster to a management cluster
- Add existing clusters in the management cluster
- Convert a standalone cluster to a managed cluster

##### Install a management cluster

The first step is to set up a management cluster. This will contain your centralized {{site.prodname}} Manager and Elasticsearch installation.

###### Install {{site.prodname}} on the management cluster

1. [Configure a storage class for {{site.prodname}}]({{site.baseurl}}/getting-started/create-storage).

1. Install the Tigera operators and custom resource definitions.

   ```
   kubectl create -f {{ "/manifests/tigera-operator.yaml" | absolute_url }}
   ```

1. Install your pull secret.

   ```
   kubectl create secret generic tigera-pull-secret \
       --from-file=.dockerconfigjson=<path/to/pull/secret> \
       --type=kubernetes.io/dockerconfigjson -n tigera-operator
   ```

1. Install the Tigera custom resources. For more information on configuration options available in this manifest, see [the installation reference]({{site.baseurl}}/reference/installation/api).

   Download the custom resources YAML to your local directory.
   ```
   curl -O -L {{ "/manifests/custom-resources.yaml" | absolute_url }}
   ```

   Update the file by changing the `clusterManagementType` field for the Installation custom resource from `Standalone` to `Management`.

   Now, install the modified manifest.
   ```
   kubectl create -f ./custom-resources.yaml
   ```

   You can now monitor progress with the following command:

   ```
   watch kubectl get tigerastatus
   ```

   Wait until the `apiserver` shows a status of `Available`, then proceed to the next section.

###### Install the {{site.prodname}} license on the management cluster

In order to use {{site.prodname}}, you must install the license provided to you by Tigera.

```
kubectl create -f </path/to/license.yaml>
```

You can now monitor progress with the following command:

```
watch kubectl get tigerastatus
```

When all components show a status of `Available`, proceed to the next section.

###### Secure {{site.prodname}} on the management cluster with network policy

To secure {{site.prodname}} component communications, install the following set of network policies.

```
kubectl create -f {{ "/manifests/tigera-policies.yaml" | absolute_url }}
```

###### Allow connections to the management plane

In order for managed clusters to connect to the shared management plane, you must first allow external clusters to reach the Manager pod within the management cluster. This can be accomplished by creating a Kubernetes service for the Manager pod.

Follow the [Kubernetes docs for setting up a service](https://kubernetes.io/docs/concepts/services-networking/service/#defining-a-service).

Regardless of the service type you choose, you must ensure it obeys the following requirements:

- The service uses protocol `TCP`
- The service maps to port `9449` on the Manager pod
- The service exists within the `tigera-manager` namespace
- The service uses label selector `k8s-app: tigera-manager`

###### View multi-cluster management in {{site.prodname}} Manager

Let’s take a quick look at viewing the management cluster and managed clusters in {{site.prodname}} Manager.

1. Log in to the {{site.prodname}} Manager. For more information on how to access the Manager, see [Configure access to the manager UI]({{site.baseurl}}/getting-started/access-the-manager).

2. In the right-hand side of the top navigation bar, you can toggle between the various clusters that are currently registered with the management plane. Clicking it displays a drop-down menu where you can pick the cluster you want to manage.

   Selecting a cluster from the drop-down menu means all other views in the {{site.prodname}} Manager are specific to that cluster. For example, going to the Policies view displays Tiers and Policies defined within the selected cluster.
   
   By default, there is one cluster available in the selection drop down immediately after installation, the management cluster itself.

3. Examine the managed clusters view. This view allows you to see all currently registered managed clusters. From here you can edit each managed cluster and register additional managed clusters. Initially, this listing will be empty. We will walk through how to register a managed cluster in the next section.

   > **Note**: The management cluster is not displayed in this view. It is not registered and cannot be edited or removed.
   {: .alert .alert-info}

##### Install a managed cluster

Once a management cluster is up and running, the next step is to register a managed cluster with the management plane and install the cluster.

###### Add a managed cluster to the management plane

1. To add managed clusters to your management cluster, click the Add Cluster button.

1. Create a name for your managed cluster (e.g. `application-cluster`) and click the Create Cluster button.

   > **Note**: The name you choose for your managed cluster is persisted in the management cluster and is used to uniquely identify this cluster. This unique name is used for storing cluster specific data like flow logs. You cannot rename the managed cluster after it is created.
   {: .alert .alert-info}

1. After clicking Create Cluster, download the manifest YAML file containing configuration for your managed cluster (the file is named `<your-chosen-cluster-name>.yaml`).

   > **Note**: The manifest you download in this step will be used later on to install configuration for your corresponding managed cluster.
   {: .alert .alert-info}

1. After you click the Close button, you will be taken back to the managed clusters view with your new managed cluster.

1. Within the managed cluster manifest you previously downloaded from the {{site.prodname}} Manager, navigate to the `managementClusterAddr` field for `ManagementClusterConnection` custom resource (shown in the snippet below) and set the host and port to be the corresponding values of the service you created during the installation of the management cluster.

   ```
   apiVersion: operator.tigera.io/v1
   kind: ManagementClusterConnection
   metadata:
     name: tigera-secure
   spec:
     # ManagementClusterAddr should be the externally reachable address to which your managed cluster
     # will connect. Valid examples are: "0.0.0.0:31000", "example.com:32000", "[::1]:32500"
     managementClusterAddr: "${HOST}:${PORT}"
   ```

###### Install {{site.prodname}} on the managed cluster

1. Install the Tigera operators and custom resource definitions.

   ```
   kubectl create -f {{ "/manifests/tigera-operator.yaml" | absolute_url }}
   ```

1. Install your pull secret.

   ```
   kubectl create secret generic tigera-pull-secret \
       --from-file=.dockerconfigjson=<path/to/pull/secret> \
       --type=kubernetes.io/dockerconfigjson -n tigera-operator
   ```

1. Install the Tigera custom resources. For more information on configuration options available in this manifest, see [the installation reference]({{site.baseurl}}/reference/installation/api).

   Download the custom resources YAML to your local directory.
   ```
   curl -O -L {{ "/manifests/custom-resources.yaml" | absolute_url }}
   ```

   Remove the Manager custom resource from the manifest file (see the below example). The Manager component does not run within a managed cluster, since it will be controlled by the centralized management plane.
   ```
   apiVersion: operator.tigera.io/v1
   kind: Manager
   metadata:
     name: tigera-secure
   spec:
     auth:
       type: Token
   ```

   Remove the LogStorage custom resource from the manifest file (see the below example). Similar to the Manager, the LogStorage component (Elasticsearch) does not run within a managed cluster.
   ```
   apiVersion: operator.tigera.io/v1
   kind: LogStorage
   metadata:
     name: tigera-secure
   spec:
     nodes:
       count: 1
   ```

   Update the manifest file by changing the `clusterManagementType` field for the Installation custom resource from `Standalone` to `Managed`. 

   Now, install the modified manifest.
   ```
   kubectl create -f ./custom-resources.yaml
   ```

   You can now monitor progress with the following command:
   ```
   watch kubectl get tigerastatus
   ```

   Wait until the `apiserver` shows a status of `Available`, then proceed to the next section.

###### Install a {{site.prodname}} license on the managed cluster

In order to use {{site.prodname}}, you must install the license provided to you by Tigera.

```
kubectl create -f </path/to/license.yaml>
```

You can now monitor progress with the following command:

```
watch kubectl get tigerastatus
```

When all components (except for Manager & LogStorage) show a status of `Available`, proceed to the next section.

###### Secure {{site.prodname}} on the managed cluster with network policy

To secure {{site.prodname}} component communications, install the following set of network policies.

```
kubectl create -f {{ "/manifests/tigera-policies-managed.yaml" | absolute_url }}
```

##### Post-installation

###### Configure cross-cluster user permissions

To ensure a user logged into the {{site.prodname}} Manager in the management plane has permissions to access resources within a managed cluster (e.g. Policies, NetworkSets, etc.), the following requirements must be met:

- The same user / service account exists in both the management and managed clusters
  - This means the name used within the RoleBinding or ClusterRoleBinding must match
- The user / service account in the managed cluster has the permissions to access relevant resources
  - The user / service account must be bound to a role or cluster role with sufficient privileges

{{site.prodname}} uses [Kubernetes user impersonation](https://kubernetes.io/docs/reference/access-authn-authz/authentication/#user-impersonation) when sending requests from the management plane down into a managed cluster. It is assumed that the user logged into the management plane has a corresponding user with the same name in the managed cluster.

{{site.prodname}} provides some default cluster roles that you can assign to your users. For more information on how to assign the default cluster roles to your users, see [Log in to Calico Enterprise Manager UI]({{site.baseurl}}/getting-started/create-user-login).

###### Access log data in Kibana for a specific managed cluster

1. Log in to the {{site.prodname}} Manager. For more information on how to access the Manager, see [Configure access to the manager UI]({{site.baseurl}}/getting-started/access-the-manager).
2. Click on Kibana from the side navigation.
3. Log in to the Kibana dashboard. For more information on how to log into your Kibana dashboard, see [Accessing logs from Kibana]({{site.baseurl}}/security/logs/elastic/view#accessing-logs-from-kibana)
4. Navigate to Discovery view and filter logs by managed cluster indexes. Select a type of log (e.g. audit, dns, events, flow). Then, from the “Available Fields” section in the side panel, select the _index field.  

   In the example above, the selected log type (flow logs) has the index prefix, `tigera_secure_ee_flow` and three cluster indexes available:
   - Index: tigera_secure_ee_flows.cluster.20200131
   - Index: tigera_secure_ee_flows.application-cluster.20200131
   - Index: tigera_secure_ee_flows.app2.20200131

   > **Note**: The management cluster has a default cluster name to identify indexes. When filtering logs for the management cluster, use this cluster name: cluster.
   {: .alert .alert-info}

   To filter log data by a given managed cluster you can include the filter criteria `_index: <log type>.<cluster name>.*` to your query when executing a search through the Kibana UI.

###### Configure permissions to access log data per managed cluster

Log data across all managed clusters are stored in a centralized Elasticsearch within the management cluster. You can use [Kubernetes RBAC](https://kubernetes.io/docs/reference/access-authn-authz/rbac/) roles and cluster roles to define granular access to cluster log data. For example, using the RBAC rule syntax, you can define rules to control access to specific log types or specific clusters by using the `resources` and `resourceNames` list fields.

{{site.prodname}} log data is stored within Elasticsearch indexes. The indexes have the following naming scheme: 
```
<log type>.<cluster name>.<date>
```

A standalone cluster uses the cluster name `cluster` for Elasticsearch indexes. This is also the name used by a management cluster. For a managed cluster, its cluster name is the value chosen by the user at the time of registration, through the {{site.prodname}} Manager.

To restrict to a specific cluster or subset of clusters use resources. To restrict to a specific log type use resourceNames.
The following are valid cluster types: 
- "flows"
- "audit*"
- "audit_ee"
- "audit_kube"
- "events"
- "dns"

Let’s look at some examples for defining RBAC rules within a role or cluster role to restrict access to log data by log type and cluster name.

The rule below allows access to log types flow and DNS for a single cluster with the name `app-cluster`.

```
- apiGroups: ["lma.tigera.io"]
  resources: ["app-cluster"]
  resourceNames: ["flows", "dns"]
  verbs: ["get"]
```

   > **Note**: The apiGroups will always be `lma.tigera.io`. The verbs will always be `get`.
   {: .alert .alert-info}

The rule below allows access to any cluster for log types flow, DNS and audit.

```
- apiGroups: ["lma.tigera.io"]
  resources: ["*"]
  resourceNames: ["flows", "dns", "audit"]
  verbs: ["get"]
```

The rule below allows access to any cluster for all log types.

```
- apiGroups: ["lma.tigera.io"]
  resources: ["*"]
  resourceNames: ["*"]
  verbs: ["get"]
```

### Above and beyond

- [Configure access to the manager UI]({{site.baseurl}}/getting-started/access-the-manager)
- [Log in to {{site.prodname}} Manager UI]({{site.baseurl}}/getting-started/create-user-login)
- [Accessing logs from Kibana]({{site.baseurl}}/security/logs/elastic/view#accessing-logs-from-kibana)
- Installation API reference
- ManagementClusterConnection resource reference
- Adjust log storage size