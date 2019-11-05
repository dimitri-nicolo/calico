---
title: Deploying a dual ToR cluster
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

#### Dual ToR (Top of Rack) routing

There are many ways to connect {{site.prodname}} nodes to an infrastructure fabric, such
that the fabric propagates the workload routes from each node to all of the other nodes.
For the purposes of this guide we cut that possibility space down a little by making three
basic assumptions:

1. There is a BGP speaker per rack, that all of the nodes in the rack peer with, and that
   provides connection to other racks and to the outside world.  We call this the Top of
   Rack (ToR) router.

2. In fact - because this guide is about _dual ToR_ connectivity - there are two such
   speakers per rack, and each node has two fabric-facing interfaces, each of which
   connects directly to one of those ToR routers.  The connectivity from all of the nodes
   to the first ToR router (via each node's first NIC) should be independent of the
   connectivity from all of the nodes to the second ToR router (via each node's second
   NIC).

3. The ToR routers for multiple racks are somehow connected to each other such that if a
   single link or software component fails anywhere, there is still an alternative working
   path between any two cluster nodes; in other words there are two independent planes of
   connectivity between any two nodes.

We don't assume any further details about how the ToRs are connected to each other, or how
the cluster nodes are connected to the ToRs in their rack.  For example, within the rack
there could be two independent layer 2 networks, or there could be point-to-point links
between each node and each ToR.

Here's an example of how a dual ToR setup can look, showing just two racks and two nodes
in each rack.  In reality, the connections *between* racks would be more complex than
shown here, but that makes no difference to the support and setup that we need for the
{{site.prodname}} nodes in the lower half of the diagram.  The "client", "ra-server" and
"rb-server" boxes represent Kubernetes pods:

![dual-tor]({{site.baseurl}}/images/dual-tor.png)

#### Programming and using ECMP routes

{{site.prodname}} can learn multiple possible paths to a given destination or prefix - for
example, to a /26 block of pod IP addresses on a given cluster node.  When that happens,
as will be the case in a deployment with dual connectivity planes, the multiple paths are
programmed into the Linux routing table as ECMP routes.

Linux then decides how to balance traffic across the available paths, including whether
this is informed by TCP and UDP port numbers as well as source and destination IP
addresses, whether the decision is made per-packet, per-connection, or in some other way,
and so on; and the details here have varied with Linux kernel version.  For a clear
account of the exact options and behaviors for different kernel versions, please see [this
blog](https://cumulusnetworks.com/blog/celebrating-ecmp-part-two/).

We recommend using a 4.17 or later kernel with `fib_multipath_hash_policy = 1`.  That
gives load-balancing per-flow for both IPv4 and IPv6, based on a hash of the source and
destination IPs and port numbers; which for general traffic patterns should give roughly
even usage of the available connectivity planes.

#### Loopback IP addresses

We want to allow for either of a cluster node's links to fail at any time, and for traffic
to and from that node to continue flowing when that happens.  That means that IP data to
or from that node must not use a NIC-specific address as its source or destination IP.
(If it did, that address would become invalid when the associated link went down, and
further data packets would be unable to be routed to that address.)  Instead, the standard
solution is to define a "loopback" IP address for each node, not associated with either
NIC, and arrange that all connections between nodes use their loopback IP addresses as
source and destination, instead of NIC-specific addresses.

Note that in a Kubernetes cluster, the connections that need to use loopback IP addresses
include:

-  The kubelet on each node connecting to the API server.

-  The API server's connection to its backing etcd database, and peer connections between
   the etcd cluster members.

-  Pod connections that involve an SNAT or MASQUERADE in the data path, as can be the
   case when connecting to a Service through a cluster IP or NodePort.

-  Direct connections between pod IPs on different nodes.  Although these connections
   have pod IPs as their source and destination, they still rely on the source node
   having a route to the destination pod IP, and it is important that this route is via
   the loopback IP of the destination node, not via one of its interface-specific
   addresses.

Despite what the name might suggest, these loopback IP address must be routable from the
other nodes in the cluster.  They are called "loopback" only because they are often
provisioned through definition on the Linux loopback ("lo") interface.

#### NIC-specific addresses for BGP

There is one case where we still want to use NIC-specific addresses, for the
{{site.prodname}} node's BGP sessions to its ToR routers.  If there is a loss of
connectivity between the node and a ToR router, we specifically *want* the corresponding
BGP session to fail, because that will trigger removing or deprioritising the routes that
were learnt over that BGP session.

{{site.prodname}} incorporates BIRD as its BGP speaker, and in terms of BIRD config, we
can get that behaviour in a dual ToR setup by omitting the "source address" field that
{{site.prodname}} would normally specify.  There is now a `sourceAddress` field in the
`BGPPeer` resource for that purpose.  Setting `sourceAddress` to `None` allows [Linux to
compute the source address](http://linux-ip.net/html/routing-saddr-selection.html) for the
BGP session automatically, which in practice gives the local NIC-specific address in the
same subnet as the ToR router IP that we are peering with.

#### Fast link failure handling with BFD and LLGR

By default it takes 210s for a link failure to be notified to other BGP peers:

-  90s of BGP hold time - the time it takes for BGP to notice the loss of connectivity

-  120s of Graceful Restart time - the time allowed for graceful restart of BGP peers,
   before routes over the failed link are removed.

In a dual ToR deployment we want link failure to be detected and propagated as quickly as
possible, so that routes over the broken link can be withdrawn or deprioritised, and nodes
can react by switching to use routes via the other, still-working plane.

For fast detection, the solution is to enable BFD on the relevant BGP peerings.
[Bidirectional Forwarding Detection (BFD)](https://tools.ietf.org/html/rfc5880) is a
protocol that detects very quickly when forwarding along a particular path stops working -
whether that's because a link has broken somewhere, or some software component along the
path.  With BIRD it is trivial to enable BFD on directly-connected peerings, and dual ToR
peerings are in practice directly-connected.  There is now a `failureDetectionMode` field
in the `BGPPeer` resource, that enables this.

For quickly propagating the consequences of link failure to other routers, traditional BGP
graceful restart (GR) would be actively harmful: the whole point of GR is not to propagate
such consequences, so as to avoid route flapping in other routers in the network.  If we
want fast propagation and response in other routers, we should simply not configure GR.
However we do still want GR behaviour in one scenario: when the BIRD on a
{{site.prodname}} node is restarted for some reason like software update or config change.
In that scenario, we know that the BIRD daemon will be running again very shortly, and
have no reason to believe there is any actual data plane connectivity problem anywhere, so
we’d like the restart not to cause routing to change in the rest of the network.

[Long-lived graceful restart
(LLGR)](https://tools.ietf.org/html/draft-uttaro-idr-bgp-persistence-05) is the solution
here.  With LLGR, routers handle a link failure by immediately re-advertising the routes
over the link with a very low preference.  Then:

-  In the BIRD restart scenario, all of the routes to that BIRD are re-advertised with low
   preference, but remain available.  Because other routers have no other option for
   getting to that node, they will continue routing to it in the same way as before.  In
   comparison with GR in this scenario, LLGR does generate control plane churn - i.e. in
   BGP, and in Linux routing table updates - but, like with GR, there should be no change
   to how a given packet is routed through the network.

-  In the link failure scenario, the routes over that particular link are re-advertised
   with low preference.  In a dual ToR setup, each other router will then see that low
   preference route, and another route with normal preference that uses the other, working
   link to that node; and therefore traffic will use the route over the working link.

Therefore the `BGPPeer` resource now has a `restartMode` field that can be set to
`LongLivedGracefulRestart`, and we recommend that for a dual ToR deployment.

### How to

With the above concepts in mind, here are the steps you will need to bring up a dual ToR
deployment:

- [Decide your IP addressing scheme](#decide-your-ip-addressing-scheme)
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

You need to plan a formulaic IP addressing scheme for the cluster nodes' IP addresses,
including

-  their interface-specific addresses, for each connectivity plane

-  their loopback addresses.

For the example here, we use 172.31.X.Y, where:

-  X = 10 * `RACK_NUMBER` + `PLANE_NUMBER`

-  Y = `NODE_NUMBER` (within rack)

-  Loopback addresses have `PLANE_NUMBER` = 0.

-  For the ToR router in each plane, we use `NODE_NUMBER` = 250.

So for the first node in rack A, for example,

-  its loopback address would be 172.31.10.1

-  its interface-address on the NIC attached to plane 1 would be 172.31.11.1, with default
   gateway 172.31.11.250

-  its interface-address on the NIC attached to plane 2 would be 172.31.12.1, with default
   gateway 172.31.12.250.

#### Boot cluster nodes with those addresses

How you do this will be mostly deployment-dependent - for example, you might use DHCP to
provision the interface-specific addresses - but there are some key rules for the
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

Later, once the cluster is fully installed, {{site.prodname}} itself will handle
advertising each node's loopback address to the other nodes in the cluster.  However we
also need those addresses to be reachable *during* the cluster installation.  To achieve
that, each node needs static routes like this for the loopback addresses of other nodes in
the same rack:

    ip route add 172.31.10.0/24 nexthop dev eth0 nexthop dev eth1

and like this for the loopback addresses of nodes in each other rack (here, rack B):

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

However, when the method reaches the point of needing to configure a Tigera-specific
resource - typically, the license key - you may see that that fails.  If that happens, the
explanation for it is that the various components of {{site.prodname}} have been scheduled
to nodes that are split across different racks, and we don't yet have a working data path
between pods running in different racks.

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

-  `sourceAddress: None` to allow BIRD to choose the BGP peering source address
   automatically.

-  `failureDetectionMode: BFDIfDirectlyConnected` to enable BFD, when possible, for fast
   failure detection.

-  `restartMode: LongLivedGracefulRestart` to enable LLGR handling when the node needs to
   be restarted, if your ToR routers support LLGR.  If not, we recommend instead
   `maxRestartTime: 10s`.

-  `birdGatewayMode: DirectIfDirectlyConnected` to enable the "direct" next hop algorithm
   as described above, if that is helpful for optimal interworking with your ToR routers.

To do that, you will need to:

1. Install and configure
   [calicoctl]({{site.baseurl}}/{{page.version}}/getting-started/calicoctl/install).

2. Set the correct AS number on each {{site.prodname}} node.

3. Label each {{site.prodname}} node, if your BGPPeer configuration uses labels.

4. Configure BGPPeer resources for the desired BGP peerings between each {{site.prodname}}
   node and its ToR routers.

Your details will be deployment-specific, but here we show what that detail would be for
our example cluster and addressing scheme, following an [AS per
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
mesh](bgp#disabling-the-full-node-to-node-bgp-mesh) so that the BGPPeer configuration is
used instead:

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

You obviously need to configure your ToR routers to accept all the BGP peerings from
{{site.prodname}} nodes, and to configure whatever is needed to propagate routes between
the ToR routers in different racks.  In addition we recommend consideration of the
following points.

BFD should be enabled if possible on all BGP sessions - both to the {{site.prodname}}
nodes, and between racks in your core infrastructure - so that a break in connectivity
anywhere can be rapidly detected.  The handling should be to initiate LLGR procedures if
possible, or else terminate the BGP session non-gracefully.

LLGR should be enabled if possible on all BGP sessions - again, both to the
{{site.prodname}} nodes, and between racks in your core infrastructure.  Traditional BGP
graceful restart should not be used, because this will delay the cluster's response to a
break in one of the connectivity planes.

#### Complete {{site.prodname}} installation

If you didn't complete the {{site.prodname}} installation above, now return to that and
retry the step that failed.  It should now succeed.  Then do the remaining steps of the
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

-  The {{site.prodname}} BIRD daemon that runs on each node advertises a /32 route for
   that node's loopback address.

-  That /32 route is propagated dynamically through the BGP network, according as the dual
   connectivity planes are or are not working.

-  On each other cluster node, the result is a /32 route that is - correctly - ECMP if
   both planes are working, or a single path route if one of the planes is broken.

-  The static /24 routes are still present as well, but are ignored because of the /32
   routes being more specific.

If you prefer, you can now delete the static loopback routes.

#### Verify the deployment

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
