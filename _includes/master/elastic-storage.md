{% if include.orch != "openshift" %}
  {% assign cli = "kubectl" %}
{% else %}
  {% assign cli = "oc" %}
{% endif %}

1. The bundled Elasticsearch operator is configured to use a `StorageClass` called `elasticsearch-storage` and local storage.
   Use the following command to apply the manifest.

   ```bash
   {{cli}} apply -f \
   {{site.url}}/{{page.version}}/getting-started/kubernetes/installation/hosted/cnx/1.7/elastic-storage-local.yaml
   ```

   > **Note**: You can also
   > [view the manifest in a new tab]({{site.url}}/{{page.version}}/getting-started/kubernetes/installation/hosted/cnx/1.7/elastic-storage-local.yaml){:target="_blank"}.
   {: .alert .alert-info}

   > **Tip**: To use storage other than local, refer to the
   > [Kubernetes documentation on StorageClasses](https://kubernetes.io/docs/concepts/storage/storage-classes/#provisioner)
   > for a list of alternatives and some sample manifests. You may need to configure
   > a provisioner or cloud provider integration. Edit the manifest above or create a new
   > one with a `StorageClass` called `elasticsearch-storage` and the other necessary details.
   > Then apply it. `{{cli}} apply -f my-storage-class.yaml`.
   {: .alert .alert-success}
