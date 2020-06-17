---
title: Query remote clusters with calicoq
description: Use CLI to check if remote cluster endpoints are accessible. 
canonical_url: /networking/federation/calicoq
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
the `calicoq` output includes details about the error.

For example, the following output shows an error indicating the secret `remote-cluster-secret` is not accessible.

```bash
E0615 12:24:04.895079   30873 reflector.go:153] github.com/projectcalico/libcalico-go/lib/backend/syncersv1/remotecluster/secret_watcher.go:111: Failed to list *v1.Secret: secrets "remote-cluster-secret" is forbidden: User "system:serviceaccount:policy-demo:limited-sa" cannot list resource "secrets" in API group "" in the namespace "remote-cluster-ns"
Endpoints matching selector all():
  Workload endpoint host-a/k8s/kube-system.kube-dns-5fbcb4d67b-7wbhv/eth0
  Workload endpoint host-b/k8s/kube-system.cnx-manager-66c4dbc5b7-6ghsm/eth0
```
{: .no-select-button}

The following example output shows an error indicating the connection information for the Remote Cluster Configuration resource
`remote-cluster-1` is not correct.

```bash
Endpoints matching selector all():
  Workload endpoint host-a/k8s/kube-system.kube-dns-5fbcb4d67b-7wbhv/eth0
  Workload endpoint host-b/k8s/kube-system.cnx-manager-66c4dbc5b7-6ghsm/eth0
The following problems were encountered connecting to the remote clusters
which may have resulted in incomplete data:
-  RemoteClusterConfiguration(remote-cluster-1): connection to remote cluster failed: Get https://192.168.0.55:6443/api/v1/pods: dial tcp 192.168.0.55:6443: i/o timeout
```
{: .no-select-button}
