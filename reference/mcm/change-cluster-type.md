---
title: Post-installation
description: Change your cluster management type to or from Standalone, Management or Managed.
canonical_url: '/reference/mcm/change-cluster-type'
---

## Big Picture
In this page we will show you how to change the type of a cluster. The easiest way to find out your current
type is by using `kubectl`. The field `clusterManagementType` is part of the `Installation` spec.
```bash
kubectl get installation -o yaml
```


## Change a Standalone cluster into a management cluster
1.  Change your installation type to Management.
    ```bash
    kubectl patch installations.operator.tigera.io default --type merge -p '{"spec":{"clusterManagementType":"Management"}}'
    ```
1.  Create a service to expose the management cluster. For high availability, we recommend that you change the type
    of the service for a more scalable option. For more information see our [post-installation instructions]({{site.baseurl}}/reference/mcm/post-installation).
                                                       
    Apply the following service manifest:
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

## Change a Management cluster into a Standalone cluster
1.  Change your installation type to Standalone.
    ```bash
    kubectl patch installations.operator.tigera.io default --type merge -p '{"spec":{"clusterManagementType":"Standalone"}}'
    ```
1.  Delete the service that you created to expose the management cluster. This example assumes that the service is called
    `tigera-manager-mcm`.
    ```bash
    kubectl delete service tigera-manager-mcm -n tigera-manager
    ```
1.  Delete all ManagedCluster resources that are still present in your cluster.
    ```bash
    kubectl delete managedcluster --all
    ```

> **Note**: While the operator will automatically clean up the credentials that were created for managed clusters, the
            data will still be retained in the Elasticsearch cluster of your management cluster for as long as you have 
            specified in the retention section of your [LogStorage]({{site.baseurl}}/reference/installation/api).       
{: .alert .alert-info}

## Change a Standalone cluster into a Managed cluster
For these instructions we assume you already have a management cluster up and running.
1.  Inside your **Management** cluster, log into the manager UI. Under the `Managed Clusters` page, add a managed cluster and 
    download the manifest. Complete the manifest by filling in the `ManagementClusterConnection` field. The 
    [quickstart guide]({{site.baseurl}}/reference/mcm/quickstart#add-a-managed-cluster-to-the-management-plane) 
    elaborates on this step with screenshots. 
    
    Apply the manifest to your cluster.
    ```bash
    kubectl apply -f <your-managed-cluster.yaml>
    ```
    You are now ready to apply the necessary changes to your **Standalone** cluster.
1.  Change your installation type of your Standalone cluster to Managed.
    ```bash
    kubectl patch installations.operator.tigera.io default --type merge -p '{"spec":{"clusterManagementType":"Managed"}}'
    ```
1.  Remove the resources that are not necessary in Managed clusters. 
    ```bash
    kubectl delete manager tigera-secure
    kubectl delete logstorage tigera-secure
    ```
1.  Replace the network policies that are used by Standalone clusters with the network policies for Managed clusters.
    ```bash
    kubectl delete -f {{ "/manifests/tigera-policies.yaml" | absolute_url }}
    kubectl create -f {{ "/manifests/tigera-policies-managed.yaml" | absolute_url }}
    ```
1.  You can now monitor progress with the following command until everything is `Available`.
    ```bash
    watch kubectl get tigerastatus
    ```
1.  In the **Management** cluster, verify that your cluster is connected. Log into the manager UI. On the `Managed Clusters` 
    page, verify that your cluster has `Connection status: Connected`.

## Change a Managed cluster into a Standalone cluster
1.  Remove the ManagementClusterConnection from your cluster.
     ```bash
     kubectl delete managementclusterconnection tigera-secure
     kubectl delete secret tigera-managed-cluster-connection -n tigera-operator
     ```
1.  Install the Tigera custom resources. For more information on configuration options available in this manifest, see [the installation reference]({{site.baseurl}}/reference/installation/api).
    ```bash
    kubectl apply -f {{ "/manifests/custom-resources.yaml" | absolute_url }}
    ```
1.  Delete the network policies that secure managed clusters.
    ```bash
    kubectl delete -f {{ "/manifests/tigera-policies-managed.yaml" | absolute_url }}
    ```
1.  You can now monitor progress with the following command:
    ```bash
    watch kubectl get tigerastatus
    ```
    When all components show a status of `Available`, proceed to the next step.
1.  Apply the network policies that secure standalone clusters.
    ```bash
    kubectl create -f {{ "/manifests/tigera-policies.yaml" | absolute_url }}
    ```
