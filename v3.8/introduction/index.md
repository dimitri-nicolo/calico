---
title: About Tigera Secure Enterprise Edition (EE)
canonical_url: https://docs.tigera.io/v2.3/introduction/
description: Home
layout: docwithnav
---

## What is {{site.tseeprodname}}?

Modern applications are more distributed, dynamically orchestrated, and
run across multi-cloud infrastructure. To protect workloads and enforce
compliance, connectivity must be established and secured in a highly dynamic
environment that includes microservices, containers, and virtual machines.

{{site.tseeprodname}} provides secure application connectivity across multi-cloud and
legacy environments, with the enterprise control and compliance capabilities
required for mission-critical deployments.

Designed from the ground up as cloud-native software, {{site.tseeprodname}} builds on leading
open source projects like [Calico](https://docs.projectcalico.org/).
It connects and secures container, virtual machine, and bare metal host
workloads in public cloud and private data centers.

## Why use {{site.tseeprodname}}?

### Best practices for network security

{{site.tseeprodname}}’s rich network policy model makes it easy to lock down communication so the only traffic that flows is the traffic you want to flow.
You can think of {{site.tseeprodname}}’s security enforcement as wrapping each of your workloads with its own personal firewall that is dynamically
re-configured in real time as you deploy new services or scale your application up or down.

{{site.tseeprodname}}’s policy engine can enforce the same policy model at the host networking layer and (if using Istio & Envoy) at the service mesh
layer, protecting your infrastructure from compromised workloads and protecting your workloads from compromised infrastructure.

### Performance

{{site.tseeprodname}} uses the Linux kernel’s built-in highly optimized forwarding and access control capabilities to deliver native Linux networking dataplane
performance, typically without requiring any of the encap/decap overheads associated with first generation SDN networks. {{site.tseeprodname}}’s control plane
and policy engine has been fine tuned over many years of production use to minimize overall CPU usage and occupancy.

### Scalability

{{site.tseeprodname}}’s core design principles leverage best practice cloud-native design patterns combined with proven standards based network protocols
trusted worldwide by the largest internet carriers. The result is a solution with exceptional scalability that has been running at scale in
production for years. {{site.tseeprodname}}’s development test cycle includes regularly testing multi-thousand node clusters.  Whether you are running a 10
node cluster, 100 node cluster, or more, you reap the benefits of the improved performance and scalability
characteristics demanded by the largest Kubernetes clusters.

### Interoperability

{{site.tseeprodname}} enables Kubernetes workloads and non-Kubernetes or legacy workloads to communicate seamlessly and securely.  Kubernetes pods are first
class citizens on your network and able to communicate with any other workload on your network.  In addition {{site.tseeprodname}} can seamlessly extend to
secure your existing host based workloads (whether in public cloud or on-prem on VMs or bare metal servers) alongside Kubernetes.  All workloads
are subject to the same network policy model so the only traffic that is allowed to flow is the traffic you expect to flow.

### Looks familiar

{{site.tseeprodname}} uses the Linux primitives that existing system administrators are already familiar with. Type in your favorite Linux networking command
and you’ll get the results you expect.  In the vast majority of deployments the packet leaving your application is the packet that goes on the wire,
with no encapsulation, tunnels, or overlays.  All the existings tools that system and network administrators use to gain visibility
and analyze networking issues work as they do today.

### Full Kubernetes network policy support

{{site.tseeprodname}}’s network policy engine formed the original reference implementation of Kubernetes network policy during the development of the API. {{site.tseeprodname}} is
distinguished in that it implements the full set of features defined by the API giving users all the capabilities and flexibility envisaged when the API was defined.
And for users that require even more power, {{site.tseeprodname}} supports an extended set of network policy capabilities that work seamlessly alongside the Kubernetes API
giving users even more flexibility in how they define their network policies.
