---
title: Querying remote clusters with calicoq
canonical_url: https://docs.tigera.io/v2.3/usage/federation/calicoq
---

You can use `calicoq` to query endpoints in remote clusters. The output from `calicoq` will
include the name of the remote cluster (as configured in the Remote Cluster Configuration resource)
prepended to the name of any enumerated endpoint within the remote cluster.

An example command follows.

```bash
calicoq eval "all()"
```

If all remote clusters are accessible, `calicoq` should return something like the following.
In this example, the `remote-cluster-1` prefix indicates the remote cluster endpoints.

```bash
Endpoints matching selector all():
  Workload endpoint remote-cluster-1/host-1/k8s/kube-system.kube-dns-5fbcb4d67b-h6686/eth0
  Workload endpoint remote-cluster-1/host-2/k8s/kube-system.cnx-manager-66c4dbc5b7-6d9xv/eth0
  Workload endpoint host-a/k8s/kube-system.kube-dns-5fbcb4d67b-7wbhv/eth0
  Workload endpoint host-b/k8s/kube-system.cnx-manager-66c4dbc5b7-6ghsm/eth0
```
{: .no-select-button}

If a remote cluster is inaccessible, for example due to network failure or due to a misconfiguration,
the `calicoq` output includes details about the error. For example, the following output describes
an error indicating the kubeconfig file configured in the Remote Cluster Configuration resource
`remote-cluster-1` has not been mounted correctly into the `calicoq` pod.

```bash
Endpoints matching selector all():
  Workload endpoint host-a/k8s/kube-system.kube-dns-5fbcb4d67b-7wbhv/eth0
  Workload endpoint host-b/k8s/kube-system.cnx-manager-66c4dbc5b7-6ghsm/eth0
The following problems were encountered connecting to the remote clusters
which may have resulted in incomplete data:
-  RemoteClusterConfiguration(remote-cluster-1): connection to remote cluster failed: stat /etc/remote-cluster-1/kubeconfig: no such file or directory
```
{: .no-select-button}
