Download the {{site.prodname}} manifests for OpenShift and add them to the generated manifests directory:

```bash
curl {{ "/manifests/ocp/crds/01-crd-alertmanager.yaml" | absolute_url }} -o manifests/01-crd-alertmanager.yaml
curl {{ "/manifests/ocp/crds/01-crd-apiserver.yaml" | absolute_url }} -o manifests/01-crd-apiserver.yaml
curl {{ "/manifests/ocp/crds/01-crd-compliance.yaml" | absolute_url }} -o manifests/01-crd-compliance.yaml
curl {{ "/manifests/ocp/crds/01-crd-manager.yaml" | absolute_url }} -o manifests/01-crd-manager.yaml
curl {{ "/manifests/ocp/crds/01-crd-eck-apmserver.yaml" | absolute_url }} -o manifests/01-crd-eck-apmserver.yaml
curl {{ "/manifests/ocp/crds/01-crd-eck-elasticsearch.yaml" | absolute_url }} -o manifests/01-crd-eck-elasticsearch.yaml
curl {{ "/manifests/ocp/crds/01-crd-eck-kibana.yaml" | absolute_url }} -o manifests/01-crd-eck-kibana.yaml
curl {{ "/manifests/ocp/crds/01-crd-eck-trustrelationship.yaml" | absolute_url }} -o manifests/01-crd-eck-trustrelationship.yaml
curl {{ "/manifests/ocp/crds/01-crd-installation.yaml" | absolute_url }} -o manifests/01-crd-installation.yaml
curl {{ "/manifests/ocp/crds/01-crd-intrusiondetection.yaml" | absolute_url }} -o manifests/01-crd-intrusiondetection.yaml
curl {{ "/manifests/ocp/crds/01-crd-logstorage.yaml" | absolute_url }} -o manifests/01-crd-logstorage.yaml
curl {{ "/manifests/ocp/crds/01-crd-logcollector.yaml" | absolute_url }} -o manifests/01-crd-logcollector.yaml
curl {{ "/manifests/ocp/crds/01-crd-prometheusrule.yaml" | absolute_url }} -o manifests/01-crd-prometheusrule.yaml
curl {{ "/manifests/ocp/crds/01-crd-prometheus.yaml" | absolute_url }} -o manifests/01-crd-prometheus.yaml
curl {{ "/manifests/ocp/crds/01-crd-servicemonitor.yaml" | absolute_url }} -o manifests/01-crd-servicemonitor.yaml
curl {{ "/manifests/ocp/crds/01-crd-tigerastatus.yaml" | absolute_url }} -o manifests/01-crd-tigerastatus.yaml
curl {{ "/manifests/ocp/crds/01-crd-managementclusterconnection.yaml" | absolute_url }} -o manifests/01-crd-managementclusterconnection.yaml
curl {{ "/manifests/ocp/tigera-operator/00-namespace-tigera-operator.yaml" | absolute_url }} -o manifests/00-namespace-tigera-operator.yaml
curl {{ "/manifests/ocp/tigera-operator/02-rolebinding-tigera-operator.yaml" | absolute_url }} -o manifests/02-rolebinding-tigera-operator.yaml
curl {{ "/manifests/ocp/tigera-operator/02-role-tigera-operator.yaml" | absolute_url }} -o manifests/02-role-tigera-operator.yaml
curl {{ "/manifests/ocp/tigera-operator/02-serviceaccount-tigera-operator.yaml" | absolute_url }} -o manifests/02-serviceaccount-tigera-operator.yaml
curl {{ "/manifests/ocp/tigera-operator/02-tigera-operator.yaml" | absolute_url }} -o manifests/02-tigera-operator.yaml
curl {{ "/manifests/ocp/misc/00-namespace-tigera-prometheus.yaml" | absolute_url }} -o manifests/00-namespace-tigera-prometheus.yaml
curl {{ "/manifests/ocp/prometheus-operator/04-clusterrolebinding-prometheus-operator.yaml" | absolute_url }} -o manifests/04-clusterrolebinding-prometheus-operator.yaml
curl {{ "/manifests/ocp/prometheus-operator/04-clusterrolebinding-prometheus.yaml" | absolute_url }} -o manifests/04-clusterrolebinding-prometheus.yaml
curl {{ "/manifests/ocp/prometheus-operator/04-clusterrole-prometheus-operator.yaml" | absolute_url }} -o manifests/04-clusterrole-prometheus-operator.yaml
curl {{ "/manifests/ocp/prometheus-operator/04-clusterrole-prometheus.yaml" | absolute_url }} -o manifests/04-clusterrole-prometheus.yaml
curl {{ "/manifests/ocp/prometheus-operator/04-deployment-prometheus-operator.yaml" | absolute_url }} -o manifests/04-deployment-prometheus-operator.yaml
curl {{ "/manifests/ocp/prometheus-operator/04-serviceaccount-prometheus-operator.yaml" | absolute_url }} -o manifests/04-serviceaccount-prometheus-operator.yaml
curl {{ "/manifests/ocp/prometheus-operator/04-serviceaccount-prometheus.yaml" | absolute_url }} -o manifests/04-serviceaccount-prometheus.yaml
curl {{ "/manifests/ocp/misc/99-alertmanager-secret.yaml" | absolute_url }} -o manifests/99-alertmanager-secret.yaml
curl {{ "/manifests/ocp/misc/99-alertmanager-service.yaml" | absolute_url }} -o manifests/99-alertmanager-service.yaml
curl {{ "/manifests/ocp/misc/99-prometheus-service.yaml" | absolute_url }} -o manifests/99-prometheus-service.yaml
curl {{ "/manifests/ocp/01-cr-installation.yaml" | absolute_url }} -o manifests/01-cr-installation.yaml
curl {{ "/manifests/ocp/01-cr-apiserver.yaml" | absolute_url }} -o manifests/01-cr-apiserver.yaml
curl {{ "/manifests/ocp/01-cr-manager.yaml" | absolute_url }} -o manifests/01-cr-manager.yaml
curl {{ "/manifests/ocp/01-cr-compliance.yaml" | absolute_url }} -o manifests/01-cr-compliance.yaml
curl {{ "/manifests/ocp/01-cr-intrusiondetection.yaml" | absolute_url }} -o manifests/01-cr-intrusiondetection.yaml
curl {{ "/manifests/ocp/01-cr-alertmanager.yaml" | absolute_url }} -o manifests/01-cr-alertmanager.yaml
curl {{ "/manifests/ocp/01-cr-logstorage.yaml" | absolute_url }} -o manifests/01-cr-logstorage.yaml
curl {{ "/manifests/ocp/01-cr-logcollector.yaml" | absolute_url }} -o manifests/01-cr-logcollector.yaml
curl {{ "/manifests/ocp/01-cr-prometheus.yaml" | absolute_url }} -o manifests/01-cr-prometheus.yaml
curl {{ "/manifests/ocp/01-cr-prometheusrule.yaml" | absolute_url }} -o manifests/01-cr-prometheusrule.yaml
curl {{ "/manifests/ocp/01-cr-servicemonitor.yaml" | absolute_url }} -o manifests/01-cr-servicemonitor.yaml
```

> **Note**: The Tigera operator manifest downloaded above includes an initialization container which configures Amazon AWS
> security groups for {{site.prodname}}. If not running on AWS, you should remove the init container from `manifests/02-tigera-operator.yaml`.
{: .alert .alert-info}
