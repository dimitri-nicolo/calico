
# Voltron Installation Guide

Setup for Multi-Cluster Management is now tailered for TSEE v2.5 using Helm charts. Please follow the [User Guide](https://docs.google.com/document/d/1fNWy1lRJ5E41b-kxf5pF-mwltGfY2rWuvYdfSSG5HhA). 

Note: [How to Manage Multiple Terraforms](#how-to-manage-multiple-terraforms).

## Management Cluster

Please follow the [User Guide](https://docs.google.com/document/d/1fNWy1lRJ5E41b-kxf5pF-mwltGfY2rWuvYdfSSG5HhA). 

## App Clusters

Please follow the [User Guide](https://docs.google.com/document/d/1fNWy1lRJ5E41b-kxf5pF-mwltGfY2rWuvYdfSSG5HhA). 

## How to Manage Multiple Terraforms

If your git clone of CRC was used to bring a cluster up, how do you bring another up?

1. git clone another clone of https://github.com/tigera/calico-ready-clusters
1. use `git worktree add ...` on your existing clone -- [inspiration](https://spin.atomicobject.com/2016/06/26/parallelize-development-git-worktrees/)
1. use `terraform workspace` [details](https://www.terraform.io/docs/state/workspaces.html)

```
terraform workspace new <clustername>
terraform apply -var prefix=<username-clustername>
cp master_ssh_key <clustername>-master_ssh
cp admin.conf <clustername>-admin.conf
```

