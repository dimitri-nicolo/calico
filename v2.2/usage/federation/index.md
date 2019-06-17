---
title: Overview
canonical_url: https://docs.tigera.io/v2.3/usage/federation/
---

## About federation

Federation allows you to:

- [Create policies that reference pods and host endpoints across multiple clusters](#federated-endpoint-identity).

- [Create Kubernetes services that span multiple clusters while still keeping fine-grained policy](#federated-services).

## Terminology

When discussing federation, we use the following terms:

- **Local clusters** retrieve endpoint data from remote clusters.

- **Remote clusters** allow local clusters to retrieve endpoint data.

Each cluster in the federation acts as both a local and remote cluster.

## Federated endpoint identity

Federated endpoint identity allows a local Kubernetes cluster to include the workload endpoints (i.e. pods) and host
endpoints of a remote cluster in the calculation of the local policies that are applied on each node of the local
cluster. Pods on a remote cluster may be referenced by label in the local policy rules, rather than needing to reference
them by IP address. The main policy selector still only refers to local endpoints; that selector chooses which local
endpoints the policy is applied to.

As an example, you can easily create a policy which applies to the local endpoints denying inbound traffic from all pods
in a remote cluster except for those within the same namespace.

Federated endpoint identity does not cause the network policies to be federated, i.e., the policies from a remote
cluster won't apply to the endpoints on the local cluster. Similarly, the policy from the local cluster is only rendered
locally and applied to the local endpoints.

Policy rule selectors on the local cluster should be able to reference the correct endpoints across all of the clusters.
This requires coordination between clusters for the following:
   -  Namespace names
   -  Host Endpoint label names and values
   -  Pod label names and values within the namespace

Configuration for this feature is through the [Remote Cluster Configuration](/{{page.version}}/reference/calicoctl/resources/remoteclusterconfiguration)
resource. A Remote Cluster Configuration resource should be added for each remote cluster that you want to include.
Similar configuration should be applied on the remote clusters if you require Federated Endpoint Identity on those
clusters.

Refer to [Configuring local clusters](./configure-rcc) for more information.

## Federated Services

For pod-to-pod communication it is usually necessary to use a service discovery mechanism to locate the pod IP addresses.
Kubernetes provides `services` for accessing related sets of pods. If a Kubernetes service has a pod selector specified
then Kubernetes will manage that service, populating the service endpoints from the local pods that match the selector.

Tigera Federated Services Controller is used alongside Federated Endpoint
Identity to provide discovery of remote pods. It extends the standard Kubernetes service and endpoints functionality to
provide federation of [Kubernetes endpoints](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.10/#endpoints-v1-core) across all of the clusters.

Configuration for this feature is also through the [Remote Cluster Configuration](/{{page.version}}/reference/calicoctl/resources/remoteclusterconfiguration)
resource. In addition, a [service annotation](/{{page.version}}/usage/federation/services-controller) is used to configure
a federated service.

> **Note**: The controller always uses the pod IP for the service endpoints even for pods in remote clusters,
> thus if a pod on the local cluster uses a federated service to access a pod in a remote cluster, source and
> destination IP addresses are preserved allowing {{site.prodname}} fine-grained policy to be applied.
>
> Contrast this to Kubernetes Federation, where federated services use the public service IP to access a remote service.
> The use of the public IP requires NAT and is therefore not suitable for Federated Endpoint Identity.
{: .alert .alert-info}

The `calicoq` command line tool will display the endpoints from remote clusters provided that the files specified
in the Remote Cluster Configuration resources are also accessible to `calicoq`, using the same file path.
[Running `calicoq` as a pod on the local cluster](/{{page.version}}/usage/calicoq/#installing-calicoq-as-a-kubernetes-pod)
is the simplest way to ensure it has access to the correct configuration.

At this time, neither the {{site.prodname}} Manager nor `calicoctl` can be used to view endpoints from remote clusters.
