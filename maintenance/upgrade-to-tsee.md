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
{% assign calico_minor_version_with_path = calico_minor_version %}
{% endif %}

## Prerequisite

Ensure that your Kubernetes cluster is running with open source Calico on the latest {{ calico_minor_version | append: '.x' }} release.
If not, follow the [Calico upgrade documentation](https://docs.projectcalico.org/{{calico_minor_version_with_path}}/maintenance/kubernetes-upgrade){:target="_blank"} before continuing.

{{site.prodname}} only supports clusters with a Kubernetes datastore.
Please contact Tigera Support for assistance upgrading a cluster with an `etcdv3` datastore.

If your cluster already has {{site.prodname}} installed, follow the [Upgrading {{site.prodname}} from an earlier release guide]({{site.baseurl}}/maintenance/kubernetes-upgrade-tsee) instead.


## Upgrade Calico to {{site.prodname}}

> **Note**: GKE and AKS upgrades from open source Calico are not currently supported.
{: .alert .alert-info}

### Before you begin

**Required**

- [Configure a storage class for {{site.prodname}}.]({{site.baseurl}}/getting-started/create-storage)

### Install {{site.prodname}}

{% tabs %}

<label:Kubernetes,active:true>
<%

{% include content/upgrade-operator-simple.md upgradeFrom="OpenSource" %}

%>

<label: EKS>
<%

> **Note**: EKS upgrades from open source Calico are not production ready due to limited testing.
>
> When upgrading from older versions of open source Calico, delete the unused
> deployment `calico-typha-horizontal-autoscaler` with the following command:
>  ```
>  kubectl -n kube-system delete deployments.apps calico-typha-horizontal-autoscaler
>  ```
{: .alert .alert-info}

{% include content/upgrade-operator-simple.md upgradeFrom="OpenSource" provider="EKS" %}

%>

<label: OpenShift>
<%

> **Note**: Operator-based upgrades from open source Calico are not recommended for production clusters due to limited testing. Upgrades not tested with open source Calico prior to v3.15.
{: .alert .alert-info}

__Download the new manifests__

Make the manifests directories.

```bash
mkdir manifests manifests_cr
```

{% include content/openshift-manifests.md %}

{% include content/openshift-prometheus-operator.md %}

{% include content/openshift-resources.md %}

__Add an image pull secret__

{% include content/openshift-pull-secret.md %}

> (Optional) If your cluster architecture requires any custom [{{site.prodname}} resources]({{site.baseurl}}/reference/resources) to function at startup, install them now using [calicoctl]({{site.baseurl}}/reference/calicoctl/overview).

__Install {{site.prodname}}__

1. Apply the Tigera operators and custom resource definitions.

   ```bash
   oc apply -f manifests/
   ```

2. (Optional) If your cluster architecture requires any custom [{{site.prodname}} resources]({{site.baseurl}}/reference/resources) to function at startup, install them now using [calicoctl]({{site.baseurl}}/reference/calicoctl/overview).

3. Apply the Tigera custom resources. For more information on configuration options available, see [the installation reference]({{site.baseurl}}/reference/installation/api).

   ```bash
   oc apply -f manifests_cr/
   ```

4. Patch installation.

   ```bash
   oc patch installations.operator.tigera.io default --type merge -p '{"spec":{"variant":"TigeraSecureEnterprise","imagePullSecrets":[{"name":"tigera-pull-secret"}]}}'
   ```

5. You can now monitor the upgrade progress with the following command:

   ```bash
   watch oc get tigerastatus
   ```

%>

{% endtabs %}

Wait until the `apiserver` shows a status of `Available`, then proceed to the next section.

**Note**: If there are any problems you can use `kubectl get tigerastatus -o yaml` to get more details.
{: .alert .alert-info}

### Install the {{site.prodname}} license

In order to use {{site.prodname}}, you must install the license provided to you by Tigera.

```
kubectl create -f </path/to/license.yaml>
```

You can now monitor progress with the following command:

```
watch kubectl get tigerastatus
```

When all components show a status of `Available`, proceed to the next section.

### Secure {{site.prodname}} with network policy

To secure {{site.prodname}} component communications, install the following set of network policies.

```bash
kubectl apply -f {{ "/manifests/tigera-policies.yaml" | absolute_url }}
```
