---
title: Upgrade from Calico to Calico Enterprise
description: Steps to upgrade from open source Calico to Calico Enterprise.
canonical_url: /maintenance/upgrade-to-tsee
ignore_installation_cr: true
cr_directory: manifests_cr
---

{% assign calico_minor_version = site.data.versions.first["calico"].minor_version %}
{% assign archive_path = site.data.versions.first["calico"].archive_path %}

{% if archive_path and archive_path != "" %}
{% capture calico_minor_version_with_path %}{{ archive_path }}/{{ calico_minor_version }}{% endcapture %}
{% else %}
{% assign calico_minor_version_with_path = {{ calico_minor_version }} %}
{% endif %}

## Prerequisite
Ensure that your Kubernetes cluster is running with open source Calico on the latest {{ calico_minor_version | append: '.x' }}
release. If not, follow the [Calico upgrade documentation](https://docs.projectcalico.org/{{calico_minor_version_with_path}}/maintenance/kubernetes-upgrade) before continuing.

{{site.prodname}} only supports clusters with a Kubernetes datastore. Please contact Tigera Support for assistance upgrading a
cluster with an `etcdv3` datastore.

If your cluster already has {{site.prodname}} installed, follow the [Upgrading {{site.prodname}} from an earlier release guide]({{site.baseurl}}/maintenance/kubernetes-upgrade-tsee)
instead.

## Upgrade Calico to {{site.prodname}}

### Upgrade a Kubernetes cluster

{% include content/upgrade-operator-simple.md upgradeFrom="OpenSource" %}

### Upgrade managed cloud clusters (EKS)

Follow a slightly modified operator-based install process for {{site.prodname}}
for your EKS cluster: substitute `kubectl apply` in place of `kubectl create`.

For example, in order to upgrade an [EKS cluster]({{site.baseurl}}/getting-started/kubernetes/managed-public-cloud/eks):

   ```
   kubectl apply -f {{ "/manifests/tigera-operator.yaml" | absolute_url }}              # replace "create" with "apply"
   kubectl apply -f {{ "/manifests/tigera-prometheus-operator.yaml" | absolute_url }}   # replace "create" with "apply"
   kubectl apply -f {{ "/manifests/eks/custom-resources.yaml" | absolute_url }}         # replace "create" with "apply"
   ```

> **Note**: When upgrading from older versions of open source Calico, delete the unused
> deployment `calico-typha-horizontal-autoscaler` with the following command:
>  ```
>  kubectl -n kube-system delete deployments.apps calico-typha-horizontal-autoscaler
>  ```
> 
> **Note**: EKS upgrades from open source Calico are not production ready due to limited testing.
> GKE and AKS upgrades from open source Calico are not currently supported.
{: .alert .alert-info}

### Upgrade OpenShift clusters


> **Note**: Operator-based upgrades from open source Calico are not recommended for production clusters due to limited testing. Upgrades not tested with open source Calico prior to v3.15.
{: .alert .alert-info}

#### Download the new manifests

Make the manifests directories.

```bash
mkdir manifests manifests_cr
```

{% include content/openshift-manifests.md %}

{% include content/openshift-prometheus-operator.md %}

{% include content/openshift-resources.md %}

#### Add an image pull secret

1. Download the pull secret manifest template into the manifests directory.

   ```
   curl {{ "/manifests/ocp/02-pull-secret.yaml" | absolute_url }} -o manifests/02-pull-secret.yaml
   ```

1. Update the contents of the secret with the image pull secret provided to you by Tigera.

   For example, if the secret is located at `~/.docker/config.json`, run the following commands.

   ```bash
   SECRET=$(cat ~/.docker/config.json | tr -d '\n\r\t ' | base64 -w 0)
   sed -i "s/SECRET/${SECRET}/" manifests/02-pull-secret.yaml
   ```

#### Install {{site.prodname}}

1. [Configure a storage class for {{site.prodname}}.]({{site.baseurl}}/getting-started/create-storage)

1. Export your current CRs for PrometheusRule to a file.
   ```bash
   oc get prometheusrules.monitoring.coreos.com -A -o yaml > prometheusrules.yaml
   ```

1. Delete the PrometheusRule CRD.
    ```bash
   oc delete crd prometheusrules.monitoring.coreos.com 
   ```
   
1. Apply the Tigera operators and custom resource definitions.
   ```bash
   oc apply -f manifests/
   ```

1. (Optional) If your cluster architecture requires any custom [{{site.prodname}} resources]({{site.baseurl}}/reference/resources) to function at startup, install them now using [calicoctl]({{site.baseurl}}/reference/calicoctl/overview).

1. Apply the Tigera custom resources. For more information on configuration options available, see [the installation reference]({{site.baseurl}}/reference/installation/api).
   ```bash
   oc apply -f manifests_cr/
   ```

1. Patch installation.
   ```bash
   oc patch installations.operator.tigera.io default --type merge -p '{"spec":{"variant":"TigeraSecureEnterprise","imagePullSecrets":[{"name":"tigera-pull-secret"}]}}'
   ```

1. Apply the CRs for PrometheusRule.
   ```bash
   oc apply -f prometheusrules.yaml  
   ```   

1. You can now monitor the upgrade progress with the following command:
   ```bash
   watch oc get tigerastatus
   ```

   Wait until the `apiserver` shows a status of `Available`, then proceed to the next section.


#### Install the {{site.prodname}} license

In order to use {{site.prodname}}, you must install the license provided to you by Tigera.

```
oc create -f </path/to/license.yaml>
```

You can now monitor progress with the following command:

```
watch oc get tigerastatus
```

When all components show a status of `Available`, proceed to the next section.

#### Secure {{site.prodname}} with network policy

To secure {{site.prodname}} component communications, install the following set of network policies.

```bash
oc apply -f {{ "/manifests/tigera-policies.yaml" | absolute_url }}
```
