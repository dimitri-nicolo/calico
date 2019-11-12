## Log storage requirements

{{site.prodname}} installs an Elasticsearch cluster for storing logs internally.

We use
[Elastic Cloud on Kubernetes](https://www.elastic.co/guide/en/cloud-on-k8s/current/k8s-overview.html)
to install and manage the cluster.  All you need to do is provide at least one node with suitable storage.

The cluster is configured to use a StorageClass called `tigera-elasticsearch`.  You must
set up this StorageClass before installing {{site.prodname}}.

We recommend using local disks for storage when possible (this offers the best performance),
but high performance remote storage can also be used.  Examples of suitable remote storage include
AWS SSD type EBS disks, or GCP PD-SSDs.

For information on how to configure storage, please consult the [Kubernetes](https://kubernetes.io/docs/concepts/storage/storage-classes/)
or [OpenShift](https://docs.openshift.com/container-platform/4.2/storage/understanding-persistent-storage.html) documentation.
- If you're going to use local disks, you may find the [sig-storage local static provisioner](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner)
  useful.  It can create and manage PersistentVolumes by watching for disks mounted in a certain directory.
- If you're planning on using cloud storage, ensure you've set up the cloud provider integration.

### Sizing

Many factors need to be considered when sizing this cluster:
- Scale, and nature of the traffic patterns in the cluster
- Retention periods for logs
- Aggregation and export interval configuration
- Desired replication factor for data
For tailored recommendations on sizing the cluster, please contact Tigera support.

#### Example test / development topology

1 Elasticsearch node, with 8GB RAM, 2 CPU cores and 200GB of storage.

#### Example production topology

3 Elasticsearch nodes, each with 96GB RAM, 8 CPU cores and 1TB of storage.
