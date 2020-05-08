---
title: Post-installation
description: Configuring permissions, accessing data and detailed information for multi-cluster management.
canonical_url: '/getting-started/mcm/post-installation'
---

In the [quickstart guide]({{site.baseurl}}/getting-started/mcm/quickstart), we showed how you can easily get started with
multi-cluster management. In this section, we highlight some topics that are important to understand in-depth when 
preparing a multi-cluster management setup for production use.

## Configure cross-cluster user permissions

Any user that logs into the {{site.prodname}} Manager UI must use a valid service account or user account from the 
management cluster. In this cluster, the service account `tigera-manager` will perform a TokenReview for authenticating 
the user. The following authorization checks are performed:
- In the management cluster, RBAC checks are made to verify that the user is allowed to use the UI as described in [configure access to the manager UI]({{site.baseurl}}/getting-started/cnx/access-the-manager).
- In the cluster that is selected in the top-right corner of the screen, to verify that the user has access to the resources 
displayed displayed on the page.

In the quickstart we created a user with the clusterrole `tigera-network-admin` in the management cluster and 
`tigera-ui-user` in the managed cluster. These permissions may be too broad for your use case. We recommend that you 
revise these if necessary. For more information on how to assign the cluster roles to your users, see [log in to Calico Enterprise Manager UI]({{site.baseurl}}/getting-started/cnx/create-user-login).

## Configuring a service to expose the management cluster

In the quickstart guide we exposed the management plane using a NodePort service. While this is the quickest way to expose
the management cluster, there are drawbacks to NodePort services that one should consider. These are outlined in the 
[Kubernetes docs for setting up a service](https://kubernetes.io/docs/concepts/services-networking/service/#defining-a-service).

Regardless of the service type you choose, you must ensure it obeys the following requirements:

- The service uses protocol `TCP`
- The service maps to port `9449` on the Manager pod
- The service exists within the `tigera-manager` namespace
- The service uses label selector `k8s-app: tigera-manager`

As an alternative to the NodePort service, here is an example that demonstrates a LoadBalancer service:
```shell
apiVersion: v1
kind: Service
metadata:
  name: tigera-manager-mcm
  namespace: tigera-manager
spec:
  type: LoadBalancer
  ports:
  - port: 9449
    protocol: TCP
    targetPort: 9449
  selector:
    k8s-app: tigera-manager
```
> **Note**: Note that [LoadBalancers](https://kubernetes.io/docs/concepts/services-networking/#loadbalancer) might take 
            additional setup, depending on how you provisioned your kubernetes cluster.  
{: .alert .alert-info}

> **Note**: Don't forget to update the address in each of your managed clusters, either by editing the 
            ManagementClusterConnection manifest that you [downloaded]({{site.baseurl}}/getting-started/mcm/quickstart#add-a-managed-cluster-to-the-management-plane) 
            and applying it or using kubectl `kubectl edit managementclusterconnection tigera-secure`.
{: .alert .alert-info}

## Access log data in Kibana for a specific managed cluster

1. Log in to the {{site.prodname}} Manager. For more information on how to access the Manager, see [Configure access to the manager UI]({{site.baseurl}}/getting-started/cnx/access-the-manager).
2. Click on Kibana from the side navigation.
3. Log in to the Kibana dashboard. For more information on how to log in to your Kibana dashboard, see [Accessing logs from Kibana]({{site.baseurl}}/security/logs/elastic/view#accessing-logs-from-kibana).
4. Navigate to Discovery view and filter logs by managed cluster indexes. Select a type of log (e.g. audit, dns, events, flow). Then, from the “Available Fields” section in the side panel, select the `_index` field.

   ![Kibana Cluster Indexes]({{site.baseurl}}/images/mcm/mcm-kibana-cluster-indexes.png)
   {: .align-center}

   In the example above, the selected log type (flow logs) has the index prefix, `tigera_secure_ee_flows` and two cluster indexes available:
   - Index: tigera_secure_ee_flows.cluster.20200207
   - Index: tigera_secure_ee_flows.app-cluster-1.20200207

   > **Note**: The management cluster has a default cluster name to identify indexes. When filtering logs for the management cluster, use this cluster name: cluster.
   {: .alert .alert-info}

   To filter log data by a given managed cluster you can include the filter criteria `_index: <log type>.<cluster name>.*` to your query when executing a search through the Kibana UI.

## Configure permissions to access log data per managed cluster

Log data across all managed clusters are stored in a centralized Elasticsearch within the management cluster. You can use [Kubernetes RBAC](https://kubernetes.io/rbac/) roles and cluster roles to define granular access to cluster log data. For example, using the RBAC rule syntax, you can define rules to control access to specific log types or specific clusters by using the `resources` and `resourceNames` list fields.

{{site.prodname}} log data is stored within Elasticsearch indexes. The indexes have the following naming scheme: 

```shell
<log-type>.<cluster-name>.<date>
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

```yaml
- apiGroups: ["lma.tigera.io"]
  resources: ["app-cluster"]
  resourceNames: ["flows", "dns"]
  verbs: ["get"]
```

   > **Note**: The apiGroups will always be `lma.tigera.io`. The verbs will always be `get`.
   {: .alert .alert-info}

The rule below allows access to any cluster for log types flow, DNS and audit.

```yaml
- apiGroups: ["lma.tigera.io"]
  resources: ["*"]
  resourceNames: ["flows", "dns", "audit"]
  verbs: ["get"]
```

The rule below allows access to any cluster for all log types.

```yaml
- apiGroups: ["lma.tigera.io"]
  resources: ["*"]
  resourceNames: ["*"]
  verbs: ["get"]
```

## Above and beyond

- [Configure access to the manager UI]({{site.baseurl}}/getting-started/cnx/access-the-manager)
- [Log in to {{site.prodname}} Manager UI]({{site.baseurl}}/getting-started/cnx/create-user-login)
- [Accessing logs from Kibana]({{site.baseurl}}/security/logs/elastic/view#accessing-logs-from-kibana)
- [Installation API reference]({{site.baseurl}}/reference/installation/api#operator.tigera.io/v1.Installation)
- [ManagementClusterConnection resource reference]({{site.baseurl}}/reference/installation/api#operator.tigera.io/v1.ManagementClusterConnection)
- [Adjust log storage size]({{site.baseurl}}/maintenance/logstorage/adjust-log-storage-size)
