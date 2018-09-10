1. {{site.prodname}} can send logs to ElasticSearch to provide easy to use auditing and compliance
   reports and enable certain UI functions.  Using this feature requires an ElasticSearch cluster, configured with suitable storage.
   {{site.prodname}} includes the [ElasticSearch operator](https://github.com/upmc-enterprises/elasticsearch-operator),
   which deploys and manages an ElasticSearch cluster inside Kubernetes / OpenShift for you.

   If you don't want to deploy ElasticSearch inside your orchestrator or have an existing cluster, 
   see the [documentation for using your own ElasticSearch]({{site.url}}/{{page.version}}/usage/logs/byo-elastic) and skip the next step.

1. The bundled ElasticSearch operator is configured to use an `elasticsearch-storage` `StorageClass`.
   We provide three implementations ([notes]({{site.baseurl}}/{{page.version}}/usage/logs/configure-elastic#managing-storage-for-operator-created-clusters)):
   `local` (single node only: uses `/var/tigera/elastic-data`), `aws` (uses AWS EBS)
   and `gce` (uses GCE persistent disks); but you can also provide your own implementation: skip to the next step if so.
   
   Download the appropriate manifest: [local]({{site.baseurl}}/{{page.version}}/getting-started/kubernetes/installation/hosted/cnx/1.7/elastic-storage-local.yaml){:target="_blank"},
   [aws]({{site.baseurl}}/{{page.version}}/getting-started/kubernetes/installation/hosted/cnx/1.7/elastic-storage-aws.yaml){:target="_blank"} or
   [gce]({{site.baseurl}}/{{page.version}}/getting-started/kubernetes/installation/hosted/cnx/1.7/elastic-storage-gce.yaml){:target="_blank"}

   ```bash
   curl \
   {{site.baseurl}}/{{page.version}}/getting-started/kubernetes/installation/hosted/cnx/1.7/elastic-storage-aws.yaml \
   -O
   kubectl apply -f elastic-storage-*.yaml
   ```

   For AWS or GCE you'll need to have configured the appropriate cloud provider integration to allow Kubernetes to provision the volumes.

1. To use a different StorageClass (e.g. Ceph RBD or NFS), create a `StorageClass` called `elasticsearch-storage`.
   The [Kubernetes documentation on StorageClasses](https://kubernetes.io/docs/concepts/storage/storage-classes/#provisioner)
   has a list of possible options and links to sample manifests.  The
   ones with a tick under internal provisioner can be used without further
   Kubernetes configuration: the others require a provisioner to be set up.

   ```
   kubectl apply -f my-storage-class.yaml
   ```