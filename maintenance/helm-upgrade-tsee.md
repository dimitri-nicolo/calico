---
title: Upgrade Calico Enterprise from an earlier release using Helm
description: Upgrade from an earlier release of Calico Enterprise using Helm.
canonical_url: /maintenance/helm-upgrade-tsee
---

## Prerequisites

The following upgrade will work only if you are using Calico Enterprise 2.4.2 or higher, and you used helm charts to 
install Calico Enterprise.

Verify that your Kubernetes cluster qualifies for upgrade, and that Tiller is installed on the cluster.

> **Note**: The following instructions assume that Tiller is still installed on
> your cluster.
>
> **Note**: For additional helm documentation, please refer to our
> [**helm installation docs**]({{site.baseurl}}/reference/other-install-methods/kubernetes/installation/helm/).
{: .alert .alert-info}

## Prepare your cluster for the upgrade

During the upgrade the controller that manages Elasticsearch is updated. Because of this, the Elasticsearch cluster is 
temporarily removed during the upgrade. Features that depend on LogStorage are temporarily unavailable, among which
are the dashboards in the Manager UI. Data ingestion is temporarily paused and will continue when Elasticsearch is up
and running again. Therefore, we recommend the following steps:
1. If it exists, delete the install Job from the previous {{site.prodname}} install, (Jobs are immutable.)
   ```bash
   kubectl delete -n calico-monitoring job elastic-tsee-installer
   ```

1. If you are running v2.6.0 or higher, temporarily remove Elasticsearch and Kibana.
   ```bash
   kubectl delete kibana tigera-kibana -n calico-monitoring
   kubectl delete elasticsearch tigera-elasticsearch -n calico-monitoring
   ```

1. Retain data from your current installation (optional), by ensuring that the currently-mounted persistent volumes 
   have their reclaim policy set to [retain data](https://kubernetes.io/docs/tasks/administer-cluster/change-pv-reclaim-policy/).
   This step is only recommended for users that use a valid Elastic license. Trial licenses can get invalidated during 
   the upgrade.

## Upgrade to {{page.version}} {{site.prodname}}

If your {{site.prodname}} was previously installed using helm, follow these steps to upgrade:

1. Find the helm installation names. We will use these names in the following
   upgrade steps.
   ```bash
   helm list
   ```

   Your output should look similar to the following:
   ```bash
   NAME                    REVISION        UPDATED                         STATUS          CHART                  APP VERSION     NAMESPACE
   coiled-bat              1               Fri Jul 19 13:44:37 2019        DEPLOYED        tigera-secure-ee-core-                 default
   fashionable-anteater    1               Fri Jul 19 14:28:50 2019        DEPLOYED        tigera-secure-ee-
   ```

1. Run the Helm upgrade command for `tigera-secure-ee-core`
   ```bash
   helm upgrade <helm installation name for tigera-secure-ee-core> tigera-secure-ee-core-{% include chart_version_name %}.tgz
   ```

1. (Optional) If you choose to retain data, make your persistent volumes ready for reuse. First verify that you have 
   deleted Kibana and Elasticsearch as described above.
   ```bash
   kubectl get kibana,elasticsearch -n calico-monitoring
   ```
   Then, verify that your persistent volumes are no longer bound.
   ```bash
   kubectl get pv | grep tigera-elasticsearch
   ```
   For each volume in the storage class `tigera-elasticsearch`, make sure it is available again.
   ```bash
   PV_NAME=<name-of-your-pv>
   kubectl patch pv $PV_NAME -p '{"spec":{"claimRef":null}}'
   ```

1. First remove and then re-apply the CRDs.
   ```bash
   kubectl delete -f {{ "/reference/other-install-methods/kubernetes/installation/helm/calico-enterprise/operator-crds.yaml" | absolute_url }}
   kubectl apply -f {{ "/reference/other-install-methods/kubernetes/installation/helm/calico-enterprise/operator-crds.yaml" | absolute_url }}
   ```

1. Cleanup Elasticsearch operator webhook.
   ```bash
   kubectl delete validatingwebhookconfigurations validating-webhook-configuration
   kubectl delete service -n calico-monitoring elastic-webhook-service
   ```

1. Run the helm upgrade command for `tigera-secure-ee`.
   ```bash
   helm upgrade <helm installation name for tigera-secure-ee> tigera-secure-ee-{% include chart_version_name %}.tgz --set createCustomResources=false
   ```
