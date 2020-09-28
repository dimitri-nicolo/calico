1. Download the new manifests for Tigera operator.
   ```bash
   curl -L -O {{ "/manifests/tigera-operator.yaml" | absolute_url }}
   ```

1. Download the new manifests for Prometheus operator. This step is required if you previously installed Prometheus operator as part of {{site.prodname}}.
   ```bash
   curl -L -O {{ "/manifests/tigera-prometheus-operator.yaml" | absolute_url }}
   ```

1. If you previously [installed using a private registry]({{site.baseurl}}/getting-started/private-registry), you will need to
   [push the new images]({{site.baseurl}}/getting-started/private-registry/private-registry-regular#push-calico-enterprise-images-to-your-private-registry)
   and then [update the manifest]({{site.baseurl}}/getting-started/private-registry/private-registry-regular#run-the-operator-using-images-from-your-private-registry)
   downloaded in the previous step.

1. Apply the manifests for Tigera operator.
   ```bash
   kubectl apply -f tigera-operator.yaml
   ```

1. If you downloaded the manifests for Prometheus operator from the earlier step, then apply them now.
   ```bash
   kubectl apply -f tigera-prometheus-operator.yaml
   ```

{%- if include.upgradeFrom == "OpenSource" %}

1. Install your pull secret.
   ```bash
   kubectl create secret generic tigera-pull-secret \
       --from-file=.dockerconfigjson=<path/to/pull/secret> \
       --type=kubernetes.io/dockerconfigjson -n tigera-operator
   ```

{%- endif %}
{%- if include.upgradeFrom == "OpenSource" %}

1. Install the Tigera custom resources. For more information on configuration options available in this manifest, see [the installation reference]({{site.baseurl}}/reference/installation/api).
   ```bash
   {%- if include.provider == "EKS" %}
   kubectl apply -f {{ "/manifests/eks/custom-resources.yaml" | absolute_url }}
   {%- else %}
   kubectl apply -f {{ "/manifests/custom-resources.yaml" | absolute_url }}
   {%- endif %}
   ```

{%- endif %}
{%- if include.upgradeFrom != "OpenSource" %}

1. If your cluster is a management cluster, apply a [ManagementCluster]({{site.baseurl}}/reference/installation/api#operator.tigera.io/v1.ManagementCluster)
   CR to your cluster.
   ```bash
   kubectl apply -f - <<EOF
   apiVersion: operator.tigera.io/v1
   kind: ManagementCluster
   metadata:
     name: tigera-secure
   EOF
   ```

1. Install the new network policies to secure {{site.prodname}} component communications.

   If your cluster is a **managed** cluster, apply this manifest.
   
   ```bash
   kubectl apply -f {{ "/manifests/tigera-policies-managed.yaml" | absolute_url }}
   ```
   
   For other clusters, use this manifest.
   
   ```bash
   kubectl apply -f {{ "/manifests/tigera-policies.yaml" | absolute_url }}
   ```
   
1. You can monitor progress with the following command:
   ```bash
   watch kubectl get tigerastatus
   ```

    **Note**: If there are any problems you can use `kubectl get tigerastatus -o yaml` to get more details.
    {: .alert .alert-info}

1. Remove unused policies in your cluster.

   If your cluster is a **managed** cluster, run this command:

   ```bash
   kubectl delete -f {{ "/manifests/default-tier-policies-managed.yaml" | absolute_url }}
   ```

   For other clusters, run this command:

   ```bash
   kubectl delete -f {{ "/manifests/default-tier-policies.yaml" | absolute_url }}
   ```
{% endif %}
