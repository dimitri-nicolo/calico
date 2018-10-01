{% if include.orch != "openshift" %}
  {% assign cli = "kubectl" %}
{% else %} 
  {% assign cli = "oc" %}
{% endif %}

1. {{site.prodname}} can send logs to Elasticsearch to provide easy to use auditing and compliance
   reports and enable certain UI functions.  Using this feature requires an Elasticsearch cluster, configured with suitable storage.
   {{site.prodname}} includes the [Elasticsearch operator](https://github.com/upmc-enterprises/elasticsearch-operator),
   which deploys and manages an Elasticsearch cluster inside Kubernetes / OpenShift for you.

1. The bundled Elasticsearch operator is configured to use an `elasticsearch-storage` `StorageClass` and local storage.
   If your cluster is connected to the internet, use the following command to apply the manifest.

   ```bash
   {{cli}} apply -f {{site.url}}/{{page.version}}/getting-started/kubernetes/installation/hosted/cnx/1.7/elastic-storage-local.yaml
   ```

   > **Note**: You can also
   > [view the manifest in a new tab]({{site.url}}/{{page.version}}/getting-started/kubernetes/installation/hosted/cnx/1.7/elastic-storage-local.yaml){:target="_blank"}.
   {: .alert .alert-info}

   > For offline installs, complete the following steps instead.
   >
   > 1. Download the Elasticsearch local storage manifest.
   >
   >    ```bash
   >    curl --compressed -O \
   >    {{site.url}}/{{page.version}}/getting-started/kubernetes/installation/hosted/cnx/1.7/elastic-storage-local.yaml
   >    ```
   >
   > 1. Apply the manifest.
   >    
   >    ```bash
   >    {{cli}} apply -f elastic-storage-local.yaml
   >    ```
   >

1. To use a different StorageClass (e.g. Ceph RBD or NFS), create a `StorageClass` called `elasticsearch-storage`.
   The [Kubernetes documentation on StorageClasses](https://kubernetes.io/docs/concepts/storage/storage-classes/#provisioner)
   has a list of possible options and links to sample manifests.  You may need to configure
   a provisioner or cloud provider integration.

   ```
   {{cli}} apply -f my-storage-class.yaml
   ```
