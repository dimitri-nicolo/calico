---
title: Upgrade Calico Enterprise from an earlier release on OpenShift
description: Upgrading from an earlier release of Calico Enterprise on OpenShift.
canonical_url: /maintenance/openshift-upgrade
show_toc: false
---

## Prerequisites

Ensure that your {{site.prodname}} OpenShift cluster is running OpenShift
version v4.2 or v4.3, and the {{site.prodname}} operator version is v1.2.4 or greater.

**Note**: You can check if you are running the operator by checking for the existence of the operator namespace
with `oc get ns tigera-operator` or issuing `oc get tigerastatus`; a successful return means your installation is
using the operator.
{: .alert .alert-info}

### Prepare your cluster for the upgrade

During upgrade, the {{site.prodname}} LogStorage CR is temporarily removed so Elasticsearch can be upgraded. Features 
that depend on LogStorage are temporarily unavailable, including dashboards in the Manager UI. Data ingestion is paused 
temporarily, but resumes when the LogStorage is up and running again.

To retain data from your current installation (optional), ensure that the currently mounted persistent volumes 
have their reclaim policy set to [retain data](https://kubernetes.io/docs/tasks/administer-cluster/change-pv-reclaim-policy/).
Data retention is recommended only for users that have a valid Elasticsearch license. (Trial licenses can be invalidated 
during upgrade).

{% include content/hostendpoints-upgrade.md orch="OpenShift" %}

### Download the new manifests

Make a manifests directory.

```bash
mkdir manifests
```

{% include content/openshift-manifests.md %}

## Upgrading to {{page.version}} {{site.prodname}}

1. Export your current LogStorage CR to a file.
   ```bash
   oc get logstorage tigera-secure -o yaml --export=true > log-storage.yaml
   ```

1. Delete the LogStorage CR.
   ```bash
   oc delete -f log-storage.yaml
   ```

1. Verify that Elasticsearch and Kibana are completely removed and that your persistent volumes are no longer bound.
   ```bash
   oc get kibana -n tigera-kibana
   oc get elasticsearch -n tigera-elasticsearch
   oc get pv | grep tigera
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
   oc patch pv $PV_NAME -p '{"spec":{"claimRef":null}}'
   ```

1. Delete outdated CRDs.
   ```bash
   oc delete crd trustrelationships.elasticsearch.k8s.elastic.co  \
   	apmservers.apm.k8s.elastic.co \
   	elasticsearches.elasticsearch.k8s.elastic.co \
   	kibanas.kibana.k8s.elastic.co
   ```

1. Cleanup Elasticsearch operator webhook.
   ```bash
   oc delete validatingwebhookconfigurations validating-webhook-configuration
   oc delete service -n tigera-eck-operator elastic-webhook-service
   ```

1. Apply the updated manifests.
   ```bash
   oc apply -f manifests/
   ```

1. Apply the LogStorage CR.
   ```bash
   oc apply -f log-storage.yaml
   ```   

1. To secure the components which make up {{site.prodname}}, install the following set of network policies.
   ```bash
   oc create -f {{ "/manifests/tigera-policies-openshift.yaml" | absolute_url }}
   ```

1. You can now monitor the upgrade progress with the following command:
   ```bash
   watch oc get tigerastatus
   ```

1. If you were upgrading from a version of Calico Enterprise prior to v3.0 and followed the pre-upgrade steps for host endpoints above, review traffic logs from the temporary policy,
add any global network policies needed to whitelist traffic, and delete the temporary network policy **allow-all-upgrade**.

