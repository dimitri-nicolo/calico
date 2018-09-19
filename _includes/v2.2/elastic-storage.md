{% if include.orch != "openshift" %}
  {% assign cli = "kubectl" %}
{% else %} 
  {% assign cli = "oc" %}
{% endif %}

1. {{site.prodname}} can send logs to Elasticsearch to provide easy to use auditing and compliance
   reports and enable certain UI functions.  Using this feature requires an Elasticsearch cluster, configured with suitable storage.
   {{site.prodname}} includes the [Elasticsearch operator](https://github.com/upmc-enterprises/elasticsearch-operator),
   which deploys and manages an Elasticsearch cluster inside Kubernetes / OpenShift for you.

   If you don't want to deploy Elasticsearch inside your orchestrator or have an existing cluster,
   see the [documentation for using your own Elasticsearch]({{site.url}}/{{page.version}}/usage/logs/byo-elastic) and skip the next step.

1. The bundled Elasticsearch operator is configured to use an `elasticsearch-storage` `StorageClass` and local storage.
   Use the following command to apply the manifest.

   ```bash
   {{cli}} apply -f elastic-storage-local.yaml \
   {{site.url}}/{{page.version}}/getting-started/kubernetes/installation/hosted/cnx/1.7/elastic-storage-local.yaml
   ```

1. To use a different StorageClass (e.g. Ceph RBD or NFS), create a `StorageClass` called `elasticsearch-storage`.
   The [Kubernetes documentation on StorageClasses](https://kubernetes.io/docs/concepts/storage/storage-classes/#provisioner)
   has a list of possible options and links to sample manifests.  The
   ones with a tick under internal provisioner can be used without further
   Kubernetes configuration: the others require a provisioner to be set up.

   ```
   {{cli}} apply -f my-storage-class.yaml
   ```
