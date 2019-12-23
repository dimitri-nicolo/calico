---
title: Upgrading CNX in Kubernetes
---

This document covers upgrading an open source Calico cluster to {{site.tseeprodname}}.

The upgrade procedure is supported for Calico v3.0.x.

## Upgrading an open source Calico cluster to {{site.tseeprodname}}

1. [Upgrade the open source Calico cluster](https://docs.projectcalico.org/v3.0/getting-started/kubernetes/upgrade/)
1. [Add {{site.tseeprodname}}](#adding-cnx).

## Adding {{site.tseeprodname}}
This section covers taking an existing Kubernetes system with Calico and adding {{site.tseeprodname}}.

#### Prerequisites
This procedure assumes the following:

* Your system is running the latest 3.0.x release of Calico. If not, follow the [instructions to upgrade](https://docs.projectcalico.org/v3.0/getting-started/kubernetes/upgrade/) it to the latest 3.0.x release
* You have obtained the {{site.tseeprodname}} specific binaries by following the instructions in [getting started]({{site.baseurl}}/{{page.version}}/getting-started/) and uploaded them to a private registry.
* You have the Calico manifest that was used to install your system available. This is the manifest which includes the `calico/node` DaemonSet.
{% include {{page.version}}/cnx-k8s-apiserver-requirements.md %}

#### Prepare for the Upgrade
 Edit your **calico** manifest:
   - change the calico/node `image:` key to point at the {{site.tseeprodname}} `tigera/cnx-node` image in your private registry.
   - if you're using Typha, change the calico/typha `image:` key to point at the {{site.tseeprodname}} `tigera/typha` image in your private registry.
   - add the following to the `env:` section of the `calico-node` Daemonset:

     ```
     - name: FELIX_PROMETHEUSREPORTERENABLED
       value: "true"
     - name: FELIX_PROMETHEUSREPORTERPORT
       value: "9081"
     ```

   - add the following policy at the end of the `CustomResourceDefinition` section:

     ```
     ---

     apiVersion: apiextensions.k8s.io/v1beta1
     description: Calico Tiers
     kind: CustomResourceDefinition
     metadata:
       name: tiers.crd.projectcalico.org
     spec:
       scope: Cluster
       group: crd.projectcalico.org
       version: v1
       names:
        kind: Tier
        plural: tiers
        singular: tier
     ```

Edit your **rbac-kdd** manifest if you used one:
  - add `tiers` to the list of resources under the `crd.projectcalico.org` apiGroup.

#### Perform the upgrade
 1. Use the following command to initiate a rolling update, using the the Calico manifest you prepared above.

    ```
    kubectl apply -f calico.yaml
    ```

    You can check the status of the rolling update by running:

    ```
    kubectl -n kube-system rollout status ds/calico-node
    ```

1. Download the {{site.tseeprodname}} manifest
([etcd](installation/hosted/cnx/1.7/cnx-etcd.yaml) or [KDD](installation/hosted/cnx/1.7/cnx-kdd.yaml))
defining the {{site.tseeprodname}} Manager API server and {{site.tseeprodname}}
Manager web application resources.

1. Rename the file cnx.yaml - this is what subsequent instructions will refer to

{% include {{page.version}}/cnx-mgr-install.md %}

{% include {{page.version}}/gs-next-steps.md %}
