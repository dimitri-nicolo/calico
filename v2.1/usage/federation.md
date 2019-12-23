---
title: Federated Endpoint Identity for Kubernetes
canonical_url: https://docs.tigera.io/v2.3/usage/federation
---

Federated endpoint identity allows local policy rules to reference the labels on workload and host endpoints from a remote cluster. This means that the policies rendered on the local cluster can reference endpoints from remote clusters.

Federated endpoint identity does not cause the network policies to be federated, just the labels on remote endpoints are fetched from remote clusters. The policies from a remote cluster won't apply to the endpoints on the local cluster, but the policies on the local cluster can reference the labels on the endpoints on remote clusters. Traffic on the local cluster that uses the IP addresses of the remote endpoints will have policy enforced on it using the local policy rules and the labels from the remote endpoints.

The endpoints from remote clusters appear in the output of the calicoq tool. They do not appear in either the the {{site.tseeprodname}} manager or the calicoctl command lines tools.

The configuration of this feature uses the [RemoteClusterConfiguration](/{{page.version}}/reference/calicoctl/resources/remoteclusterconfiguration) resource. The resources represent a remote cluster that the local cluster should retrieve endpoint information from. The local cluster can talk to multiple remote clusters and they can run either the `etcdv3` and `kubernetes` datastores.

Typha is required to use federated endpoint identity.

### Using Kubernetes secrets
The [RemoteClusterConfiguration](/{{page.version}}/reference/calicoctl/resources/remoteclusterconfiguration) resource contains the connection details for remote clusters and some fields in this resource reference files. It is recommended that the files are backed by a Kubernetes secret so that they can easily be mapped into the Typha containers. The Typha section of the manifest files contain commented out configuration to create this secrets volume.
* [Hosted Kubernetes with networking](/{{page.version}}/getting-started/kubernetes/installation/hosted/kubernetes-datastore/calico-networking/1.7/calico.yaml)
* [Hosted Kubernetes with policy only](/{{page.version}}/getting-started/kubernetes/installation/hosted/kubernetes-datastore/policy-only/1.7/calico.yaml)

After creating a secret using the command below, uncomment the two sections in the appropriate manifest file and apply it to your cluster.

Create the secret with `kubectl create secret generic federated-calico-secret --from-file=kubeconfig-cluster1 --namespace=kube-system`. Where `kubeconfig-cluster1` is a kubeconfig file for a remote cluster. The `--from-file` flag can be repeated multiple times. For further information on working with secrets, see the [Kubernetes documentation](https://kubernetes.io/docs/concepts/configuration/secret/)

The files will appear in the Typha pod in the `/etc/calico-federation` directory, so a [RemoteClusterConfiguration](/{{page.version}}/reference/calicoctl/resources/remoteclusterconfiguration) reference the files using this path, e.g.:

```yaml
apiVersion: projectcalico.org/v3
kind: RemoteClusterConfiguration
metadata:
  name: cluster1
spec:
  datastoreType: kubernetes
  kubeconfig: /etc/calico-federation/kubeconfig-cluster1
```

### Adding a remote cluster

To add a remote cluster, create a `RemoteClusterConfiguration` resource using calicoctl. Typha will retrieve remote endpoint information and the rules on the local cluster will start applying to it.

### Removing a remote cluster

Remove a remote cluster by deleting the resource using calicoctl. All of the endpoint information from the remote cluster will be removed from the local cluster.
