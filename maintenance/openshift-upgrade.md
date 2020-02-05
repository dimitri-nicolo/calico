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

{% include content/openshift-manifests.md %}


Next, apply the manifests.
```
oc apply -f manifests/
``` 

To secure the components which make up {{site.prodname}}, install the following set of network policies.
```
oc create -f {{ "/manifests/tigera-policies.yaml" | absolute_url }}
```

You can now monitor the upgrade progress with the following command:
```
watch oc get tigerastatus
```
