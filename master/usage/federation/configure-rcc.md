---
title: Configuring a Remote Cluster for Federation
---

The configuration of both Federated Endpoint Identity and the optional Federated Services are handled through the
[Remote Cluster Configuration](/{{page.version}}/reference/calicoctl/resources/remoteclusterconfiguration) resource. Each 
instance of this resource represents a single remote cluster from which the local cluster can retrieve endpoint 
information. The local cluster can talk to multiple remote clusters. Federated Services also requires additional configuration
through the service annotations.

Both `etcdv3` and `kubernetes` Calico datastores deployments are supported for remote clusters.

The Remote Cluster Configuration contains a number of fields that are used to specify file paths (e.g. Kubeconfig). It is
recommended that these files are backed by a Kubernetes secret and then mounted into the calico/node and Federated 
Services Controller pods.

In addition to the Remote Cluster Configuration resources, it may be necessary to modify your IP Pool configuration.

## Configuring the Remote Cluster Configuration resources

### Calico datastore type

#### etcdv3

If the remote cluster uses etcdv3 as the Calico datastore, set the `datastoreType` in the RemoteClusterConfiguration
to `etcdv3`, and populate the required `etcd*` fields in the resource.

In addition, if you intend to use the Federated Services Controller then you must also specify access details for the
remote cluster Kubernetes API by populating, as appropriate, the `kubeconfig` and/or other `k8s*` fields.

#### Kubernetes API

If the remote cluster uses the Kubernetes API as the Calico datastore, set the `datastoreType` in the RemoteClusterConfiguration
to `kubernetes`, and populate as appropriate the `kubeconfig` and/or other `k8s*` fields.

### Creating a Kubeconfig for a remote cluster

The {{site.prodname}} installation manifests (see [Installing {{site.prodname}} on Kubernetes](/{{page.version}}/getting-started/kubernetes/installation/index))
contain a Service Account specifically for use as a remote cluster. This is true for both Kubernetes API and etcdv3 Calico 
deployments.

If your remote cluster has the `tigera-federation-remote-cluster` service account then you can create a `Kubeconfig` 
specifically for this service account. The local cluster can use this to access the Kubernetes API on the remote 
cluster.

Furthermore, if your remote cluster has RBAC enabled and you have applied the appropriate RBAC manifest for your {{site.prodname}}
deployment type, then access via this service account is limited to read only access of the minimal required set of 
resource types for {{site.prodname}} federated endpoint identity and service federation.

See the [Writing kubeconfig files](/{{page.version}}/usage/writing-kubeconfig) guide for details on creating a Kubeconfig for a Service Account.

> **Tips**
> * If you are running {{site.prodname}} in an AWS EKS deployment then it will be necessary to create a Kubeconfig as described since
>   the admin kubeconfig uses heptio authentication which is currently not supported by Calico.
> * If your remote cluster uses etcdv3 for the Calico datastore, and you do not require the Federated Services Controller,
>   it is not necessary to specify any Kubernetes access information for the remote cluster.
{: .alert .alert-success}

> **Warning**: When upgrading your Calico deployment, it is important to ensure the calico, and federated services controller 
> manifest that you apply also include any secrets that you previously had mounted in. Failure to do so may result in loss of 
> Federation function which may include loss of connectivity between clusters. 
{: .alert .alert-danger}

### Using Kubernetes secrets

The [RemoteClusterConfiguration](/{{page.version}}/reference/calicoctl/resources/remoteclusterconfiguration) resource 
contains the connection details for remote clusters and some fields in this resource reference files. It is 
recommended that the files are backed by a Kubernetes secret so that they can easily be mapped into the Typha and Federated
Services Controller containers.

Both the Typha section of the main {{site.prodname}} manifest file and the Federated Services Controller manifest file contains
commented out configuration templates to create a secrets volume.

The templates in the manifests refer to a single secret and a volume mount called `tigera-federation-remotecluster`. It is 
probably simplest to maintain a seperate secret containing all of the files required for each single RemoteClusterConfiguration.
In this case, you would duplicate and rename the secret and volume mounts to include entries for each remote cluster.

After creating the secrets and editing the manifest files to reference the secrets, apply the manifests to your cluster
using kubectl.

As an example, to create a secret called `tigera-federation-remotecluster` which contains the contents of a kubeconfig file
`kubeconfig-remotecluster`, you would run the command:

```$bash
kubectl create secret generic tigera-federation-remotecluster --from-file=kubeconfig-remotecluster --namespace=kube-system`
```

> **Tips**:
> -  The `--from-file` flag can be repeated multiple times to include multiple files in the secret which will get
>    mounted into the same directory. For further information on working with secrets, see the [Kubernetes documentation](https://kubernetes.io/docs/concepts/configuration/secret/)
> -  It is not currently possible to modify an existing Kubernetes secret. Therefore, if you need to add or modify files in a
>    secret then it may be less disruptive to create a new secrets containing all of the required files and update and apply 
>    the installation manifests to use the new secret. The old secret can be removed after all of components have been restarted.
{: .alert .alert-success}

The files will appear in the Typha pod in the `/etc/tigera-federation-remotecluster` directory, so a 
[RemoteClusterConfiguration](/{{page.version}}/reference/calicoctl/resources/remoteclusterconfiguration) reference 
the files using this path, e.g.:

```yaml
apiVersion: projectcalico.org/v3
kind: RemoteClusterConfiguration
metadata:
  name: cluster1
spec:
  datastoreType: kubernetes
  kubeconfig: /etc/tigera-federation-remotecluster/kubeconfig-remotecluster
```

## Configuring IP Pool resources for federated endpoint identity

If your local cluster has NATOutgoing configured on your IP Pools, it is necessary to configure IP Pools covering the IP ranges 
of your remote clusters. This ensures outgoing NAT is not performed on packets bound for the remote clusters. These additional
IP Pools should have `disabled` set to `true` to ensure the pools are not used for IP assignment on the local cluster.

The IP Pool CIDR used for Pod IP allocation should not overlap with any of the IP ranges used by the pods and nodes of any
other federated cluster.

For example, you may configure the following on your local cluster, referring the the IPPool on a remote cluster:

```yaml
apiVersion: projectcalico.org/v3
kind: IPPool
metadata:
  name: cluster1-main-pool
spec:
  cidr: 192.168.0.0/18
  disabled: true
```
