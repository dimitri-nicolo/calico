(Optional) Download the {{site.prodname}} manifests for OpenShift specific to Prometheus operator and add them to the generated manifests directory:
```bash
curl {{ "/manifests/ocp/crds/01-crd-alertmanager.yaml" | absolute_url }} -o manifests/01-crd-alertmanager.yaml
curl {{ "/manifests/ocp/crds/01-crd-prometheusrule.yaml" | absolute_url }} -o manifests/01-crd-prometheusrule.yaml
curl {{ "/manifests/ocp/crds/01-crd-thanosrulers.yaml" | absolute_url }} -o manifests/01-crd-thanosrulers.yaml
curl {{ "/manifests/ocp/crds/01-crd-prometheus.yaml" | absolute_url }} -o manifests/01-crd-prometheus.yaml
curl {{ "/manifests/ocp/crds/01-crd-servicemonitor.yaml" | absolute_url }} -o manifests/01-crd-servicemonitor.yaml
curl {{ "/manifests/ocp/prometheus-operator/04-clusterrolebinding-prometheus-operator.yaml" | absolute_url }} -o manifests/04-clusterrolebinding-prometheus-operator.yaml
curl {{ "/manifests/ocp/prometheus-operator/04-clusterrole-prometheus-operator.yaml" | absolute_url }} -o manifests/04-clusterrole-prometheus-operator.yaml
curl {{ "/manifests/ocp/prometheus-operator/04-deployment-prometheus-operator.yaml" | absolute_url }} -o manifests/04-deployment-prometheus-operator.yaml
curl {{ "/manifests/ocp/prometheus-operator/04-serviceaccount-prometheus-operator.yaml" | absolute_url }} -o manifests/04-serviceaccount-prometheus-operator.yaml
```
