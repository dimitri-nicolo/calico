---
title: Upgrade Calico Enterprise installed with the operator
description: Upgrading from an earlier release of Calico Enterprise with the operator.
canonical_url: /maintenance/kubernetes-upgrade-tsee
show_toc: false
---

## Prerequisites

Verify that your Kubernetes cluster is using a version of {{site.prodname}} installed with the operator, by running 
`kubectl get tigerastatus`. If the result is successful, then your installation is using the operator.

If your cluster is on a version earlier than 2.6 or does not use the operator, contact Tigera support to upgrade.

If your cluster has a Calico installation, contact Tigera support to upgrade.

## Prepare your cluster for the upgrade

During the upgrade the controller that manages Elasticsearch is updated. Because of this, the {{site.prodname}} LogStorage 
CR is temporarily removed during upgrade. Features that depend on LogStorage are temporarily unavailable, among which
are the dashboards in the Manager UI. Data ingestion is temporarily paused and will continue when the LogStorage is
up and running again.

To retain data from your current installation (optional), ensure that the currently mounted persistent volumes 
have their reclaim policy set to [retain data](https://kubernetes.io/docs/tasks/administer-cluster/change-pv-reclaim-policy/).
Retaining data is only recommended for users that use a valid Elastic license. Trial licenses can get invalidated during 
the upgrade.

{% include content/hostendpoints-upgrade.md orch="Kubernetes" %}

## Upgrade to {{page.version}} {{site.prodname}}

1. Export your current LogStorage CR to a file.
   ```bash
   kubectl get logstorage tigera-secure -o yaml --export=true > log-storage.yaml
   ```

1. Delete the LogStorage CR.
   ```bash
   kubectl delete -f log-storage.yaml
   ```

1. Verify that Elasticsearch and Kibana are completely removed and that your persistent volumes are no longer bound.
   ```bash
   kubectl get kibana -n tigera-kibana
   kubectl get elasticsearch -n tigera
   kubectl get pv | grep tigera-elasticsearch
   ```
   The outputs should look similar to the following:
   ```
   No resources found.
   No resources found.
   pvc-bd2eef7d   10Gi       RWO            Retain           Released   tigera-elasticsearch/tigera-secure-es-gqmh-elasticsearch-data   tigera-elasticsearch            7m24s
   ```

1. (Optional) If you choose to retain data, make your persistent volumes ready for reuse. For each volume in the storage 
   class that you specified in `log-storage.yaml`, make sure it is available again. By default the storage class name is 
   `tigera-elasticsearch`.
   ```bash
   PV_NAME=<name-of-your-pv>
   kubectl patch pv $PV_NAME -p '{"spec":{"claimRef":null}}'
   ```

1. Delete outdated CRDs.
   ```bash
   kubectl delete crd trustrelationships.elasticsearch.k8s.elastic.co  \
   	apmservers.apm.k8s.elastic.co \
   	elasticsearches.elasticsearch.k8s.elastic.co \
   	kibanas.kibana.k8s.elastic.co
   ```

1. Cleanup Elasticsearch operator webhook.
   ```bash
   kubectl delete validatingwebhookconfigurations validating-webhook-configuration
   kubectl delete service -n tigera-eck-operator elastic-webhook-service
   ```

1. Download the new operator manifest.
   ```bash
   curl -L -O {{ "/manifests/tigera-operator.yaml" | absolute_url }}
   ```

1. If you previously [installed using a private registry]({{site.baseurl}}/getting-started/private-registry), you will need to
   [push the new images]({{site.baseurl}}/getting-started/private-registry#push-calico-enterprise-images-to-your-private-registry)
   and then [update the manifest]({{site.baseurl}}/getting-started/private-registry#run-the-operator-using-images-from-your-private-registry)
   downloaded in the previous step.

   **Note**: There is no need to update `custom-resources.yaml` or
   [configure the operator]({{site.baseurl}}/getting-started/private-registry#configure-the-operator-to-use-images-from-your-private-registry).
   {: .alert .alert-info}

1. Install the new network policies to secure {{site.prodname}} component communications.
   ```bash
   kubectl apply -f {{ "/manifests/tigera-policies.yaml" | absolute_url }}
   ```

1. Apply the Tigera operator.
   ```bash
   kubectl apply -f tigera-operator.yaml
   ```

1. Apply the LogStorage CR.
   ```bash
   kubectl apply -f log-storage.yaml
   ```

1. You can monitor progress with the following command:
   ```bash
   watch kubectl get tigerastatus
   ```

   **Note**: If there are any problems you can use `kubectl get tigerastatus -o yaml` to get more details.
   {: .alert .alert-info}

1. If you were upgrading from a version of Calico prior to v3.14 and followed the pre-upgrade steps for host endpoints above, review traffic logs from the temporary policy,
   add any global network policies needed to whitelist traffic, and delete the temporary network policy **allow-all-upgrade**.

{% include content/auto-hostendpoints-migrate.md orch="Kubernetes" %}
