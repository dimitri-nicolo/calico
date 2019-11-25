---
title: Configuring Elasticsearch
canonical_url: https://docs.tigera.io/v2.3/usage/logs/configure-elastic
---

{{site.prodname}} uses ElasticSearch to store and manage certain logs.

## RBAC for access to Elasticsearch stored logs

Access to Elasticsearch is currently provided through the [kube-apiserver-proxy](https://kubernetes.io/docs/tasks/access-application-cluster/access-cluster/#discovering-builtin-services).
Login is not supported yet, but access to the service is be restricted through
a combination of NetworkPolicy and Kubernetes RBAC.

- NetworkPolicy restricting access to Elasticsearch is provided to allow access
  only from the Kubernetes API Server.  This policy
  can be refined by editing your copy of the [cnx-policy.yaml](../../getting-started/kubernetes/installation/hosted/cnx/1.7/cnx-policy.yaml) manifest.

- The Kubernetes API Server proxy supports limited RBAC to restrict access
  based on the API path.  The [sample RBAC configuration](../../reference/cnx/rbac-tiered-policies)
  grants the required access for web interface users.

## Customizing the index names when sharing Elasticsearch clusters

The index names include a configurable cluster name.  The default value is `cluster`.  This should be customized
if you're using multiple Kubernetes or OpenShift clusters with a single shared Elasticsearch cluster.

The index names used are:
- `tigera_secure_ee_flows.<cluster name>.<date>`
- `tigera_secure_ee_audit_ee.<cluster name>.<date>`
- `tigera_secure_ee_audit_kube.<cluster name>.<date>`
- `tigera_secure_ee_dns.<cluster name>.<date>`

Edit the cluster name by following this procedure.  The two values must match for
   {{site.prodname}} Manager to be able to read the correct logs from Elasticsearch.

1. Edit the `tigera.cnx-manager.cluster-name` field in the `tigera-cnx-manager-config` ConfigMap.
   This ConfigMap can be found in `cnx.yaml`.

1. Edit the `tigera.elasticsearch.cluster-name` field in `tigera-es-config` ConfigMap.  This ConfigMap
   can be found in the `monitor-calico.yaml`.

## Configuring retention periods for Elasticsearch logs

{{site.prodname}} includes a [Curator](https://www.elastic.co/guide/en/elasticsearch/client/curator/current/about.html)
deployment that deletes old indices.  By default flow logs are removed after 7 days,
audit logs after 90 days, snapshots after 90 days, and compliance reports
after 90 days.

These time periods can be configured by editing the `tigera-es-config` ConfigMap,
found in `monitor-calico.yaml`.

If you don't wish to run Curator, you can delete the `tigera-es-curator` Deployment
from `monitor-calico.yaml`.

## Configuring the operator created cluster

An `elasticsearchcluster` resource is defined in [monitor-calico.yaml](../../getting-started/kubernetes/installation/hosted/cnx/1.7/monitor-calico.yaml).
The parameters of this cluster can be altered to meet the needs of your deployment - they're described
in the [ElasticSearch operator documentation](https://github.com/upmc-enterprises/elasticsearch-operator).

## Managing storage for operator created clusters

The operator created cluster stores its data in volumes created through
a Kubernetes `StorageClass`.  It is your responsibility to make a suitable StorageClass available.
The cluster is configured to use a StorageClass called `tigera-elasticsearch`, but this name can be modified
to refer to any suitable StorageClass you have set up.

#### Local

[Local volumes](https://kubernetes.io/docs/concepts/storage/volumes/#local) allow ElasticSearch to use
high performance local drives for storage.

#### Other remote storage

{{site.prodname}} is also tested using AWS EBS / GCE PD off-instance storage for ElasticSearch.
You should be able to use any StorageClass that offers performance comparable to the SSD variants
of those products, although note that local storage will still offer the best performance.
