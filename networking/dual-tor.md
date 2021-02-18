---
title: Deploy a dual ToR cluster
description: Configure a dual plane cluster for redundant connectivity between workloads.
---

### Big picture

Deploy a dual plane cluster to provide redundant connectivity between your workloads for on-premises deployments.

>**Note**: Dual ToR is not supported if you are using BGP with encapsulation (VXLAN or IP-in-IP).
{: .alert .alert-info}

### Value

A dual plane cluster provides two independent planes of connectivity between all cluster
nodes.  If a link or software component breaks somewhere in one of those planes, cluster
nodes can still communicate over the other plane, and the cluster as a whole continues to
operate normally.

### Features

This how-to guide uses the following features:

**{{site.nodecontainer}}** run as a container, pre-Kubernetes

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
please see [this blog](https://cumulusnetworks.com/blog/celebrating-ecmp-part-two/){:target="_blank"}.

#### Loopback IP addresses

Loopback IP addresses are IP addresses which are assigned to a loopback interface on the
node.  Despite what the name might suggest, loopback IP addresses can be used for sending
and receiving data to and from other nodes.  They help to provide redundant networking,
because using a loopback address avoids tying traffic to a particular interface and, in a
dual ToR setup, allows that traffic to continue flowing even if a particular interface
goes down.

#### BFD

Bidirectional Forwarding Detection (BFD) is [a
protocol](https://tools.ietf.org/html/rfc5880){:target="_blank"} that detects very quickly when forwarding
along a particular path stops working - whether that's because a link has broken
somewhere, or some software component along the path.

In a dual ToR setup, rapid failure detection is important so that traffic flows within the
cluster can quickly adjust to using the other available connectivity plane.

#### Long lived graceful restart

Long Lived Graceful Restart (LLGR) is [an extension for
BGP](https://tools.ietf.org/html/draft-uttaro-idr-bgp-persistence-05){:target="_blank"} that handles link
failure by lowering the preference of routes over that link.  This is a compromise between
the base BGP behaviour - which is immediately to remove those routes - and traditional BGP
Graceful Restart behaviour - which is not to change those routes at all, until some
configured time has passed.

For a dual ToR setup, LLGR is helpful, as explained in more detail by [this
blog](https://vincent.bernat.ch/en/blog/2018-bgp-llgr){:target="_blank"}, because:

-  If a link fails somewhere, the immediate preference lowering allows traffic to adjust
   immediately to use the other connectivity plane.

-  If a node is restarted, we still get the traditional Graceful Restart behaviour whereby
   routes to that node persist in the rest of the network.

### How to

#### Prepare YAML resources describing the layout of your cluster

Prepare BGPPeer resources to specify how each node in your cluster should peer with the
ToR routers in its rack.  For example, if your rack 'A' has ToRs with IPs 172.31.11.100
and 172.31.12.100 and the rack AS number is 65001:


This step is not strictly specific to dual ToR deployment.  Any multi-rack deployment
needs to peer cluster nodes with the ToR or ToRs for their rack, so that pods hosted by
those nodes can be reached from elsewhere.  But the timing of the configuration is more
critical here, because dual ToR relies on BGP to advertise stable **node** addresses - for
dual-homed nodes - as well as pod IPs.  It's also more complex because of dual-homed nodes
having two ToR peers.  We recommend the following approach to ensure that BGP
configuration is in place at the right time.

Suppose a rack has two ToRs with IPs .  Dual-homed nodes should peer with both, and single-homed nodes with only
the first.  That is generically expressed by these BGPPeer resources:

```
apiVersion: projectcalico.org/v3
kind: BGPPeer
metadata:
  name: ra1
spec:
  nodeSelector: "rack == 'ra' || rack == 'ra_single'"
  peerIP: 172.31.11.100
  asNumber: 65001
  sourceAddress: None
---
apiVersion: projectcalico.org/v3
kind: BGPPeer
metadata:
  name: ra2
spec:
  nodeSelector: "rack == 'ra'"
  peerIP: 172.31.12.100
  asNumber: 65001
  sourceAddress: None
EOF
```

Then each dual-homed node in the rack should be labelled with `rack: ra`, and each
single-homed node with `rack: ra_single`.  Also every node in the rack must have a
`projectcalico.org/ASNumber: 65001` annotation so that it knows its own AS number.

Then repeat all that for further racks: with `rb`, `rb_single`, `65002`, `rc`,
`rc_single`, `65003`, and so on.

Timing-wise, the configuration relevant to any given dual-homed node should be defined
before that node is added to the cluster, and this is easily achieved in most Kubernetes
platforms because they allow a Node object to exist in advance.  Therefore, as soon as the
Kubernetes API is available, but before installing Calico:

-  use [calicoctl]({{site.baseurl}}/maintenance/clis/calicoctl/install) to create all
   BGPPeer resources, following the pattern above

-  use `kubectl` to create a Node resource for each node that will be added to cluster,
   labelled and annotated with its rack and AS number; for example:

   ```
   kubectl create -f - <<EOF
   apiVersion: v1
   kind: Node
   metadata:
     name: worker1
     labels:
       rack: ra
     annotations:
       projectcalico.org/ASNumber: "65001"
   EOF
   ```

Once BGPPeer resources have been configured, you should [disable the full node-to-node
mesh](bgp#disable-the-default-bgp-node-to-node-mesh):

```
calicoctl patch bgpconfig default -p '{"spec":{"nodeToNodeMeshEnabled": "false"}}'
```

Depending on what your ToR supports, consider also setting these fields in each BGPPeer:

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
   > "recursive"](https://bird.network.cz/?get_doc&v=16&f=bird-6.html#ss6.3){:target="_blank"}.  "recursive"
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




  - Nodes: AS number, label to select BGPPeers
  - BGPPeers: peerings from nodes to ToRs
  - BGPConfiguration: disable full mesh
  - IPPools: stable addresses, the default IP pool for pods
  - EarlyNetworkConfiguration: peerings from nodes to ToRs, stable addresses

- Dual-ToR network environment + BGP config on ToRs and core routers.

Non-OpenShift

- Arrange cnx-node run on each node.  Reboot.

- kubeadm init and join

- Run Tigera operator

OpenShift

- Add MachineConfig to arrange cnx-node run on each node.

- Run OpenShift install

So inputs:

- YAML
- Nodes: { node name, rack label (or IPs to peer to), AS number }
- BGPPeers
- IPPools

#### Arrange for dual-homed nodes to run {{site.nodecontainer}} on each boot

{{site.prodname}}'s {{site.nodecontainer}} image normally runs as a Kubernetes pod, but
for dual ToR setup it should also run as a container after each boot of a dual-homed node.
For example:

```
podman run --privileged --net=host \
    -v /calico-dual-tor:/calico-dual-tor -e CALICO_DUAL_TOR=/calico-dual-tor/details.sh \
    {{page.registry}}{{site.imageNames["node"]}}:latest
```

When a dual-homed node first boots (and then each time it reboots) this container does the
pre-Kubernetes setup needed for dual ToR operation, namely:

- provisioning a stable loopback address, and ensuring that this address is used for all
  connections to and from the node

- starting BGP, with peerings to the node's ToRs, to advertise the stable address to the
  network, so that other nodes can route to this one.

> **Note**: This must happen *before* any Kubernetes components start running on the node,
> because we want Kubernetes connections to use the stable address.
{: .alert .alert-info}

The container needs a **custom file** - describing deployment-specific IP addressing and
AS numbering - to be mapped into the container and identified by the `CALICO_DUAL_TOR`
environment variable.  The requirements for this file are described in the next section.

Exactly **how** to arrange for this container to run will depend on your platform's
workflow for adding a node to the cluster.

-  If the workflow allows intervention before Kubernetes starts installing on the new
   node, you can create a service to run the container, enabled to run on subsequent
   boots.  For example, as a systemd unit:

   ```
   [Service]
   ExecStartPre=-/bin/podman rm -f calico-dual-tor
   ExecStartPre=/bin/mkdir -p /etc/calico-dual-tor
   ExecStartPre=/bin/curl -o /etc/calico-dual-tor/details.sh http://172.31.1.1:8080/calico-dual-tor/details-map.sh
   ExecStart=/bin/podman run --rm --privileged --net=host --name=calico-dual-tor -v /etc/calico-dual-tor:/etc/calico-dual-tor -e CALICO_DUAL_TOR=/etc/calico-dual-tor/details.sh {{page.registry}}{{site.imageNames["node"]}}:latest
   [Install]
   WantedBy=multi-user.target
   ```

   This example also shows how you could download the custom file for your deployment from
   a central location.

   Then reboot, so that the dual ToR setup happens, and then allow Kubernetes installation
   to continue.

-  If the workflow does not allow, the platform may have an abstraction for achieving the
   same thing.  For example, OpenShift's `MachineConfig` API can be used to specify files
   and a systemd unit (as above) to be installed and enabled on each new node.

#### Describe your IP addressing and AS numbering

On each boot, the available information for the node is its per-interface IP addresses
(either statically configured, or obtained from DHCP), and based on those
{{site.nodecontainer}} needs to be told:

- what stable address to provision for the node

- the IP addresses of the ToRs to peer with

- the AS number for the node and its ToRs.

You must provide this information in the form of shell code like this:

```
details_are()
{
    echo "DUAL_TOR_STABLE_ADDRESS=$1"
    echo "DUAL_TOR_AS_NUMBER=$2"
    echo "DUAL_TOR_PEERING_ADDRESS_1=$3"
    echo "DUAL_TOR_PEERING_ADDRESS_2=$4"
}

get_dual_tor_details()
{
    case $1 in

    172.31.21.1 | 172.31.22.1 )
        details_are 172.31.20.1 65002 172.31.21.100 172.31.22.100
        ;;
    172.31.21.2 | 172.31.22.2 )
        details_are 172.31.20.2 65002 172.31.21.100 172.31.22.100
        ;;
    172.31.21.3 | 172.31.22.3 )
        details_are 172.31.20.3 65002 172.31.21.100 172.31.22.100
        ;;
    172.31.21.4 | 172.31.22.4 )
        details_are 172.31.20.4 65002 172.31.21.100 172.31.22.100
        ;;
    172.31.21.5 | 172.31.22.5 )
        details_are 172.31.20.5 65002 172.31.21.100 172.31.22.100
        ;;
    172.31.11.1 | 172.31.12.1 )
        details_are 172.31.10.1 65001 172.31.11.100 172.31.12.100
        ;;
    172.31.11.2 | 172.31.12.2 )
        details_are 172.31.10.2 65001 172.31.11.100 172.31.12.100
        ;;
    172.31.11.3 | 172.31.12.3 )
        details_are 172.31.10.3 65001 172.31.11.100 172.31.12.100
        ;;
    172.31.11.4 | 172.31.12.4 )
        details_are 172.31.10.4 65001 172.31.11.100 172.31.12.100
        ;;
    172.31.11.5 | 172.31.12.5 )
        details_are 172.31.10.5 65001 172.31.11.100 172.31.12.100
        ;;

    esac
}
```

{{site.nodecontainer}} calls `get_dual_tor_details <IP>`, with `<IP>` being one of the
interface-specific addresses, and the function must print the needed information in the
form:

```
DUAL_TOR_STABLE_ADDRESS=172.31.10.5
DUAL_TOR_AS_NUMBER=65001
DUAL_TOR_PEERING_ADDRESS_1=172.31.11.100
DUAL_TOR_PEERING_ADDRESS_2=172.31.12.100
```

Therefore you need to decide upfront how your dual-homed nodes will split across the racks
in your deployment, and hence which ToRs they will peer with.  Or alternatively, devise an
algorithmic scheme such that the correct AS number, stable address and ToRs addresses can
be computed mathematically, for any dual-homed node, from either of its interface-specific
addresses.  Then encode that information in your version of `get_dual_tor_details`, and
host that somewhere that is accessible from each new node as it boots, so that it can be
fed into the {{site.nodecontainer}} container, as illustrated by the example in the
previous section.

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

#### Install Kubernetes and {{site.prodname}}

1. Follow your preferred method for deploying Kubernetes, and [installing {{site.prodname}} on-premises]({{site.baseurl}}/getting-started).

1. During {{site.prodname}} installation, disable the default encapsulation setting, IP-in-IP.

   Encapsulation is not supported and must be disabled. Set `encapsulation: None`. For help, see the [Installation reference]({{site.baseurl}}/reference/installation/api).

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
