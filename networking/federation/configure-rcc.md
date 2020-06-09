---
title: Configure access to remote clusters
description: Steps to allow a local cluster to pull endpoint data from a remove cluster.
canonical_url: /networking/federation/configure-rcc
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


## Adding resources for a remote cluster

To configure a Remote Cluster Configuration resource [create a Secret](#adding-a-secret-with-cluster-access-information)
which contains the remote cluster connection information, [ensure the secrets can be accessed](#ensure-secrets-can-be-retrieved)
 and then [create a Remote Cluster Configuration resource](#adding-a-remote-cluster-configuration)
that references the Secret.

### Prerequisite

Before you can add a Remote Cluster Configuration resource to a cluster, you must
[install {{site.prodname}}]({{site.baseurl}}/getting-started/kubernetes) on the cluster.


### Adding a Secret with Cluster Access information

The simplest method to create a secret with the appropriate fields is to use the `kubectl` command
as it will correctly encode the data and format the file.

Create a Secret with a command like the following for a remote cluster.
```bash
kubectl create secret generic remote-cluster-secret-name -n calico-system \
    --from-literal=datastoreType=kubernetes \
    --from-file=kubeconfig=<kubeconfig file>
```

Additional fields can be added by adding arguments to the command:
- `--from-literal=<key>=<value>` for adding literal values
- `--from-file=<key>=<file>` for adding values with file contents

### Ensure secrets can be retrieved

{{site.prodname}} does not generally have access to all secrets in a cluster so it is necessary to create a
Role and RoleBinding for each namespace where the secrets for RemoteClusterConfigurations are being created. You can reduce the
number of Role and RoleBindings needed by putting the remote cluster Secrets in a dedicated namespace.

```
kubectl create -f - <<EOF
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: remote-cluster-secret-access
  namespace: <namespace>
rules:
- apiGroups: [""]
  resources: ["secrets"]
  verbs: ["watch", "list", "get"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: remote-cluster-secret-access
  namespace: <namespace>
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: remote-cluster-secret-access
subjects:
- kind: ServiceAccount
  name: calico-typha
  namespace: calico-system
EOF
```

> **Note**: If you need to configure access to multiple namespaces you will need to change the name used
> for the Role and RoleBinding so they are unique for each namespace.
{: .alert .alert-info}

### Adding a Remote Cluster Configuration

Each instance of the [Remote Cluster Configuration]({{site.baseurl}}/reference/resources/remoteclusterconfiguration)
resource represents a single remote cluster from which the local cluster can retrieve endpoint information.

```yaml
apiVersion: projectcalico.org/v3
kind: RemoteClusterConfiguration
metadata:
  name: cluster-n
spec:
  clusterAccessSecret:
    name: remote-cluster-secret-name
    namespace: remote-cluster-secret-namespace
    kind: Secret
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
