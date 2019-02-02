---
title: Example AWS configuration
redirect_from: latest/usage/federation/aws
canonical_url: https://docs.tigera.io/v2.3/usage/federation/aws
---

## Overview

This section gives a brief overview of a example AWS cluster peered with an on-premise cluster running on physical hardware.
Both clusters are running {{site.prodname}}. The clusters have federated identity configured, with each cluster
referencing the other using the Remote Cluster Configuration resource. See [Configuring access to remote clusters](./configure-rcc) for
more details.

The diagram below captures the main configuration details for this particular set up, which may be adapted for your specific
requirements. This is purely a guide for setting up one specific configuration.

![A diagram showing the key configuration requirements setting up an AWS cluster (using AWS VPN CNI) peering
with an on-premise cluster.](/images/federation/aws-rcc.svg)

## AWS configuration
- A VPC CIDR is chosen that does not overlap with the on-premise IP ranges.
- There are 4 subnets within the VPC, split across two AZs (for availability) such that each AZ has a public and private subnet. In this
  particular example, the split of responsibility is:
  - The private subnet is used for node and pod IP allocation
  - The public subnet is used to home a NAT gateway for pod-to-internet traffic.
- The VPC is peered to an on-premise network using a VPN. This is configured as in AWS as a VPN gateway for the AWS side, and a
  classic VPN for the customer side. BGP is used for route distribution.
- Routing table for private subnet has:
  - ``"propagate"`` set to ``"true"`` to ensure BGP-learned routes are distributed
  - Default route to the NAT gateway for public internet traffic
  - Local VPC traffic
- Routing table for public subnet has default route to the internet gateway.
- Security group for the worker nodes has:
  - Rule to allow traffic from the peered networks
  - Other rules required for settings up VPN peering (refer to the AWS docs for details)
- The {{site.prodname}} configuration for this deployment has:
  - No IP pools (SNAT is handled by the NAT gateway; IPIP is disabled)
  - no BGP configuration since BGP peering is handled by AWS
  - a Remote Cluster Configuration resource to reference the on-premise cluster

It is possible to automatically create a Network Load Balancer (NLB) for the AWS deployment by applying a service with the
correct annotation. See below for an example manifest.

```yaml
apiVersion: v1
kind: Service
metadata:
  annotations:
    service.beta.kubernetes.io/aws-load-balancer-type: nlb
  name: nginx-external
spec:
  externalTrafficPolicy: Local
  ports:
  - name: http
    port: 80
    protocol: TCP
    targetPort: 80
  selector:
    run: nginx
  type: LoadBalancer
```

## On-premise configuration
In this example the cluster is installed on real hardware and node and pod IPs are routable,
using an edge VPN router to peer with the AWS cluster.

The {{site.prodname}} configuration for this deployment has:
- IP Pool resource configured for the on-premise IP assignment (IPIP is disabled)
- If the IP Pool has Outgoing NAT enabled then an IP Pool covering the AWS cluster VPC should be added, with `disabled`
  set to `true`. This pool would not be used for IP allocations, but would instruct Felix to not perform SNAT for traffic
  to the AWS cluster.
- BGP peering to the VPN router. For example, if the VPN Router is configured as a route reflector for the on-premise cluster, you would:
  - Configure the default BGP Configuration resource to disable node-to-node mesh.
  - Configure a global BGP Peer resource to peer with the VPN Router.
- A Remote Cluster Configuration resource to reference the AWS cluster
- The {{site.prodname}} Federated Services Controller is installed to provide discovery of the AWS cluster
  services.
