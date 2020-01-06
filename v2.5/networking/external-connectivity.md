---
title: External connectivity
canonical_url: 'https://docs.tigera.io/v2.3/usage/external-connectivity'
---

{{site.tseeprodname}} creates a routed network on which your containers look like normal IP
speakers. You can connect to them from a host in your cluster (assuming the
network policy you've assigned allows this) using their IP address.

This document discusses connectivity between {{site.tseeprodname}} endpoints and hosts outside
the cluster.

## Outbound connectivity

Outbound connectivity refers to connections originating from {{site.tseeprodname}} endpoints
to destinations outside the cluster.

The easiest way to get outbound connectivity is to turn on NAT Outgoing on all
{{site.tseeprodname}} pools you want to be able to access the internet.

```shell
calicoctl get ipPool
```

### For each pool that needs connectivity:
```
cat << EOF | calicoctl apply -f -
- apiVersion: projectcalico.org/v3
  kind: IPPool
  metadata:
    name: ippool-ext-1
  spec:
    cidr: 192.168.0.0/16
    natOutgoing: true
EOF
```

[set `ipipMode: Always` if needed]

Please note that many solutions for inbound connectivity will also provide
outbound connectivity.

## Inbound connectivity

Inbound connectivity refers to connections to {{site.tseeprodname}} endpoints originating from
outside the cluster.

There are two main approaches: BGP peering into your network infrastructure, or
using orchestrator specific options.

Remember to configure your network policy to allow traffic from the internet!

### BGP peering

This requires access to BGP capable switches or routers in front of your {{site.tseeprodname}}
cluster.

In general, this will involve peering the nodes in your {{site.tseeprodname}} cluster with BGP
capable switches, which act as the gateway to reach {{site.tseeprodname}} endpoints in the
cluster from outside.

A common scenario is for your container hosts to be on their own isolated layer
2 network, like a rack in your server room or an entire data center.  Access to
that network is via a router, which also is the default router for all the
container hosts.

![hosts-on-layer-2-network]({{site.url}}/images/hosts-on-layer-2-network.png)

See the [BGP peering document]({{site.url}}/{{page.version}}/networking/bgp)
for information on how to set up the `{{site.nodecontainer}}` sides of the sessions.
Consult the documentation for your BGP capable switch/router to set up the
switch sides of the sessions.

If you have a small number of hosts, you can configure BGP sessions between your router and each {{site.tseeprodname}}-enabled host. With many hosts, you may wish to use a
route reflector or set up a Layer 3 topology.

There's further advice on network topologies in [{{site.tseeprodname}} over Ethernet fabrics]({{site.url}}/{{page.version}}/networking/design/l2-interconnect-fabric).
We'd also encourage you to [get in touch](https://www.projectcalico.org/contact/)
to discuss your environment.

### Orchestrator specific

{{site.tseeprodname}} supports a number of orchestrator specific options for inbound
connectivity, such as Kubernetes service IPs.

Consult the [documentation for your orchestrator]({{site.url}}/{{page.version}}/getting-started/) for more information.
