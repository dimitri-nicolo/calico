---
title: Quickstart for multi-cluster management
description: Getting started with multi-cluster management.
canonical_url: '/getting-started/mcm/quickstart'
---

## Big picture

At the end of this guide you will have two clusters running; one management cluster and one managed cluster. The guide 
will have optional steps to get you started quickly, which include:
- Setting up a user with `tigera-network-admin` access in the management cluster and `tigera-ui-viewer` access in the managed cluster. 
- Setting up a node port service to expose the management cluster

For both these steps there are more advanced alternatives for production deployments. See 
[post-installation instructions]({{site.baseurl}}/getting-started/mcm/post-installation) for more details.

## Requirements
- You will need 2 Kubernetes clusters to get started with multi-cluster management. For more details, see
 [install Kubernetes]({{site.baseurl}}/getting-started/kubernetes/quickstart#install-kubernetes).
- `kubectl` is installed on each cluster.
- A {{site.prodname}} license and docker registry config file.

## Install the management cluster

The first step is to set up a management cluster. This will contain your centralized {{site.prodname}} Manager and Elasticsearch installation.

### Install {{site.prodname}} on the management cluster

1. [Configure a storage class for {{site.prodname}}]({{site.baseurl}}/getting-started/create-storage).

   > **Note**: Since the management cluster will store all log data across your managed clusters, careful consideration should be made in choosing an appropriate size for the storage class to accommodate your anticipated volume of data. You may also need to adjust the log storage settings accordingly to scale up your centralized Elasticsearch cluster. For more information, see [Adjust log storage size]({{site.baseurl}}/maintenance/adjust-log-storage-size).
   {: .alert .alert-info}

1. Install the Tigera operator and custom resource definitions.

   ```shell
   kubectl create -f {{ "/manifests/tigera-operator.yaml" | absolute_url }}
   ```

1. Install your pull secret.

   ```shell
   kubectl create secret generic tigera-pull-secret \
       --type=kubernetes.io/dockerconfigjson -n tigera-operator \
       --from-file=.dockerconfigjson=<path/to/pull/secret>
   ```

1. Install the Tigera custom resources. For more information on configuration options available in this manifest, see [the installation reference]({{site.baseurl}}/reference/installation/api).

   Download the custom resources YAML to your local directory and set the `clusterManagementType` to `Management`.
   ```shell
   curl -O -L {{ "/manifests/custom-resources.yaml" | absolute_url }}
   sed -i 's/clusterManagementType: Standalone/clusterManagementType: Management/' custom-resources.yaml
   ```

   Now, install the modified manifest.
   ```shell
   kubectl create -f ./custom-resources.yaml
   ```

   You can now monitor progress with the following command:

   ```shell
   watch kubectl get tigerastatus
   ```

   Wait until the `apiserver` and `management-cluster-connection` show a status of `Available`, then proceed to the next section.

### Install the {{site.prodname}} license on the management cluster

In order to use {{site.prodname}}, you must install the license provided to you by Tigera.

```shell
kubectl create -f </path/to/license.yaml>
```

You can now monitor progress with the following command:

```shell
watch kubectl get tigerastatus
```

When all components show a status of `Available`, proceed to the next section.

### Secure {{site.prodname}} on the management cluster with network policy

To secure {{site.prodname}} component communications, install the following set of network policies.

```shell
kubectl create -f {{ "/manifests/tigera-policies.yaml" | absolute_url }}
```

### Setup a service to allow connections to the management plane

In order for managed clusters to connect to the shared management plane, you must allow external clusters to reach the 
`tigera-manager` pod. This can be accomplished by creating a Kubernetes service.

For this quickstart we will use an example `NodePort` service. For high availability, we recommend that you change this 
service for a more scalable option. For more information see our [post-installation instructions]({{site.baseurl}}/getting-started/mcm/post-installation).

Apply the following service manifest:

```shell
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

### Setup an admin user to manage clusters (optional)
In order to add managed clusters to a management cluster, you need to have access to the {{site.prodname}} Manager UI.
Apply these steps to create a privileged user called `mcm-user` in the `default` namespace:
```shell
kubectl create serviceaccount mcm-user -n default
kubectl create clusterrolebinding mcm-user-admin --serviceaccount=default:mcm-user --clusterrole=tigera-network-admin
```
To log into the {{site.prodname}} Manager UI, you can get a secret using this command for token-based authentication:
```shell
{% raw %}kubectl get secret $(kubectl get serviceaccount mcm-user -o jsonpath='{range .secrets[*]}{.name}{"\n"}{end}' | grep token) -o go-template='{{.data.token | base64decode}}' && echo{% endraw %}
```

### Log in to the {{site.prodname}} Manager UI
[Log in]({{site.baseurl}}/getting-started/cnx/access-the-manager) to the {{site.prodname}} Manager using the secret created in the last step.

On the left-hand side there is button called `Managed Clusters` that brings you to the page you will be using to add 
managed clusters.
![Cluster Created]({{site.baseurl}}/images/mcm/mcm-cluster-created.png)
{: .align-center}

You have now successfully installed a management cluster.

## Install the managed cluster

Now that your management cluster is up and running, the next step is to register and install a managed cluster.

### Add a managed cluster to the management plane

1. Navigate to the managed clusters view in the {{site.prodname}} Manager and click the Add Cluster button.

1. Create a name for your managed cluster (e.g. `app-cluster-1`) and click the Create Cluster button.

   ![Create Cluster]({{site.baseurl}}/images/mcm/mcm-create-cluster.png)
   {: .align-center}

   > **Note**: The name you choose for your managed cluster is persisted in the management cluster and is used to uniquely identify this cluster. This unique name is used for storing cluster specific data like flow logs. You cannot rename the managed cluster after it is created.
   {: .alert .alert-info}

1. After clicking Create Cluster, download the manifest YAML file containing configurations for your managed cluster (the file is named `<your-chosen-cluster-name>.yaml`).

   ![Download Manifest]({{site.baseurl}}/images/mcm/mcm-managed-cluster-manifest.png)
   {: .align-center}

   > **Note**: The manifest you download in this step will be used later on to install configuration for your corresponding managed cluster.
   {: .alert .alert-info}

1. After you click the Close button, you will be taken back to the managed clusters view, now showing your new cluster.

   ![Cluster Created]({{site.baseurl}}/images/mcm/mcm-cluster-created.png)
   {: .align-center}

1. Within the managed cluster manifest you previously downloaded from the {{site.prodname}} Manager, navigate to the `managementClusterAddr` field for the `ManagementClusterConnection` custom resource (shown in the example below) and set the host and port to be the corresponding values of the service you created during the installation of the management cluster.

   ```yaml
   apiVersion: operator.tigera.io/v1
   kind: ManagementClusterConnection
   metadata:
     name: tigera-secure
   spec:
     # ManagementClusterAddr should be the externally reachable address to which your managed cluster
     # will connect. Valid examples are: "0.0.0.0:31000", "example.com:32000", "[::1]:32500"
     managementClusterAddr: "${HOST}:${PORT}"
   ```

### Install {{site.prodname}} on the managed cluster

1. Install the Tigera operators and custom resource definitions.

   ```shell
   kubectl create -f {{ "/manifests/tigera-operator.yaml" | absolute_url }}
   ```

1. Install your pull secret.

   ```shell
   kubectl create secret generic tigera-pull-secret \
       --from-file=.dockerconfigjson=<path/to/pull/secret> \
       --type=kubernetes.io/dockerconfigjson -n tigera-operator
   ```

1. Install the Tigera custom resources. For more information on configuration options available in this manifest, see [the installation reference]({{site.baseurl}}/reference/installation/api).

   Download the custom resources YAML to your local directory and set the `clusterManagementType` to `Managed`.

   ```shell
   curl -O -L {{ "/manifests/custom-resources.yaml" | absolute_url }}
   sed -i 's/clusterManagementType: Standalone/clusterManagementType: Managed/' custom-resources.yaml
   ```

   Remove the Manager custom resource from the manifest file (shown in the example below). The Manager component does not run within a managed cluster, since it will be controlled by the centralized management plane.

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

   Remove the LogStorage custom resource from the manifest file (shown in the example below). Similar to the Manager, the LogStorage component (Elasticsearch) does not run within a managed cluster.

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

   ```shell
   kubectl create -f ./custom-resources.yaml
   ```

   Finally, apply the managed cluster manifest you downloaded when you registered the cluster in the management plane (from section [Add a managed cluster to the management plane](#add-a-managed-cluster-to-the-management-plane)).

   ```shell
   kubectl create -f ./<your-chosen-cluster-name>.yaml
   ```

   This will bring up the component called `tigera-guardian`, which will connect the managed cluster to the management plane. All log data will be sent through this component to the centralized Elasticsearch.

   You can now monitor progress with the following command:

   ```shell
   watch kubectl get tigerastatus
   ```

   Wait until the `apiserver` and `management-cluster-connection` show a status of `Available`, then proceed to the next section.

### Install a {{site.prodname}} license on the managed cluster

In order to use {{site.prodname}}, you must install the license provided to you by Tigera.

```shell
kubectl create -f </path/to/license.yaml>
```

You can now monitor progress with the following command:

```shell
watch kubectl get pods --all-namespaces -o wide
```

Wait until you see the `tigera-compliance` namespace get created, then proceed to the next section.

### Secure {{site.prodname}} on the managed cluster with network policy

To secure {{site.prodname}} component communications, install the following set of network policies.

```shell
kubectl create -f {{ "/manifests/tigera-policies-managed.yaml" | absolute_url }}
```

### Give your mcm-user view permissions in the managed cluster (optional)
In order to view data for your managed cluster in the {{site.prodname}} Manager UI, we need to give the user permissions
to view data.

In your managed cluster, apply these steps to create view permissions for `mcm-user` in the `default` namespace:
```shell
kubectl create serviceaccount mcm-user -n default
kubectl create clusterrolebinding mcm-user-admin --serviceaccount=default:mcm-user --clusterrole=tigera-ui-user
```

## Above and beyond

- [Post-installation instructions]({{site.baseurl}}/getting-started/mcm/post-installation)
