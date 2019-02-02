---
title: Configuring Elasticsearch
redirect_from: latest/usage/logs/configure-elastic
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

## Configuring the operator created cluster

An `elasticsearchcluster` resource is defined in [monitor-calico.yaml](../../getting-started/kubernetes/installation/hosted/cnx/1.7/monitor-calico.yaml).
The parameters of this cluster can be altered to meet the needs of your deployment - they're described
in the [ElasticSearch operator documentation](https://github.com/upmc-enterprises/elasticsearch-operator).

## Managing storage for operator created clusters

The operator created cluster stores its data in volumes created through
a Kubernetes `StorageClass`.  Three options are bundled with {{site.prodname}} -
in manifests called `elastic-storage*.yaml`.  The installation options cover
how to get these manifests - a few notes on each option follow.

#### Local

The `local` implementation creates two volumes using the host filesystem.
Those volumes are stored in `/var/tigera/elastic-data`, which must be writable
by Kubernetes.

The [elastic-storage-local.yaml](../../getting-started/kubernetes/installation/hosted/cnx/1.7/elastic-storage-local.yaml)
manifest creates two of these volumes by default - enough for the default
1 master, 1 client, 1 data cluster set up.  If you wish to modify the size of
the volumes, they must match the configured request in the `elasticsearchcluster`
CRD exactly.

> **Warning**: The `local` `StorageClass` only works on single node clusters -
> choose another implementation for production use.
{: .alert .alert-danger}
