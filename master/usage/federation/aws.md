---
title: Federated Endpoint Identity on AWS
---

The simplest way to use the {{site.prodname}} Federation features with an AWS cluster is to set up your Kubernetes cluster
to use the AWS VPC CNI plugin for networking and IP assignment. If you are installing an AWS EKS cluster, this is 
automatically configured to use the AWS VPC CNI plugin.

The AWS VPC CNI plugin creates ENI interfaces for the pods that fall within the VPC of the cluster. Routing to these 
pods is automatically handled by AWS. VPC peerings and VPN connections can be used to provide seemless IP connectivity 
between your AWS cluster and a remote cluster.

If you do not use the AWS VPC CNI plugin, then connecting to another cluster would require installation of a VPN enabled BGP
router within your VPC rather than being able to use the built-in VPN and VPC peering functionality of AWS. Whilst this is
feasible with additional routing components, it is not discussed further here.

This document consists of the following:
-  A pre-installation checklist that should be read prior to setting up an AWS cluster
-  Installation setup that covers installation steps prior to installing Calico
-  An overview of the configuration required to set up an AWS EKS cluster that is peering with, and federating
   a remote on-prem cluster.

## Pre-installation checklist

The following installation requirements should be considered before setting up your Kubernetes cluster in AWS. This assumes
you will be using the AWS VPC CNI plugin to handle routing and IP assignment.

1. When setting up the cluster, ensure you select a VPC that does not overlap with the IP ranges (host and pod IPs) of 
any cluster that you are federating. If you are peering with another AWS cluster, then it should be in a different VPC.

1. If you require access from the pods to the public internet, configure an external NAT gateway for your cluster. 
If you are using an AWS NAT gateway, then you'll need to a configure one or more public subnets within your VPC to home 
the NAT gateway.

## Installing the AWS VPC CNI plugin (including EKS)

To install {{site.prodname}} on an AWS Kubernetes cluster (including EKS), turn up your cluster using your preferred approach 
(e.g. kops, eksctl). 

Before installing Calico, install the AWS VPC CNI plugin using the following instructions:

1. Download the AWS VPC CNI manifest:

   ```bash
   curl https://raw.githubusercontent.com/aws/amazon-vpc-cni-k8s/master/config/v1.1/aws-k8s-cni.yaml -O
   ```

1. Edit the manifest to disable the on-node SNAT. This is required to allow clusters in other VPCs, or connected via a VPN,
   to communicate with the pods. Without this modification, the default behavior for the CNI plugin is to perform SNAT 
   for any packet routed outside the VPC. 
    
   Add the following environment variable snippet to the container environments:
 
   ```bash
   - name: AWS_VPC_K8S_CNI_EXTERNALSNAT
     value: "true"
   ```
   
   For details see the [Amazon VPC CNI Plugin Version 1.1](https://aws.amazon.com/blogs/opensource/vpc-cni-plugin-v1-1-available)
   release notes.
   
1. Apply the manifest using kubectl

   ```bash
   kubectl apply -f aws-k8s-cni.yaml
   ```
   
1. Follow the standard Calico instructions to install [{{site.prodname}} for policy only](/{{page.version}}/getting-started/kubernetes/installation/other), 
   making sure to download the correct manifest.

## Typical AWS configuration with Federated Endpoint Identity

This section gives a brief overview of a example EKS cluster peered with an on-prem cluster running on physical hardware.
Both clusters are running {{site.prodname}}. The clusters have Federated Endpoint Identity configured, with each cluster
referencing the other using the Remote Cluster Configuration resource. See the [Configuring a Remote Cluster for Federation](./configure-rcc) guide for 
more details.

The diagram below captures the main configuration details for this particular set up, which may be adapted for your specific 
requirements. This is purely a guide for settting up one specific configuration.

![A diagram showing the key configuration requirements setting up an AWS cluster (using AWS VPN CNI) peering 
with an on-prem cluster.](/images/federation/aws-rcc.png)

#### AWS configuration:
- A VPC CIDR is chosen that does not overlap with the on-prem IP ranges.
- There are 4 subnets within the VPC, split across two AZs (for availability) such that each AZ has a public and private subnet. In this
  particular example, the split of responsibility is:
  - The private subnet is used for node and pod IP allocation
  - The public subnet is used to home a NAT gateway for pod-to-internet traffic.
- The VPC is peered to an on-prem network using a VPN. This is configured as in AWS as a vgw for the AWS side, and a 
  cgw for the customer side. BGP is used for route distribution.
- Routing table for private subnet has :
  - "propagate" set to "true" to ensure BGP-learned routes are distributed
  - default route to the NAT gateway for public internet traffic
  - local VPC traffic.
- Routing table for public subnet has:
  - default route to the internet gateway.
- Security Group for the worker nodes has:
  - Rule to allow traffic from the peered networks
  - Other rules required for settings up VPN peering (refer to the AWS docs for details)
- The Calico configuration for this deployment has:
  - no IP Pools (SNAT is handled by the NAT gateway; IPIP is disabled)
  - no BGP configuration since BGP peering is handled by AWS
  - a Remote Cluster Configuration resource to reference the on-prem cluster
- The {{site.prodname}} Federated Services Controller is installed to provide discovery of the on-prem cluster
  services.

It is possible to automatically create a Network Load Balancer for the AWS deployment by applying a service with the
correct annotation. The diagram has an example manifest.

#### On-prem configuration
It is assumed that in this example the cluster is installed on real hardware, and that node and pod IPs are routable, 
using an edge VPN router to peer with the AWS cluster.
- The Calico configuration for this deployment has:
  - IP Pool configured for the on-prem IP assignment (IPIP is disabled)
  - If the IP Pool has Outgoing NAT enabled then an IP Pool covering the AWS cluster VPC should be added, with `disabled`
    set to `true`. This pool would not be used for IP allocations, but would instruct Felix to not perform SNAT for traffic
    to the AWS cluster.
  - BGP peering to the VPN router. For example, disable node-to-node mesh and configure one or more Route Reflectors as
    global peers for the entire on-prem cluster.
  - a Remote Cluster Configuration resource to reference the AWS cluster
- The {{site.prodname}} Federated Services Controller is installed to provide discovery of the AWS cluster
  services.
