---
title: Configuring access to remote clusters
redirect_from: latest/networking/federation/configure-rcc
canonical_url: https://docs.tigera.io/v2.3/usage/federation/configure-rcc
---

## About configuring access to remote clusters

To allow a local cluster to pull endpoint data from a remote cluster, you must add a Remote Cluster
Configuration resource to the local cluster.

For example, let's imagine that you want to federate three clusters named `cluster-a`, `cluster-b`,
and `cluster-c`.

After installing {{site.prodname}} on each of the clusters:

- Add two Remote Cluster Configuration resources to `cluster-a`: one for `cluster-b` and another
  for `cluster-c`.

- Add two Remote Cluster Configuration resources to `cluster-b`: one for `cluster-a` and another
  for `cluster-c`.

- Add two Remote Cluster Configuration resources to `cluster-c`: one for `cluster-a` and another
  for `cluster-b`.

In addition to adding the necessary Remote Cluster Configuration resources, you may need to
[modify the local cluster's IP pool configuration](#configuring-ip-pool-resources).

## Adding a Remote Cluster Configuration resource

### Prerequisite

Before you can add a Remote Cluster Configuration resource to a cluster, you must
install {{site.prodname}} on the cluster, following the procedure appropriate to the
cluster's datastore type.
- [etcd](/{{page.version}}/reference/other-install-methods/kubernetes/installation/calico#installing-with-federation-using-etcd)
- [Kubernetes API datastore](/{{page.version}}/reference/other-install-methods/kubernetes/installation/calico#installing-with-federation-using-kubernetes-api-datastore)

You will also need [calicoctl](/{{page.version}}/getting-started/calicoctl/install) installed. `RemoteClusterConfiguration` is created with calicoctl.

### About adding a Remote Cluster Configuration resource

Each instance of the [Remote Cluster Configuration](/{{page.version}}/reference/resources/remoteclusterconfiguration)
resource represents a single remote cluster from which the local cluster can retrieve endpoint information.

The resource definition varies according to your datastore type. Refer to the section that corresponds to your datastore
type for instructions.
- [etcd](#adding-a-remote-cluster-configuration-resource-with-the-etcd-datastore)
- [Kubernetes API datastore](#remote-cluster-configuration-resource-with-the-kubernetes-api-datastore)

### Adding a Remote Cluster Configuration resource with the etcd datastore

If the remote cluster uses etcd as the {{site.prodname}} datastore, set the `datastoreType` in the Remote Cluster Configuration
resource to `etcdv3` and populate the `etcd*` fields. You must also fill in either the `kubeconfig` or the `k8s*` fields.

As long as you followed the installation instructions, the files in the
[`tigera-federation-remotecluster` secret created during installation](/{{page.version}}/reference/other-install-methods/kubernetes/installation/calico#installing-with-federation-using-etcd)
will appear in the Typha pod in the `/etc/tigera-federation-remotecluster` directory and
the Remote Cluster Configuration resource can reference the files using this path.

An example Remote Cluster Configuration resource for etcd follows.

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
set the `datastoreType` in the Remote Cluster Configuration resource
to `kubernetes` and populate the `kubeconfig` or `k8s*` fields.

As long as you followed the installation instructions, the files in the
[`tigera-federation-remotecluster` secret created during installation](/{{page.version}}/reference/other-install-methods/kubernetes/installation/calico#installing-with-federation-using-kubernetes-api-datastore)
will appear in the Typha pod in the `/etc/tigera-federation-remotecluster` directory and the [RemoteClusterConfiguration](/{{page.version}}/reference/resources/remoteclusterconfiguration)
can reference the files using this path.

An example Remote Cluster Configuration resource for the Kubernetes API datastore follows.

```yaml
apiVersion: projectcalico.org/v3
kind: RemoteClusterConfiguration
metadata:
  name: cluster-n
spec:
  datastoreType: kubernetes
  kubeconfig: /etc/tigera-federation-remotecluster/kubeconfig-rem-cluster-n
```

## Configuring IP pool resources

If your local cluster has `NATOutgoing` configured on your IP pools, you must to configure IP pools covering the IP ranges
of your remote clusters. This ensures that outgoing NAT is not performed on packets bound for the remote clusters. These additional
IP pools should have `disabled` set to `true` to ensure the pools are not used for IP assignment on the local cluster.

The IP pool CIDR used for pod IP allocation should not overlap with any of the IP ranges used by the pods and nodes of any
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
