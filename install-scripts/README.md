
# Voltron (streamlined) Installation Guide

Note: [How to Manage Multiple Terraforms](#how-to-manage-multiple-terraforms).

## Management Cluster

1. install a [calico ready cluster](https://github.com/tigera/calico-ready-clusters)
1. [install vanilla TSEE](https://tigera.atlassian.net/wiki/spaces/ENG/pages/46759954/Install+CNX+with+CRC+calico-ready-clusters)
1. voltron needs to be accessible from where the guardians run; you need an IP:PORT combination with appropriate firewall rules that
reaches voltron:
   - in tigera-dev GCP, with all clusters in the same account, run `kubectl get nodes -o wide` and grab the INTERNAL-IP of _any_ node  
   (consider your firewall rules if that doesn't work)
1. from this repo, run `make manifests BRANCH_NAME=master`
1. edit `manifests/voltron.yaml`, change `cnx-voltron.public-ip` to the node IP, keep the port to `30449`
1. scp the following files, from this repo, to the master node:  
   - install-scripts/post-cnx.bash
   - install-scripts/fixup-compliance.bash
   - install-scripts/fixup-es-proxy.bash
   - install-scripts/register-guardian.bash
   - manifests/voltron.yaml
1. on the master node, run `bash post-cnx.bash`, `bash fixup-compliance.bash` and `bash fixup-es-proxy.bash`
1. wait for voltron pod to be running, `kubectl get pods -n calico-monitoring`
1. run `bash register-guardian.bash "mgmt" "Local cluster - Mgmt"`
1. run `kubectl apply -f guardian-mgmt.yaml`

## App Clusters

For each app cluster:

1. install a [calico ready cluster](https://github.com/tigera/calico-ready-clusters)
1. [install vanilla TSEE](https://tigera.atlassian.net/wiki/spaces/ENG/pages/46759954/Install+CNX+with+CRC+calico-ready-clusters)
1. ssh to the master node of the management cluster, _NOT_ app cluster
1. run `bash register-guardian.bash "id" "displayName"`
1. copy the generated `guardian-$(id).yaml` to a location where you can run kubectl for your _app cluster_
1. run `kubectl apply -f guardian-$(id).yaml`

## Easier TSEE install?

This is an alternative to "install vanilla TSEE"

Consider using `remote-cnx.bash` from the current directory. It runs under
`kubeadm/1.6/` of the CRC repo. It assumes the existence of:

- config.json
- license.yaml  -- available [here](https://tigera.atlassian.net/wiki/spaces/ENG/pages/44925032/Files+we+should+neither+lose+nor+give+away+test+licenses+secrets+etc)
- cnx_jane_pw   -- optionally, contains jane password in plaintext

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

