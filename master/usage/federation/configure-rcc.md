---
title: Configuring remote clusters
---

## About configuring remote clusters

To create policies that reference endpoints across multiple clusters, you must allow the clusters to obtain endpoint information from
each other by [adding Remote Cluster Configuration resources](#adding-a-remote-cluster-configuration-resource).

For example, if you have two clusters and you want to create policies that reference endpoints on both, you would:

- Add a Remote Cluster Configuration resource to cluster A that allows it to obtain endpoint information from cluster B.
- Add a Remote Cluster Configuration resource to cluster B that allows it to obtain endpoint information from cluster A.

You can add multiple Remote Cluster Configuration resources to a cluster, allowing it to obtain endpoint information
from more than one cluster.

In addition to adding the necessary Remote Cluster Configuration resources, you may need to
[modify your IP Pool configuration](#configuring-ip-pool-resources-for-federated-endpoint-identity).

## Adding a Remote Cluster Configuration resource

### Prerequisite

Before you can add a Remote Cluster Configuration resource to a cluster, you must
install {{site.prodname}} on the cluster, following the procedure appropriate to the
cluster's datastore type.
- [etcd](../../getting-started/kubernetes/installation/calico#installing-with-federation-using-etcd)
- [Kubernetes API datastore](../../getting-started/kubernetes/installation/calico#installing-with-federation-using-kubernetes-api-datastore)

### About adding a Remote Cluster Configuration resource

Each instance of the [Remote Cluster Configuration](/{{page.version}}/reference/calicoctl/resources/remoteclusterconfiguration) resource represents a single remote cluster from which the local cluster can retrieve endpoint
information. The local cluster can talk to multiple remote clusters.

The resource definition varies according to your datastore type. Refer to the section that corresponds to your datastore
type for instructions.
- [etcd](#etcd-datastore)
- [Kubernetes API datastore](#kubernetes-api-datastore)

### Adding a Remote Cluster Configuration resource with the etcd datastore

If the remote cluster uses etcd as the {{site.prodname}} datastore, set the `datastoreType` in the RemoteClusterConfiguration
to `etcdv3` and populate the `etcd*` fields. You must also fill in either the `kubeconfig` or the `k8s*` fields.

As long as you followed the installation instructions, the files in the
[`tigera-federation-remotecluster` secret created during installation](/{{page.version}}/getting-started/kubernetes/installation/calico#installing-with-federation-using-etcd)
will appear in the Typha pod in the `/etc/tigera-federation-remotecluster` directory and the [RemoteClusterConfiguration](/{{page.version}}/reference/calicoctl/resources/remoteclusterconfiguration)
can reference the files using this path.

An example Remote Configuration Resource for etcd follows.

```yaml
apiVersion: projectcalico.org/v3
kind: RemoteClusterConfiguration
metadata:
  name: cluster-n
spec:
  datastoreType: etcdv3
  etcdEndpoints: "https://10.0.0.1:2379,https://10.0.0.2:2379"
  # Include the remote-cluster kubeconfig for the federated services controller
  kubeconfig: /etc/tigera-federation-remotecluster/kubeconfig-rem-cluster-n
```

### Remote Cluster Configuration resource with the Kubernetes API datastore

If the remote cluster uses the Kubernetes API datastore for {{site.prodname}} data,
set the `datastoreType` in the `RemoteClusterConfiguration`
to `kubernetes` and populate the `kubeconfig` or `k8s*` fields.

As long as you followed the installation instructions, the files in the
[`tigera-federation-remotecluster` secret created during installation]({{page.version}}/getting-started/kubernetes/installation/calico#installing-with-federation-using-kubernetes-api-datastore)
will appear in the Typha pod in the `/etc/tigera-federation-remotecluster` directory and the [RemoteClusterConfiguration](/{{page.version}}/reference/calicoctl/resources/remoteclusterconfiguration)
can reference the files using this path.

An example Remote Configuration Resource for the Kubernetes API datastore follows.

```yaml
apiVersion: projectcalico.org/v3
kind: RemoteClusterConfiguration
metadata:
  name: cluster-n
spec:
  datastoreType: kubernetes
  kubeconfig: /etc/tigera-federation-remotecluster/kubeconfig-rem-cluster-n
```

## Configuring IP Pool resources for federated endpoint identity

If your local cluster has NATOutgoing configured on your IP Pools, it is necessary to configure IP Pools covering the IP ranges
of your remote clusters. This ensures outgoing NAT is not performed on packets bound for the remote clusters. These additional
IP Pools should have `disabled` set to `true` to ensure the pools are not used for IP assignment on the local cluster.

The IP Pool CIDR used for pod IP allocation should not overlap with any of the IP ranges used by the pods and nodes of any
other federated cluster.

For example, you may configure the following on your local cluster, referring to the `IPPool` on a remote cluster:

```yaml
apiVersion: projectcalico.org/v3
kind: IPPool
metadata:
  name: cluster1-main-pool
spec:
  cidr: 192.168.0.0/18
  disabled: true
```
