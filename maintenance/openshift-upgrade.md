---
title: Upgrading Calico Enterprise from an earlier release on OpenShift
description: Upgrading from an earlier release of Calico Enterprise on OpenShift.
canonical_url: /maintenance/openshift-upgrade
show_toc: false
---

## Prerequisite

Ensure that your {{site.prodname}} OpenShift cluster is running OpenShift
version v4.2 or v4.3 and the {{site.prodname}} operator version is v1.0.4.

**Note**: You can check if you are running the operator by checking for the existence of the operator namespace
with `oc get ns tigera-operator` or issuing `oc get tigerastatus`,
if either of those return successfully then your installation is using the operator.
{: .alert .alert-info}

## Upgrading to {{page.version}} {{site.prodname}}

1. Download the {{site.prodname}} manifests for OpenShift and add them to the generated manifests directory:
   ```bash
   mkdir manifests
   curl {{ "/manifests/ocp/crds/01-crd-installation.yaml" | absolute_url }} -o manifests/01-crd-installation.yaml
   curl {{ "/manifests/ocp/crds/01-crd-managementclusterconnection.yaml" | absolute_url }} -o manifests/01-crd-managementclusterconnection.yaml
   curl {{ "/manifests/ocp/tigera-operator/02-role-tigera-operator.yaml" | absolute_url }} -o manifests/02-role-tigera-operator.yaml
   curl {{ "/manifests/ocp/tigera-operator/02-tigera-operator.yaml" | absolute_url }} -o manifests/02-tigera-operator.yaml
   ```

1. Delete the existing operator deployment:
   ```bash
   oc delete deployment -n tigera-operator tigera-operator
   ```

1. Delete the existing Elasticsearch secrets. When the new operator is deployed,
   these secrets will be regenerated.
   ```bash
   oc delete secret -n tigera-operator tigera-ee-compliance-benchmarker-elasticsearch-access
   oc delete secret -n tigera-operator tigera-ee-compliance-controller-elasticsearch-access
   oc delete secret -n tigera-operator tigera-ee-compliance-reporter-elasticsearch-access
   oc delete secret -n tigera-operator tigera-ee-compliance-server-elasticsearch-access
   oc delete secret -n tigera-operator tigera-ee-compliance-snapshotter-elasticsearch-access
   oc delete secret -n tigera-operator tigera-ee-curator-elasticsearch-access
   oc delete secret -n tigera-operator tigera-ee-installer-elasticsearch-access
   oc delete secret -n tigera-operator tigera-ee-intrusion-detection-elasticsearch-access
   oc delete secret -n tigera-operator tigera-ee-manager-elasticsearch-access
   oc delete secret -n tigera-operator tigera-eks-log-forwarder-elasticsearch-access
   oc delete secret -n tigera-operator tigera-fluentd-elasticsearch-access
   ```

1. Next, apply the updated manifests.
   ```bash
   oc apply -f manifests/
   ```

1. To secure the components which make up {{site.prodname}}, install the following set of network policies.
   ```bash
   oc create -f {{ "/manifests/tigera-policies-openshift.yaml" | absolute_url }}
   ```

1. You can now monitor the upgrade progress with the following command:
   ```bash
   watch oc get tigerastatus
   ```
