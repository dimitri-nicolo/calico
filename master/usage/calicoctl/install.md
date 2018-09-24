---
title: Installing calicoctl
---

## About installing calicoctl

`calicoctl` allows you to create, read, update, and delete {{site.prodname}} objects
from the command line. These objects represent the networking and policy
of your cluster.

You should limit access to `calicoctl` and your {{site.prodname}} datastore to
trusted administrators. We discuss methods of limiting access to the
{{site.prodname}} datastore in the [configuration section](/{{page.version}}/usage/calicoctl/configure/).

You can run `calicoctl` on any host with network access to the
{{site.prodname}} datastore as either a binary or a container.
For step-by-step instructions, refer to the section that
corresponds to your desired deployment.

- [As a binary on a single host](#installing-calicoctl-as-a-binary-on-a-single-host)

- [As a container on a single host](#installing-calicoctl-as-a-container-on-a-single-host)

- [As a Kubernetes pod](#installing-calicoctl-as-a-kubernetes-pod)


## Installing calicoctl as a binary on a single host

{% include {{page.version}}/ctl-binary-install.md cli="calicoctl" codepath="/calicoctl" %}

**Next step**:

[Configure `calicoctl` to connect to your datastore](/{{page.version}}/usage/calicoctl/configure/).

{% include {{page.version}}/ctl-container-install.md cli="calicoctl" %}

### Executing a calicoctl command in the pod

You can run calicoctl commands in the calicoctl pod using kubectl as shown below.

```
kubectl exec -i -n kube-system calicoctl -- /calicoctl get nodes -o wide
NAME                   ASN         IPV4             IPV6
test-master            (unknown)   10.128.0.19/32
test-node-0            (unknown)   10.128.0.20/32
test-node-1            (unknown)   10.128.0.29/32
```

Or to apply a file:
```
kubectl exec -i -n kube-system calicoctl /calicoctl -- apply -f - < myfile.yaml

```

You may find it helpful to create an alias:
```
alias calicoctl="kubectl exec -i -n kube-system calicoctl /calicoctl -- "
```
