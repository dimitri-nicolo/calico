---
title: Installing calicoq
---

## About installing calicoq

You can run `calicoq` on any host with network access to the
{{site.prodname}} datastore as either a binary or a container.
For step-by-step instructions, refer to the section that
corresponds to your desired deployment.

- [As a binary on a single host](#installing-calicoq-as-a-binary-on-a-single-host)

- [As a container on a single host](#installing-calicoq-as-a-container-on-a-single-host)

- [As a Kubernetes pod](#installing-calicoq-as-a-kubernetes-pod)


## Installing calicoq as a binary on a single host

{% include {{page.version}}/ctl-binary-install.md cli="calicoq" codepath="/calicoq" %}

**Next step**:

[Configure `calicoq` to connect to your datastore](/{{page.version}}/usage/calicoq/configure/).

{% include {{page.version}}/ctl-container-install.md cli="calicoq" %}

### Executing a calicoq command in the pod

You can run calicoq commands in the calicoq pod using kubectl as shown below.

```
$ kubectl exec -ti -n kube-system calicoq -- /calicoq eval "all()"
Endpoints matching selector all():
  Workload endpoint host-a/k8s/kube-system.kube-dns-5fbcb4d67b-7wbhv/eth0
  Workload endpoint host-b/k8s/kube-system.cnx-manager-66c4dbc5b7-6ghsm/eth0
```

If you are using [Federated Endpoint Identity](/{{page.version}}/usage/federation/index) the output from calicoq will include 
endpoints from the remote clusters.  The name of the remote cluster (as configured in the Remote Cluster Configuration 
resource) will be prepended to the  endpoint name. In the following, the `remote-cluster-1` prefix indicates these 
endpoints are from a remote cluster:

```
$ kubectl exec -ti -n kube-system calicoq -- /calicoq eval "all()"
Endpoints matching selector all():
  Workload endpoint remote-cluster-1/host-1/k8s/kube-system.kube-dns-5fbcb4d67b-h6686/eth0
  Workload endpoint remote-cluster-1/host-2/k8s/kube-system.cnx-manager-66c4dbc5b7-6d9xv/eth0
  Workload endpoint host-a/k8s/kube-system.kube-dns-5fbcb4d67b-7wbhv/eth0
  Workload endpoint host-b/k8s/kube-system.cnx-manager-66c4dbc5b7-6ghsm/eth0
```

If a remote cluster is inaccessible, for example due to network failure or due to a misconfiguration, the output 
from calicoq will include details about the error. For example, the following output describes an error indicating the 
kubeconfig file configured in the Remote Cluster Configuration resource `remote-cluster-1` has not been mounted correctly
into the calicoq pod:

```
$ kubectl exec -ti -n kube-system calicoq -- /calicoq eval "all()"
Endpoints matching selector all():
  Workload endpoint host-a/k8s/kube-system.kube-dns-5fbcb4d67b-7wbhv/eth0
  Workload endpoint host-b/k8s/kube-system.cnx-manager-66c4dbc5b7-6ghsm/eth0
The following problems were encountered connecting to the remote clusters
which may have resulted in incomplete data:
-  RemoteClusterConfiguration(remote-cluster-1): connection to remote cluster failed: stat /etc/remote-cluster-1/kubeconfig: no such file or directory
```

