Download the {{site.prodname}} manifests for OpenShift and add them to the generated manifests directory:

```bash
curl {{ "/manifests/ocp/crds/01-crd-apiserver.yaml" | absolute_url }} -o manifests/01-crd-apiserver.yaml
curl {{ "/manifests/ocp/crds/01-crd-authentication.yaml" | absolute_url }} -o manifests/01-crd-authentication.yaml
curl {{ "/manifests/ocp/crds/01-crd-compliance.yaml" | absolute_url }} -o manifests/01-crd-compliance.yaml
curl {{ "/manifests/ocp/crds/01-crd-manager.yaml" | absolute_url }} -o manifests/01-crd-manager.yaml
curl {{ "/manifests/ocp/crds/01-crd-eck-apmserver.yaml" | absolute_url }} -o manifests/01-crd-eck-apmserver.yaml
curl {{ "/manifests/ocp/crds/01-crd-eck-elasticsearch.yaml" | absolute_url }} -o manifests/01-crd-eck-elasticsearch.yaml
curl {{ "/manifests/ocp/crds/01-crd-eck-kibana.yaml" | absolute_url }} -o manifests/01-crd-eck-kibana.yaml
curl {{ "/manifests/ocp/crds/01-crd-eck-trustrelationship.yaml" | absolute_url }} -o manifests/01-crd-eck-trustrelationship.yaml
curl {{ "/manifests/ocp/crds/01-crd-eck-beat.yaml" | absolute_url }} -o manifests/01-crd-eck-beat.yaml
curl {{ "/manifests/ocp/crds/01-crd-eck-enterprisesearch.yaml" | absolute_url }} -o manifests/01-crd-eck-enterprisesearch.yaml
curl {{ "/manifests/ocp/crds/01-crd-imageset.yaml" | absolute_url }} -o manifests/01-crd-imageset.yaml
curl {{ "/manifests/ocp/crds/01-crd-installation.yaml" | absolute_url }} -o manifests/01-crd-installation.yaml
curl {{ "/manifests/ocp/crds/01-crd-intrusiondetection.yaml" | absolute_url }} -o manifests/01-crd-intrusiondetection.yaml
curl {{ "/manifests/ocp/crds/01-crd-logstorage.yaml" | absolute_url }} -o manifests/01-crd-logstorage.yaml
curl {{ "/manifests/ocp/crds/01-crd-logcollector.yaml" | absolute_url }} -o manifests/01-crd-logcollector.yaml
curl {{ "/manifests/ocp/crds/01-crd-monitor.yaml" | absolute_url }} -o manifests/01-crd-monitor.yaml
curl {{ "/manifests/ocp/crds/01-crd-tigerastatus.yaml" | absolute_url }} -o manifests/01-crd-tigerastatus.yaml
curl {{ "/manifests/ocp/crds/01-crd-managementclusterconnection.yaml" | absolute_url }} -o manifests/01-crd-managementclusterconnection.yaml
curl {{ "/manifests/ocp/crds/01-crd-managementcluster.yaml" | absolute_url }} -o manifests/01-crd-managementcluster.yaml
{%- for data in site.static_files %}
{%- if data.path contains '/manifests/ocp/crds/calico' %}
curl {{ data.path | absolute_url }} -o manifests/{{data.name}}
{%- endif -%}
{% endfor %}
curl {{ "/manifests/ocp/tigera-operator/00-namespace-tigera-operator.yaml" | absolute_url }} -o manifests/00-namespace-tigera-operator.yaml
curl {{ "/manifests/ocp/tigera-operator/02-rolebinding-tigera-operator.yaml" | absolute_url }} -o manifests/02-rolebinding-tigera-operator.yaml
curl {{ "/manifests/ocp/tigera-operator/02-role-tigera-operator.yaml" | absolute_url }} -o manifests/02-role-tigera-operator.yaml
curl {{ "/manifests/ocp/tigera-operator/02-serviceaccount-tigera-operator.yaml" | absolute_url }} -o manifests/02-serviceaccount-tigera-operator.yaml
curl {{ "/manifests/ocp/tigera-operator/02-configmap-calico-resources.yaml" | absolute_url }} -o manifests/02-configmap-calico-resources.yaml
curl {{ "/manifests/ocp/tigera-operator/02-tigera-operator.yaml" | absolute_url }} -o manifests/02-tigera-operator.yaml
{%- unless page.openshift_manifests_ignore_pullsecret %}
curl {{ "/manifests/ocp/02-pull-secret.yaml" | absolute_url }} -o manifests/02-pull-secret.yaml
{%- endunless %}
{%- unless page.openshift_manifests_ignore_installation_cr %}
curl {{ "/manifests/ocp/01-cr-installation.yaml" | absolute_url }} -o manifests/01-cr-installation.yaml
{%- endunless %}
{%- unless page.openshift_manifests_ignore_apiserver_cr %}
curl {{ "/manifests/ocp/01-cr-apiserver.yaml" | absolute_url }} -o manifests/01-cr-apiserver.yaml
{%- endunless %}
```
{% unless page.openshift_manifests_ignore_installation_cr %}
> **Note**: Read more about customizing the file `manifests/01-cr-installation.yaml` in the [Installation API Reference]({{site.baseurl}}/reference/installation/api#operator.tigera.io/v1.Installation)
{: .alert .alert-info}
{% endunless %}
