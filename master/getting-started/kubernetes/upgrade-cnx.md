---
title: Upgrading CNX in Kubernetes
---

This document covers upgrading an open source Calico cluster to {{site.prodname}}.

The upgrade procedure is supported for Calico v3.0.x.

## Upgrading an open source Calico cluster to {{site.prodname}}

1. [Upgrade the open source Calico cluster](https://docs.projectcalico.org/v3.0/getting-started/kubernetes/upgrade/)
1. [Add {{site.prodname}}](#adding-cnx).

## Adding {{site.prodname}}
This section covers taking an existing Kubernetes system with Calico and adding {{site.prodname}}.

#### Prerequisites
This procedure assumes the following:

- Your system is running the latest 3.0.x release of Calico. If not, follow the [instructions to upgrade](https://docs.projectcalico.org/v3.0/getting-started/kubernetes/upgrade/) it to the latest 3.0.x release
- You have obtained the {{site.prodname}} specific binaries by following the instructions in [getting started]({{site.baseurl}}/{{page.version}}/getting-started/) and uploaded them to a private registry.
- You have the Calico manifest that was used to install your system available. This is the manifest which includes the `calico/node` DaemonSet.

#### Prepare for the Upgrade
 Edit your **calico** manifest:
   - change the calico/node `image:` key to point at the {{site.prodname}} `tigera/cnx-node` image in your private registry.
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

Edit your **rbac-kdd** manifest:
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

 1. Install the policy query and violation alerting tools. For more information about the following instructions, see [{{site.prodname}} Hosted Install](installation/hosted/cnx/cnx).

    - Configure calico-monitoring namespace and deploy Prometheus Operator by
      applying the [operator.yaml](installation/hosted/cnx/1.7/operator.yaml) manifest.

      ```
      kubectl apply -f operator.yaml
      ```

    - Wait for custom resource definitions to be created. Check by running:

      ```
      kubectl get customresourcedefinitions --watch
      ```

    - Apply the [monitor-calico.yaml](installation/hosted/cnx/1.7/monitor-calico.yaml) manifest which will
      install prometheus and alertmanager.

      ```
      kubectl apply -f monitor-calico.yaml
      ```

 1. Add the {{site.prodname}} API Server and {{site.prodname}} Manager.

     **Note**: This step may require API downtime, because the API server's command line flags will probably need changing.
    For more information about the following instructions, see [{{site.prodname}} Hosted Install](installation/hosted/cnx/cnx).
    {: .alert .alert-warn}

    - [Decide on an authentication method, and configure Kubernetes]({{site.baseurl}}/{{page.version}}/reference/cnx/authentication).

    - Download the {{site.prodname}} manifest ([etcd](installation/hosted/cnx/1.7/cnx-etcd.yaml)
      or [KDD](installation/hosted/cnx/1.7/cnx-kdd.yaml))
      defining the {{site.prodname}} Manager API server and {{site.prodname}}
      Manager web application resources.

    - Rename the file cnx.yaml - this is what subsequent instructions will refer to

    - Update the manifest with the path to your private Docker registry.

    - See the [main installation documentation]({{site.baseurl}}/{{page.version}}/getting-started/kubernetes/installation/hosted/cnx/cnx) for details on how to set the flags.

    - The `tigera-cnx-manager-config` ConfigMap at the start of the manifest file
      defines three parameters that may need changing: the authentication type,
      the OIDC client ID (only if using Google login), and the Kubernetes API
      location (must be reachable from any system running the web application).

    - Apply the manifest

     ```
     kubectl apply -f cnx.yaml
     ```

    - Define RBAC permissions for users to access the {{site.prodname}} Manager.
      [This document]({{site.baseurl}}/{{page.version}}/reference/cnx/rbac-tiered-policies) describes how to do that.
