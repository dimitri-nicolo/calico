
# MCM Installation Guide

Setup for Multi-Cluster Management is now tailored for TSEE v2.5.1 using Helm charts. Please follow the [User Guide](https://docs.google.com/document/d/1CbJvorX3wXWMxRLqN8DJDerp-iidBN88Z5ll1_4Ziw8/edit?usp=sharing).

Note: [How to Manage Multiple Terraforms](#how-to-manage-multiple-terraforms).

## Standalone Cluster (new Voltron-only architecture)

*This step is temporary*

This step is required to a standalone TSEE cluster with multi cluster mode disabled. 

Please follow the [setup-cnx-manager-management-cluster.bash](/install-scripts/setup-cnx-manager-management-cluster.bash).

## Management Cluster

This step is required to enable multi cluster mode on a standalone TSEE cluster. A standalone already has Voltron installed. All you need to do is to enable the multi cluster mode that allows an user to create managed clusters.

*This step is optional
Voltron needs an accessible IP to accept tunnels. The script will pick the internal ip of the master node. If you wish to change this, add the following environment variable VOLTRON_PUBLIC_IP to cnx-manager-proxy container.*

Next run the following script to enable a multi cluster mode.
Please follow the [setup-voltron-mgmt-cluster.bash](/install-scripts/setup-voltron-mgmt-cluster.bash).

## Managed Clusters

Create a managed cluster via CNX manager UI and apply the manifest for guardian.
This step is required to enable multi cluster client mode on a standalone TSEE cluster.

Please follow the [setup-guardian-app-cluster.bash](/install-scripts/setup-guardian-app-cluster.bash).

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

