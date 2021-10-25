---
title: Troubleshoot egress gateways
description: Use checklist to troubleshoot common problems.
canonical_url: '/networking/egress/troubleshoot'
---

- [Checklist of common problems](#checklist-of-common-problems)
- [Connection to an egress gateway cannot be established](#connection-to-an-egress-gateway-cannot-be-established)
- [Connection to an egress gateway is established, but destination is not getting correct IP](#connection-to-an-egress-gateway-is-established-but-destination-is-not-getting-correct-ip)
- [Previous known issues](#previous-known-issues)

### Checklist of common problems

Use the following checklist to troubleshoot, or to collect details before opening a Support ticket. 

#### Is the egress gateway enabled?

Egress gateway is disabled by default.  Have you enabled it in [Felix configuration]({{site.baseurl}}/reference/resources/felixconfig) by setting `egressIPSupport` to `EnabledPerNamespace` or `EnabledPerNamespaceOrPerPod`?

#### Does your egress gateway routing go through a router?

As shown in the following diagram, from the gateway to the destination, the source IP is the egress IP.  On the return path, from the destination back to the gateway, the destination IP is the egress IP. If there are any routers between the gateway and the destination, they must all know how to route the egress IP back to the gateway. If they don’t, the attempted connection cannot be established.

![egress-basic-routing]({{site.baseurl}}/images/egress-basic-routing.svg)

Options to make routers aware of egress IP:

- Program routes statically on routers
- Peer routers with the cluster, directly or indirectly using BGP or some other protocol, or other method so routers learn about the egress IP

#### Does your egress gateway have required metadata?

Review important egress gateway metadata (for example, namespace and labels); they are required for a client to identify the gateway(s) that it should use.

#### Is natOutgoing on your IPPool set up correctly?

For most egress gateway scenarios you should have: `natOutgoing: false` on the egress IPPool. If you have `natOutgoing: true`, the egress gateway will SNAT to its own IP, which is the intended egress gateway IP. But then the egress gateway’s node will also SNAT to its own IP (i.e. the node IP), which immediately overrides the egress gateway IP.

#### Do clients and nodes have required selectors?

Review the following annotations that are required for the client to identify its egress gateways:

- egress.projectcalico.org/selector
- egress.projectcalico.org/namespaceSelector

on

- Client pod
- Client pod’s namespace

#### Check calico-node health

Check that your calico-node pods are consistently running and ready, especially on the nodes hosting the client and gateway pods. Confirming healthy pods will rule out possible bugs. 

#### Check IP rule and routing setup on the client node

**Run `ip rule`**

On the client node, run:

```
ip rule
```

**Sample output** 

You will see a line for each pod on the node that is configured to use an egress gateway. 

```
from 192.168.24.35 fwmark 0x80000/0x80000 lookup 250
```

Where:

- `192.168.24.35` is the relevant client's pod IP
- `250` is the routing table number
- `fwmark 0x80000/0x80000` is the bit/mask

If you don’t see this, it means one of the following:

- egressIPSupport is not enabled
- egressIPSupport is enabled, but you have not configured egress annotations on the client pod or on its namespace
- egressIPSupport is EnabledPerNamespace and you have configured egress annotations on the client pod, but not on its namespace


**Run `ip route show table`**

On the client node, run the following command using the routing table number from the `ip rule` command. For example: `250`.

```
ip route show table <routing-table-number>
``` 

**Sample output: clients using a single egress gateway**

```
default via 11.11.11.1 dev egress.calico onlink
```

**Sample: clients using multiple gateways**

```
default onlink

      nexthop via  11.11.11.1 dev egress.calico weight 1 onlink
      nexthop via  11.11.11.2 dev egress.calico weight 1 onlink
```

If you see nothing at all, or the following:

```
unreachable default scope link
```

- Verify that you have provisioned the gateways
- Review the selectors, and gateway namespace and labels to determine why they aren’t matching each other

#### Do you have egress IPs in BGPConfiguration svcExternalIPs?

You should not have any egress IPs or pod IP ranges in BGPConfiguration `svcExternalIPs` or `svcClusterIPs` fields; it causes problems if you do.

By default, {{site.prodname}} BGP exports all pod IPs, which includes egress gateway IPs because they are pod IPs. But you can also use the `svc*` fields in BGPConfiguration to export additional IP ranges, in particular Kubernetes Service IPs. Because {{site.prodname}} exports additional IP ranges in a different way from pod IPs, things can go wrong if you include pod IPs in the additional ranges.

### Connection to an egress gateway cannot be established

If the outbound connection cannot be established, the policy may be denying the flow. As shown in the following diagram, policy is enforced at more points in an egress gateway flow.

![egress-basic-routing]({{site.baseurl}}/images/egress-basic-routing.svg)

**Policy enforcement**:

- From the client pod, egress 
- To the gateway pod, ingress 
- From the gateway pod, egress 
- Any relevant HostEndpoints that are configured in your cluster

In [Manager UI]({{site.baseurl}}/visibility/get-started-cem), check for dropped packets because of policy on the outbound connection path. If you are using the iptables dataplane, you can also run the following command on the client and gateway nodes to look at a lower level.

```
watch iptables-save -c | grep DROP | grep -v 0:0
```

### Connection to an egress gateway is established, but destination is not getting correct IP

If you see that the outbound connection established, but the source IP is incorrect at the destination, this can indicate that other SNAT or MASQUERADE is taking effect after the packet leaves the egress gateway pod and is overriding the egress gateway IP. If you intentionally have a MASQUERADE/SNAT for another general purpose, you must filter it so it does not apply to traffic whose source IP comes from the egress gateway pool.

To check the egress gateway’s node, use iptables:

```
iptables-save -c | grep -i MASQUERADE
iptables-save -c | grep -i SNAT
```
