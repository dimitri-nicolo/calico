---
title: Deploy a dual ToR cluster
---

### Big picture

Deploy a dual plane cluster to provide redundant connectivity between your workloads.

### Value

A dual plane cluster provides two independent planes of connectivity between all cluster
nodes.  If a link or software component breaks somewhere in one of those planes, cluster
nodes can still communicate over the other plane, and the cluster as a whole continues to
operate normally.

### Features

This how-to guide uses the following features:

**BGPPeer** resource with these fields:
- sourceAddress
- failureDetectionMode
- restartMode
- maxRestartTime
- birdGatewayMode

### Concepts

#### Dual plane connectivity, aka "dual ToR"

Large on-prem Kubernetes clusters, split across multiple server racks, can use two or more
independent planes of connectivity between all the racks, so that the cluster can still
function if there is a single break in connectivity somewhere.

The redundant approach can be applied within each rack as well, such that each node has
two or more independent connections to those connectivity planes.  Typically each rack has
two top-of-rack routers ("ToRs") and each node has two fabric-facing interfaces, each of
which connects over a separate link or Ethernet to one of the ToRs for the rack.

Here's an example of how a dual plane setup might look, with just two racks and two nodes
in each rack.  For simplicity we've shown the connections *between* racks as single links;
in reality that would be more complex, but still following the overall dual plane
paradigm.

![dual-tor]({{site.baseurl}}/images/dual-tor.png)

Because of the two ToRs per rack, the whole setup is often referred to as "dual ToR".

#### ECMP routing

An "Equal Cost Multiple Path" (ECMP) route is one that has multiple possible ways to reach
a given destination or prefix, all of which are considered to be equally good.  A dual ToR
setup naturally generates ECMP routes, with the different paths going over the different
connectivity planes.

