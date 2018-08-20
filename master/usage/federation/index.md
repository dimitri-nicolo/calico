---
title: Federation for Kubernetes
---

There are two main aspects of Federation provided by {{site.prodname}}:
-  [Federated Endpoint Identity](#federated-endpoint-identity) provides the ability to create policy that references
   endpoints by label and namespace across all of the federated clusters.
-  [Federated Services](#federated-services) is used in conjunction with Federated Endpoint Identity, and provides a
   compatible way to do service discovery across the federated clusters.
   
   This document provides an overview of these features, and links to more detailed documentation where appropriate.

## Cluster requirements

To use the {{site.prodname}} Federation feature there are a few requirements for the clusters that you are federating.

-  Typha is required to use federated endpoint identity. Refer to the appropriate installation instructions for details
   on installing typha:
   - [Installing {{site.prodname}} for policy and networking](/{{page.version}}/getting-started/kubernetes/installation/calico)     
   - [Installing {{site.prodname}} for policy (advanced)](/{{page.version}}/getting-started/kubernetes/installation/other)
-  Pod and Node IPs should be routable between clusters without using IPIP or NAT.  This means:
   -  Each cluster should allocate pod IP addresses from different CIDR ranges. For etcd backed Calico, this means the
      calico manifests should be edited to change the pool CIDRs to non-overlapping values for each cluster. For 
      Kubernetes API backed clusters, each Kubernetes cluster should be configured with different, non-overlapping values
      of `--cluster-cidr` (for kubeadm, see `--pod-network-cidr`)
   -  Node IPs should be configured on different subnets between clusters.
   -  IPIP should be disabled on all of the clusters. This restriction may be removed in future releases.
-  Policy rule selectors on the local cluster should be able to reference the correct endpoints across all of the clusters. 
   This requires coordination between clusters for the following:
   -  Namespace names
   -  Host Endpoint label names and values
   -  Pod label names and values within the namespace.
   
If you are installing on AWS, follow the additional requirements in the [Federated Endpoint Identity on AWS](/{{page.version}}/usage/federation/aws)
guide.
      
## Federated Endpoint Identity

Federated endpoint identity allows a local Kubernetes cluster to include the workload endpoints (i.e. Pods) and host 
endpoints of a remote cluster in the calculation of the local policies that are applied on each node of the local 
cluster. Pods on a remote cluster may be referenced by label in the local policy rules, rather than needing to reference
them by IP address. The main policy selector still only refers to local endpoints; that selector chooses which local 
endpoints the policy is applied to.

As an example, you can easily create a policy which applies to the local endpoints denying inbound traffic from all pods
in a remote cluster except for those within the same namespace.

Federated endpoint identity does not cause the network policies to be federated, i.e. the policies from a remote 
cluster won't apply to the endpoints on the local cluster. Similarly, the policy from the local cluster is only rendered 
locally and applied to the local endpoints. 

Configuration for this feature is through the [Remote Cluster Configuration](/{{page.version}}/reference/calicoctl/resources/remoteclusterconfiguration)
resource. A Remote Cluster Configuration resource should be added for each remote cluster that you want to include.
Similar configuration should be applied on the remote clusters if you require Federated Endpoint Identity on those 
clusters.

## Federated Services

For pod to pod communication it is usually necessary to use a service discovery mechanism to locate the pod IP addresses.
Kubernetes provides `services` for accessing related sets of pods. If a Kubernetes service has a Pod selector specified 
then Kubernetes will manage that service, populating the service endpoints from the local Pods that match the selector.

Tigera Federated Services Controller is a separately installable component that is used alongside Federated Endpoint 
Identity to provide discovery of remote pods. It extends the standard Kubernetes service and endpoints functionality to 
provide federation of endpoints across all of the clusters.

Configuration for this feature is also through the [Remote Cluster Configuration](/{{page.version}}/reference/calicoctl/resources/remoteclusterconfiguration)
resource, and as such enabling Federated Services will also enable Federated Endpoint Identity. In addition, a 
[service annotation](/{{page.version}}/usage/federation/services-controller) is used to configure a federated service.

> **Note**: The controller always uses the pod IP for the service endpoints even for pods in remote clusters, 
> thus if a pod on the local cluster uses a federated service to access a pod in a remote cluster, source and 
> destination IP addresses are preserved allowing Calico fine-grained policy to be applied. 
>
> Contrast this to Kubernetes Federation, where federated services use the public service IP to access a remote service.
> The use of the public IP requires NAT and is therefore not suitable for Federated Endpoint Identity.
{: .alert .alert-info}

## Viewing the discovered remote endpoints

The `calicoq` command line tool will display the endpoints from remote clusters provided that the files specified
in the Remote Cluster Configuration resources are also accessible to calicoq, using the same file path. Running 
calicoq as a Pod on the local cluster is the simplest way to ensure it has access to the correct configuration.

At this time, neither CNX manager nor `calicoctl` can be used to view endpoints from remote clusters.

## More information and next steps

For more documentation on {{site.prodname}} Federation, see the following:
- [Federated Endpoint Identity on AWS](/{{page.version}}/usage/federation/aws)
- [Install the Federated Services Controller](/{{page.version}}/getting-started/kubernetes/installation/fed-controller)
- [Configuring a Remote Cluster for Federation](/{{page.version}}/usage/federation/configure-rcc)
- [Configuring a federated service](/{{page.version}}/usage/federation/services-controller)
