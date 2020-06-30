Download the {{site.prodname}} custom resources for OpenShift and add them to the manifests directory:

{% if page.cr_directory %}
    {% assign manifests_dir = page.cr_directory %}
{% else %}
    {% assign manifests_dir = "manifests" %}
{% endif %}

```bash
{% unless page.ignore_installation_cr == true %}curl {{ "/manifests/ocp/01-cr-installation.yaml" | absolute_url }} -o {{manifests_dir}}/01-cr-installation.yaml
{% endunless %}curl {{ "/manifests/ocp/01-cr-apiserver.yaml" | absolute_url }} -o {{manifests_dir}}/01-cr-apiserver.yaml
curl {{ "/manifests/ocp/01-cr-manager.yaml" | absolute_url }} -o {{manifests_dir}}/01-cr-manager.yaml
curl {{ "/manifests/ocp/01-cr-compliance.yaml" | absolute_url }} -o {{manifests_dir}}/01-cr-compliance.yaml
curl {{ "/manifests/ocp/01-cr-intrusiondetection.yaml" | absolute_url }} -o {{manifests_dir}}/01-cr-intrusiondetection.yaml
curl {{ "/manifests/ocp/01-cr-alertmanager.yaml" | absolute_url }} -o {{manifests_dir}}/01-cr-alertmanager.yaml
curl {{ "/manifests/ocp/01-cr-logstorage.yaml" | absolute_url }} -o {{manifests_dir}}/01-cr-logstorage.yaml
curl {{ "/manifests/ocp/01-cr-logcollector.yaml" | absolute_url }} -o {{manifests_dir}}/01-cr-logcollector.yaml
curl {{ "/manifests/ocp/01-cr-prometheus.yaml" | absolute_url }} -o {{manifests_dir}}/01-cr-prometheus.yaml
curl {{ "/manifests/ocp/01-cr-prometheusrule.yaml" | absolute_url }} -o {{manifests_dir}}/01-cr-prometheusrule.yaml
curl {{ "/manifests/ocp/01-cr-servicemonitor.yaml" | absolute_url }} -o {{manifests_dir}}/01-cr-servicemonitor.yaml
```