When using an ECMP route, Linux decides how to balance traffic across the available paths,
including whether this is informed by TCP and UDP port numbers as well as source and
destination IP addresses, whether the decision is made per-packet, per-connection, or in
some other way, and so on; and the details here have varied with Linux kernel version.
For a clear account of the exact options and behaviors for different kernel versions,
please see [this blog](https://cumulusnetworks.com/blog/celebrating-ecmp-part-two/).

#### Loopback IP addresses

Loopback IP addresses are IP addresses which are assigned to a loopback interface on the
node.  Despite what the name might suggest, loopback IP addresses can be used for sending
and receiving data to and from other nodes.  They help to provide redundant networking,
because using a loopback address avoids tying traffic to a particular interface and, in a
dual ToR setup, allows that traffic to continue flowing even if a particular interface
goes down.

#### BFD

Bidirectional Forwarding Detection (BFD) is [a
protocol](https://tools.ietf.org/html/rfc5880) that detects very quickly when forwarding
along a particular path stops working - whether that's because a link has broken
somewhere, or some software component along the path.

In a dual ToR setup, rapid failure detection is important so that traffic flows within the
cluster can quickly adjust to using the other available connectivity plane.

#### Long Lived Graceful Restart

Long Lived Graceful Restart (LLGR) is [an extension for
BGP](https://tools.ietf.org/html/draft-uttaro-idr-bgp-persistence-05) that handles link
failure by lowering the preference of routes over that link.  This is a compromise between
the base BGP behaviour - which is immediately to remove those routes - and traditional BGP
Graceful Restart behaviour - which is not to change those routes at all, until some
configured time has passed.

For a dual ToR setup, LLGR is helpful, as explained in more detail by [this
blog](https://vincent.bernat.ch/en/blog/2018-bgp-llgr), because:

-  If a link fails somewhere, the immediate preference lowering allows traffic to adjust
   immediately to use the other connectivity plane.

-  If a node is restarted, we still get the traditional Graceful Restart behaviour whereby
   routes to that node persist in the rest of the network.

### How to

Here are the steps you will need to successfully deploy a Kubernetes cluster with
{{site.prodname}} across multiple racks with dual plane connectivity:

- [Decide your IP addressing scheme](#decide-your-ip-addressing-scheme)
- [Decide your ECMP usage policy](#decide-your-ecmp-usage-policy)
- [Boot cluster nodes with those addresses](#boot-cluster-nodes-with-those-addresses)
- [Define bootstrap routes for reaching other loopback addresses](#define-bootstrap-routes-for-reaching-other-loopback-addresses)
- [Install Kubernetes and {{site.prodname}}](#install-kubernetes-and-tigera-secure-ee)
- [Configure {{site.prodname}} to peer with ToR routers](#configure-tigera-secure-ee-to-peer-with-tor-routers)
- [Configure your ToR routers and infrastructure](#configure-your-tor-routers-and-infrastructure)
- [Complete {{site.prodname}} installation](#complete-tigera-secure-ee-installation)
- [Configure {{site.prodname}} to advertise loopback addresses](#configure-tigera-secure-ee-to-advertise-loopback-addresses)
- [Verify the deployment](#verify-the-deployment)

The precise details will likely differ for any specific deployment.  For example, you may
use a Kubernetes installer that insists on booting and provisioning all the nodes as part
of a single overall "install Kubernetes" step; in that case you will need to work out how
to instruct the installer to provision the node addresses and bootstrap routes at the
right time.  But the steps presented here should be a complete and accurate description of
what is needed in principle.

#### Decide your IP addressing scheme

You will need an IP addressing scheme for the cluster nodes' IP addresses, including

-  their interface-specific addresses, for each connectivity plane

-  their loopback addresses;

and also for the ToR IP addresses.

For the example here, we use 172.31.X.Y, where:

-  X = 10 * `RACK_NUMBER` + `PLANE_NUMBER`

-  Y = `NODE_NUMBER` (within rack)

-  Loopback addresses have `PLANE_NUMBER` = 0.

-  For the ToR router in each plane, we use `NODE_NUMBER` = 250.

So for the first node in rack A, for example,

-  its loopback address is 172.31.10.1

-  its interface-address on the NIC attached to plane 1 is 172.31.11.1, with default
   gateway 172.31.11.250

-  its interface-address on the NIC attached to plane 2 is 172.31.12.1, with default
   gateway 172.31.12.250.

#### Decide your ECMP usage policy

The details and available options for how Linux uses ECMP routes have [historically varied
with kernel version](https://cumulusnetworks.com/blog/celebrating-ecmp-part-two/).

We recommend using a 4.17 or later kernel with `fib_multipath_hash_policy = 1`.  That
gives load-balancing per-flow for both IPv4 and IPv6, based on a hash of the source and
destination IPs and port numbers; which for general traffic patterns should give roughly
even usage of the available connectivity planes.

#### Boot cluster nodes with those addresses

Your address provisioning will be deployment-dependent - for example, you might use DHCP
to provision the interface-specific addresses - but there are some key rules for the
loopback addresses to work correctly:

-  The interface-specific addresses should be defined as `scope link`.  For example:

       ip address add 172.31.41.1/24 dev eth0 scope link

-  The loopback address should be defined on the "lo" device.  For example:

       ip address add 172.31.40.1/32 dev lo

It can also work to define the loopback address on a Linux dummy device, but only if there
are no other "scope global" addresses anywhere.

The effect of following those rules is that when Linux has to choose the source IP for an
outgoing connection, it will choose the loopback address.

#### Define bootstrap routes for reaching other loopback addresses

Once the cluster is fully installed and operating normally, {{site.prodname}} itself will
dynamically advertise each node's loopback address to the other nodes in the cluster.
However we also need those addresses to be reachable *during* the cluster installation,
and later when cluster nodes are restarted or down for maintenance.  Therefore, on each
node, you should program a static route like this to reach the loopback addresses of other
nodes in the same rack:

    ip route add 172.31.10.0/24 nexthop dev eth0 nexthop dev eth1

and static routes like this for the loopback addresses of nodes in each other rack (here,
rack B):

    ip route add 172.31.20.0/24 nexthop via 172.31.11.250 nexthop via 172.31.12.250

You should also configure loopback address routes on the ToR routers, and on any
intermediate infrastructure routers; details for that will be deployment-dependent.

Once that is done, each node should be able to ping the loopback address of any other
node, and `ip route` should show two ECMP routes to any other loopback address.  If you
somehow break one of the connectivity planes, the ping should still work by using the
other plane.

#### Install Kubernetes and {{site.prodname}}

Now you can follow your preferred method for deploying Kubernetes, and [our documentation
for installing {{site.prodname}}]({{site.baseurl}}/{{page.version}}/getting-started).

> **Note**: {{site.prodname}} installs by default with
> [IP-in-IP]({{site.baseurl}}/{{page.version}}/networking/vxlan-ipip) enabled, but for
> on-prem deployments as imagined here IP-in-IP is not needed and should be disabled.
> With our [operator-based
> install]({{site.baseurl}}/{{page.version}}/getting-started/kubernetes), you can do that
> by setting `encapsulation: None`; see the [install
> reference]({{site.baseurl}}/{{page.version}}/reference/installation/api) for details.
{: .alert .alert-info}

When you reach the point of configuring a Tigera-specific resource - typically, the
license key - you may see that fail.  If that happens, the explanation is that the various
components of {{site.prodname}} have been scheduled to nodes that are split across
different racks, and we don't yet have a working data path between pods running in
different racks.

> **Note**: To be more precise: at this point, we have a working data path between any
> components that are running on the nodes with host networking - i.e. using the nodes'
> loopback addresses - but not yet where the source or destination is a non-host-networked
> pod - i.e. using an IP from the pod CIDR range.  In particular, the Tigera API server
> runs as a non-host-networked pod, and we do not yet have connectivity between it and the
> Kubernetes API server, if those have been scheduled to nodes in different racks.
{: .alert .alert-info}

If you see this problem, the solution is to defer the rest of the {{site.prodname}}
installation for now and move on to the next two steps here, which will establish the
missing connectivity.

If you do not see this problem, that's OK too; it means the relevant {{site.prodname}}
components have not been scheduled in a problematic way.  Continue on to the next two
steps here, which are still needed for the cluster to provide connectivity between all
future pods.

#### Configure {{site.prodname}} to peer with ToR routers

Now, in principle, you should [configure BGPPeer resources](bgp) to tell each
{{site.prodname}} node to peer with the ToR routers for its rack, with the following field
settings.

-  `sourceAddress: None` to allow Linux to choose different interface-specific source
   addresses for the BGP sessions to the two ToRs.

   > **Note**: If there is a loss of connectivity between the node and a ToR router, we
   > specifically *want* the corresponding BGP session to fail, because that will trigger
   > removing or deprioritising the routes that were learnt over that BGP
   > session.
   {: .alert .alert-info}

-  `failureDetectionMode: BFDIfDirectlyConnected` to enable BFD, when possible, for fast
   failure detection.

   > **Note**: {{site.prodname}} only supports BFD on directly connected peerings, but in
   > practice nodes are normally directly connected to their ToRs.
   {: .alert .alert-info}

-  `restartMode: LongLivedGracefulRestart` to enable LLGR handling when the node needs to
   be restarted, if your ToR routers support LLGR.  If not, we recommend instead
   `maxRestartTime: 10s`.

-  `birdGatewayMode: DirectIfDirectlyConnected` to enable the "direct" next hop algorithm,
   if that is helpful for optimal interworking with your ToR routers.

   > **Note**: For directly connected BGP peerings, BIRD provides two gateway computation
   > modes, ["direct" and
   > "recursive"](https://bird.network.cz/?get_doc&v=16&f=bird-6.html#ss6.3).  "recursive"
   > is the default, but "direct" can give better results when the ToR also acts as the
   > route reflector (RR) for the rack.
   >
   > Specifically, a combined ToR/RR should ideally keep the BGP next hop intact (aka
   > "next hop keep") when reflecting routes from other nodes in the same rack, but add
   > itself as the BGP next hop (aka "next hop self") when forwarding routes from outside
   > the rack.  If your ToRs can be configured to do that, fine.
   >
   > If they cannot, an effective workaround is to configure the ToRs to do "next hop
   > keep" for all routes, with "gateway direct" on the {{site.prodname}} nodes.  In
   > effect the “gateway direct” applies a “next hop self” when needed, but otherwise not.
   {: .alert .alert-info}

To do that, you will need to:

1. Install and configure
   [calicoctl]({{site.baseurl}}/{{page.version}}/getting-started/calicoctl/install).

2. Set the correct AS number on each {{site.prodname}} node.

3. Label each {{site.prodname}} node, if your BGPPeer configuration uses labels.

4. Configure BGPPeer resources for the desired BGP peerings between each {{site.prodname}}
   node and its ToR routers.

Your details will be deployment-specific, but here we show those steps for our example
cluster and addressing scheme, following an [AS per
rack]({{site.baseurl}}/{{page.version}}/networking/design/l3-interconnect-fabric#the-as-per-rack-model)
model with AS 65001 for the first rack, 65002 for the second, and so on.

To set the correct AS number of each {{site.prodname}} node:

```
# For nodes in rack A:
calicoctl patch node <name> -p '{"spec":{"bgp": {"asNumber": "65001"}}}'
...
# For nodes in rack B:
calicoctl patch node <name> -p '{"spec":{"bgp": {"asNumber": "65002"}}}'
...
```

To give each {{site.prodname}} node a label for its rack:

```
# For nodes in rack A:
kubectl label node <name> rack=ra
...
# For nodes in rack B:
kubectl label node <name> rack=rb
...
```

To configure the node to ToR peerings for rack A:

```
calicoctl apply -f - <<EOF
apiVersion: projectcalico.org/v3
kind: BGPPeer
metadata:
  name: ra1
spec:
  nodeSelector: "rack == 'ra'"
  peerIP: 172.31.11.250
  asNumber: 65001
  sourceAddress: None
  failureDetectionMode: BFDIfDirectlyConnected
  restartMode: LongLivedGracefulRestart
  birdGatewayMode: DirectIfDirectlyConnected
---
apiVersion: projectcalico.org/v3
kind: BGPPeer
metadata:
  name: ra2
spec:
  nodeSelector: "rack == 'ra'"
  peerIP: 172.31.12.250
  asNumber: 65001
  sourceAddress: None
  failureDetectionMode: BFDIfDirectlyConnected
  restartMode: LongLivedGracefulRestart
  birdGatewayMode: DirectIfDirectlyConnected
EOF
```

Similarly for rack B:

```
calicoctl apply -f - <<EOF
apiVersion: projectcalico.org/v3
kind: BGPPeer
metadata:
  name: rb1
spec:
  nodeSelector: "rack == 'rb'"
  peerIP: 172.31.21.250
  asNumber: 65002
  sourceAddress: None
  failureDetectionMode: BFDIfDirectlyConnected
  restartMode: LongLivedGracefulRestart
  birdGatewayMode: DirectIfDirectlyConnected
---
apiVersion: projectcalico.org/v3
kind: BGPPeer
metadata:
  name: rb2
spec:
  nodeSelector: "rack == 'rb'"
  peerIP: 172.31.22.250
  asNumber: 65002
  sourceAddress: None
  failureDetectionMode: BFDIfDirectlyConnected
  restartMode: LongLivedGracefulRestart
  birdGatewayMode: DirectIfDirectlyConnected
EOF
```

And so on for the other racks.

Once BGPPeer resources have been configured, you should [disable the full node-to-node
mesh](bgp#disabling-the-full-node-to-node-bgp-mesh):

```
calicoctl apply -f - <<EOF
apiVersion: projectcalico.org/v3
kind: BGPConfiguration
metadata:
  name: default
spec:
  nodeToNodeMeshEnabled: false
EOF
```

#### Configure your ToR routers and infrastructure

You should configure your ToR routers to accept all the BGP peerings from
{{site.prodname}} nodes, to reflect routes between the nodes in that rack, and to
propagate routes between the ToR routers in different racks.  In addition we recommend
consideration of the following points.

BFD should be enabled if possible on all BGP sessions - both to the {{site.prodname}}
nodes, and between racks in your core infrastructure - so that a break in connectivity
anywhere can be rapidly detected.  The handling should be to initiate LLGR procedures if
possible, or else terminate the BGP session non-gracefully.

LLGR should be enabled if possible on all BGP sessions - again, both to the
{{site.prodname}} nodes, and between racks in your core infrastructure.  Traditional BGP
graceful restart should not be used, because this will delay the cluster's response to a
break in one of the connectivity planes.

#### Complete {{site.prodname}} installation

If you didn't complete the {{site.prodname}} installation above, return to that and retry
the step that failed.  It should now succeed.  Then do any remaining steps of the
installation.

#### Configure {{site.prodname}} to advertise loopback addresses

Although we already configured static /24 routes for the loopback address, for
bootstrapping, we want BGP to advertise, dynamically, the true reachability of each
loopback address.  For example, if one of the connectivity planes is broken, the static
routing may still - wrongly - show ECMP routes to a loopback address via both planes.

To get the correct dynamic advertisement, configure the loopback address CIDRs as disabled
{{site.prodname}} IP pools.  For example, for the loopback CIDRs for racks A and B:

```
kubectl create -f -<<EOF
apiVersion: projectcalico.org/v3
kind: IPPool
metadata:
  name: loopbacks-rack-a
spec:
  cidr: 172.31.10.0/24
  disabled: True
---
apiVersion: projectcalico.org/v3
kind: IPPool
metadata:
  name: loopbacks-rack-b
spec:
  cidr: 172.31.20.0/24
  disabled: True
EOF
```

What happens then is:

-  {{site.prodname}}, on each node, advertises a /32 route for that node's loopback
   address.

-  That /32 route is propagated dynamically through the BGP network, according as the dual
   connectivity planes are or are not working.

-  On each other cluster node, the result is a /32 route that is - correctly - ECMP if
   both planes are working, or a single path route if one of the planes is broken.

-  The static /24 routes are still present as well, but are ignored because of the /32
   routes being more specific.

#### Verify the deployment

If you examine traffic and connections within the cluster - for example, using `ss` or
`tcpdump` - you should see that all connections use loopback IP addresses or pod CIDR IPs
as their source and destination.  For example:

-  The kubelet on each node connecting to the API server.

-  The API server's connection to its backing etcd database, and peer connections between
   the etcd cluster members.

-  Pod connections that involve an SNAT or MASQUERADE in the data path, as can be the case
   when connecting to a Service through a cluster IP or NodePort.  At the point of the
   SNAT or MASQUERADE, a loopback IP address should be used.

-  Direct connections between pod IPs on different nodes.

The only connections using interface-specific addresses should be BGP.

If you look at the Linux routing table on any cluster node, you should see ECMP routes
like this to the loopback address of every other node in other racks:

```
172.31.20.4/32
   nexthop via 172.31.11.250 dev eth0
   nexthop via 172.31.12.250 dev eth1
```
and like this to the loopback address of every other node in the same rack:

```
172.31.10.4/32
   nexthop dev eth0
   nexthop dev eth1
```

If you launch some pods in the cluster, you should see ECMP routes for the /26 IP blocks
for the nodes where those pods were scheduled, like this:

```
10.244.192.128/26
   nexthop via 172.31.11.250 dev eth0
   nexthop via 172.31.12.250 dev eth1
```

If you do something to break the connectivity between racks of one of the planes, you
should see, within only a few seconds, that the affected routes change to have a single
path only, via the plane that is still unbroken; for example:

```
172.31.20.4/32 via 172.31.12.250 dev eth1`
10.244.192.128/26 via 172.31.12.250 dev eth1
```

When the connectivity break is repaired, those routes should change to become ECMP again.
