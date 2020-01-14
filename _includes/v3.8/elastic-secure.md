{% if include.orch != "openshift" %}
  {% assign cli = "kubectl" %}
{% else %}
  {% assign cli = "oc" %}
{% endif %}

1. Set up secret with username and password for Fluentd to authenticate with Elasticsearch.
   Replace `<fluentd-elasticsearch-password>` with the password.
   ```
   {{cli}} create secret generic elastic-fluentd-user \
   --from-literal=username=tigera-ee-fluentd \
   --from-literal=password=<fluentd-elasticsearch-password> \
   -n calico-monitoring
   ```

1. Set up secret with username and password for Curator to authenticate with Elasticsearch.
   Replace `<curator-elasticsearch-password>` with the password.
   ```
   {{cli}} create secret generic elastic-curator-user \
   --from-literal=username=tigera-ee-curator \
   --from-literal=password=<curator-elasticsearch-password> \
   -n calico-monitoring
   ```

1. Set up secret with username and password for {{site.tseeprodname}} intrusion detection controller to authenticate with Elasticsearch.
   Replace `<intrusion-detection-password>` with the password.
   ```
   {{cli}} create secret generic elastic-ee-intrusion-detection \
   --from-literal=username=tigera-ee-intrusion-detection \
   --from-literal=password=<intrusion-detection-password> \
   -n calico-monitoring
   ```


1. Set up secret with username and passwords for {{site.tseeprodname}} compliance report and dashboard to authenticate with Elasticsearch.
   Replace `<compliance-benchmarker-elasticsearch-password>`, `<compliance-controller-elasticsearch-password>`, `<compliance-reporter-elasticsearch-password>`,
   `<compliance-snapshotter-elasticsearch-password>` and `<compliance-server-elasticsearch-password>` with the appropriate passwords.
   ```
   {{cli}} create secret generic elastic-compliance-user \
   --from-literal=benchmarker.username=tigera-ee-compliance-benchmarker \
   --from-literal=benchmarker.password=<compliance-benchmarker-elasticsearch-password> \
   --from-literal=controller.username=tigera-ee-compliance-controller \
   --from-literal=controller.password=<compliance-controller-elasticsearch-password> \
   --from-literal=reporter.username=tigera-ee-compliance-reporter \
   --from-literal=reporter.password=<compliance-reporter-elasticsearch-password> \
   --from-literal=snapshotter.username=tigera-ee-compliance-snapshotter \
   --from-literal=snapshotter.password=<compliance-snapshotter-elasticsearch-password> \
   --from-literal=server.username=tigera-ee-compliance-server \
   --from-literal=server.password=<compliance-server-elasticsearch-password> \
   -n calico-monitoring
   ```

1. Set up secret with username and password for the {{site.tseeprodname}} job installer to authenticate with Elasticsearch.
   Replace `<installer-password>` with the password.
   ```
   {{cli}} create secret generic elastic-ee-installer \
   --from-literal=username=tigera-ee-installer \
   --from-literal=password=<installer-password> \
   -n calico-monitoring
   ```

1. Create a Secret in the calico-monitoring namespace containing
   * certificate authority certificate to authenticate Elasticsearch backend
   * Username and password for the tigera-ee-manager to authenticate with Elasticsearch

   Replace `<ee-manager-elasticsearch-password>` with the password.

   ```
   {{cli}} create secret generic tigera-es-config \
     --from-file=tigera.elasticsearch.ca=ElasticSearchCA.pem \
     --from-literal=tigera.elasticsearch.username=tigera-ee-manager \
     --from-literal=tigera.elasticsearch.password=<ee-manager-elasticsearch-password> \
     -n calico-monitoring
   ```
