---
title: Upgrade from Calico to Calico Enterprise
description: Steps to upgrade from open source Calico to Calico Enterprise.
canonical_url: /getting-started/kubernetes/upgrade/upgrade-to-tsee
---

{% assign calico_minor_version = site.data.versions.first["calico"].minor_version %}

## Prerequisite
Ensure that your Kubernetes cluster is running with open source Calico on the latest {{ calico_minor_version | append: '.x' }}
release. If not, follow the [Calico upgrade documentation](https://docs.projectcalico.org/{{calico_minor_version}}/maintenance/kubernetes-upgrade) before continuing.

{{site.prodname}} only supports clusters with a Kubernetes datastore. Please contact Tigera Support for assistance upgrading a
cluster with an `etcdv3` datastore.

If your cluster already has {{site.prodname}} installed, follow the [Upgrading {{site.prodname}} from an earlier release guide]({{site.baseurl}}/maintenance/kubernetes-upgrade-tsee)
instead.

## Upgrade Calico to {{site.prodname}}

### Upgrade a Kubernetes cluster

{% include content/upgrade-operator-simple.md %}

### Upgrade managed cloud clusters

Follow a slightly modified operator-based install process for {{site.prodname}}
for your platform: substitute `kubectl apply` in place of `kubectl create`.

For example, in order to upgrade an [EKS cluster](getting-started/kubernetes/managed-public-cloud/eks):

   ```
   kubectl apply -f {{ "/manifests/tigera-operator.yaml" | absolute_url }}       # replace "create" with "apply"
   kubectl apply -f {{ "/manifests/eks/custom-resources.yaml" | absolute_url }}  # replace "create" with "apply"
   ```

### Upgrade OpenShift clusters


> **Note**: Operator-based upgrades from open source Calico are not recommended for production clusters due to limited testing.
{: .alert .alert-info}
