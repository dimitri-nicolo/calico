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

### Upgrade managed cloud clusters (EKS)

Follow a slightly modified operator-based install process for {{site.prodname}}
for your EKS cluster: substitute `kubectl apply` in place of `kubectl create`.

For example, in order to upgrade an [EKS cluster]({{site.baseurl}}/getting-started/kubernetes/managed-public-cloud/eks):

   ```
   kubectl apply -f {{ "/manifests/tigera-operator.yaml" | absolute_url }}       # replace "create" with "apply"
   kubectl apply -f {{ "/manifests/eks/custom-resources.yaml" | absolute_url }}  # replace "create" with "apply"
   ```

> **Note**: GKE and AKS upgrades from open source Calico are not currently supported. EKS is not production
> ready due to limited testing.
{: .alert .alert-info}

### Upgrade OpenShift clusters


> **Note**: Operator-based upgrades from open source Calico are not recommended for production clusters due to limited testing. Upgrades not tested with open source Calico prior to v3.15.
{: .alert .alert-info}

#### Download the new manifests

Make a manifests directory.

```bash
mkdir manifests
```

{% include content/openshift-manifests.md %}

{% include content/openshift-resources.md %}

Remove installation resources.

```bash
rm manifests/01-crd-installation.yaml manifests/01-cr-installation.yaml
```

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

#### Optionally provide additional configuration

You may want to provide {{site.prodname}} with additional configuration at install-time. For example, BGP configuration or peers. You can use a Kubernetes ConfigMap with your desired {{site.prodname}} resources in order to set configuration as part of the installation. If you do not need to provide additional configuration, you can skip this section.

To include [{{site.prodname}} resources]({{site.baseurl}}/reference/resources) during installation, edit `manifests/02-configmap-calico-resources.yaml` in order to add your own configuration.

> **Note**: If you have a directory with the {{site.prodname}} resources, you can create the file with the command:
> ```
> kubectl create configmap -n tigera-operator calico-resources \
>   --from-file=<resource-directory> --dry-run -o yaml \
>   > manifests/02-configmap-calico-resources.yaml
> ```
> With recent versions of kubectl it is necessary to have a kubeconfig configured or add `--server='127.0.0.1:443'`
> even though it is not used.

> **Note**: If you have provided a `calico-resources` configmap and the tigera-operator pod fails to come up with `Init:CrashLoopBackOff`,
> check the output of the init-container with `kubectl logs -n tigera-operator -l k8s-app=tigera-operator -c create-initial-resources`.
{: .alert .alert-info}

#### Install {{site.prodname}}

1. [Configure a storage class for {{site.prodname}}.]({{site.baseurl}}/getting-started/create-storage)

1. Export your current CRs for PrometheusRule to a file.
   ```bash
   oc get prometheusrules.monitoring.coreos.com -n tigera-prometheus -o yaml > prometheusrules.yaml
   ```

1. Delete the PrometheusRule CRD.
    ```bash
   oc delete crd prometheusrules.monitoring.coreos.com 
   ```
   
1. Apply the updated manifests.
   ```bash
   oc apply -f manifests/
   ```

1. Patch installation.

   ```bash
   oc patch installations.operator.tigera.io default --type merge -p '{"spec":{"variant":"TigeraSecureEnterprise","clusterManagementType":"Standalone","imagePullSecrets":[{"name":"tigera-pull-secret"}]}}'
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