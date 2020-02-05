---
title: Upgrading Calico Enterprise installed with the operator
description: Upgrading from an earlier release of Calico Enterprise with the operator.
canonical_url: /maintenance/kubernetes-upgrade-tsee
show_toc: false
---

## Prerequisite

Ensure that your Kubernetes cluster is already running version 2.6 of {{site.prodname}} installed with the operator.
If your cluster is on a version earlier than 2.6 or does not use the operator, contact Tigera support to upgrade.

If your cluster has a Calico installation, contact Tigera support to upgrade.

**Note**: You can check if you are running the operator by checking for the existence of the operator namespace
with `kubectl get ns tigera-operator` or issuing `kubectl get tigerastatus`,
if either of those return successfully then your installation is using the operator.
{: .alert .alert-info}

## Upgrading to {{page.version}} {{site.prodname}}

1. Download the new operator manifest.
   ```
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
   ```
   kubectl apply -f {{ "/manifests/tigera-policies.yaml" | absolute_url }}
   ```

1. Apply the Tigera operator.
   ```
   kubectl apply -f tigera-operator.yaml
   ```

1. You can monitor progress with the following command:
   ```
   watch kubectl get tigerastatus
   ```

   **Note**: If there are any problems you can use `kubectl get tigerastatus -o yaml` to get more details.
   {: .alert .alert-info}
