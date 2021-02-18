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

1.  Prepare BGPPeer resources to specify how each node in your cluster should peer with
	the ToR routers in its rack.  For example, if your rack 'A' has ToRs with IPs
	172.31.11.100 and 172.31.12.100 and the rack AS number is 65001:

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

	> **Note**: The effect of the `nodeSelector` fields here is that any node with label
	> `rack: ra` will peer with both these ToRs, while any node with label `rack:
	> ra_single` will peer with only the first ToR.  For optimal dual ToR function and
	> resilience, nodes in rack 'A' should be labelled `rack: ra`, but `rack: ra_single`
	> can be used instead on any nodes which cannot be dual-homed.
	{: .alert .alert-info}

	Repeat for as many racks as there are in your cluster.  Each rack needs a new pair of
	BGPPeer resources with its own ToR addresses and AS number, and `nodeSelector` fields
	matching the nodes that should peer with its ToR routers.

	Depending on what your ToR supports, consider also setting these fields in each
    BGPPeer:

    -  `failureDetectionMode: BFDIfDirectlyConnected` to enable BFD, when possible, for
       fast failure detection.

       > **Note**: {{site.prodname}} only supports BFD on directly connected peerings, but
       > in practice nodes are normally directly connected to their ToRs.
       {: .alert .alert-info}

    -  `restartMode: LongLivedGracefulRestart` to enable LLGR handling when the node needs
       to be restarted, if your ToR routers support LLGR.  If not, we recommend instead
       `maxRestartTime: 10s`.

    -  `birdGatewayMode: DirectIfDirectlyConnected` to enable the "direct" next hop
       algorithm, if that is helpful for optimal interworking with your ToR routers.

       > **Note**: For directly connected BGP peerings, BIRD provides two gateway
       > computation modes, ["direct" and
       > "recursive"](https://bird.network.cz/?get_doc&v=16&f=bird-6.html#ss6.3){:target="_blank"}.
       > "recursive" is the default, but "direct" can give better results when the ToR
       > also acts as the route reflector (RR) for the rack.
       >
       > Specifically, a combined ToR/RR should ideally keep the BGP next hop intact (aka
       > "next hop keep") when reflecting routes from other nodes in the same rack, but
       > add itself as the BGP next hop (aka "next hop self") when forwarding routes from
       > outside the rack.  If your ToRs can be configured to do that, fine.
       >
       > If they cannot, an effective workaround is to configure the ToRs to do "next hop
       > keep" for all routes, with "gateway direct" on the {{site.prodname}} nodes.  In
       > effect the “gateway direct” applies a “next hop self” when needed, but otherwise
       > not.
       {: .alert .alert-info}

1.  Prepare Node resources to specify each node's AS number and `rack` peering label.  For
    example:

	```
	apiVersion: v1
    kind: Node
    metadata:
      name: worker1
      labels:
        rack: ra
      annotations:
        projectcalico.org/ASNumber: "65001"
    ---
    apiVersion: v1
    kind: Node
    metadata:
      name: worker2
      labels:
        rack: rb
      annotations:
        projectcalico.org/ASNumber: "65002"
    ```

	and so on.

    > **Note**: The `rack` label values here must match those in the BGPPeer resources, so
    > that each node peers with the right ToR routers.
	{: .alert .alert-info}

1.  Prepare this BGPConfiguration resource to [disable the full node-to-node
	mesh](bgp#disable-the-default-bgp-node-to-node-mesh):

	```
	apiVersion: projectcalico.org/v3
    kind: BGPConfiguration
    metadata:
      name: default
    spec:
      nodeToNodeMeshEnabled: false
    ```

1.  Prepare disabled IPPool resources for the CIDRs from which you will allocate stable
    addresses for dual-homed nodes.  For example, if the nodes in rack 'A' will have
    stable addresses from 172.31.10.0/24:

	```
	apiVersion: projectcalico.org/v3
    kind: IPPool
    metadata:
      name: ra-stable
    spec:
      cidr: 172.31.10.0/24
      disabled: true
      nodeSelector: all()
    ```

    If the next rack uses a different CIDR, define a similar IPPool for that rack, and so
    on.

    > **Note**: These IPPool definitions tell {{site.prodname}}'s BGP component to export
    > routes within the given CIDRs, which is essential for the core BGP infrastructure to
    > learn how to route to each stable address.  `disabled: true` tells {{site.prodname}}
    > *not* to use these CIDRs for pod IPs.
	{: .alert .alert-info}

1.  Prepare an enabled IPPool resource for your default CIDR for pod IPs.  For example:

	```
	apiVersion: projectcalico.org/v3
    kind: IPPool
    metadata:
      name: default-ipv4
    spec:
      cidr: 10.244.0.0/16
      nodeSelector: all()
    ```

    > **Note**: The CIDR must match what you specify elsewhere in the Kubernetes
    > installation.  For example, `networking.clusterNetwork.cidr` in OpenShift's install
    > config, or `--pod-network-cidr` with kubeadm.  You should not specify `ipipMode` or
    > `vxlanMode`, as these are incompatible with dual ToR operation.  `natOutgoing` can
    > be omitted, as here, if your core infrastructure will perform an SNAT for traffic
    > from pods to the Internet.
	{: .alert .alert-info}

1.  Combine the preceding `projectcalico/v3` resources - i.e. the BGPPeers,
    BGPConfiguration and IPPools - into a `calico-resources` ConfigMap for the Tigera
    operator.

1.  Prepare an EarlyNetworkConfiguration resource to provide the information that
    dual-homed nodes need for bootstrapping, each time that they boot.  For example, with
    IP addresses and AS numbers similar as for other resources above:

	```
	apiVersion: projectcalico.org/v3
    kind: EarlyNetworkConfiguration
    spec:
      nodes:
        # worker1
        - interfaceAddresses:
            - 172.31.11.3
            - 172.31.12.3
          stableAddress:
            address: 172.31.10.3
          asNumber: 65001
          peerings:
            - peerIP: 172.31.11.100
            - peerIP: 172.31.12.100
        # worker2
        - interfaceAddresses:
            - 172.31.21.4
            - 172.31.22.4
          stableAddress:
            address: 172.31.20.4
          asNumber: 65002
          peerings:
            - peerIP: 172.31.21.100
            - peerIP: 172.31.22.100
        ...
    ```

    > **Note**: Despite apparent similarity, this resource differs from all the others in
    > that it will be provided as a file on each newly provisioned node (whereas the other
    > resources are all served from the Kubernetes API).  It specifies the stable address
    > for each node.  It also repeats information that could be inferred from the
    > preceding resources, but that is because the information is needed for post-boot
    > setup on each node, at a point where the node cannot access the Kubernetes API.
	{: .alert .alert-info}

In summary you should now have:

-  Node resource YAML, specifying the AS number and `rack` peering label for each node

-  a `calico-resources` ConfigMap containing {{site.prodname}} BGPPeers, BGPConfiguration
   and IPPools

-  an EarlyNetworkConfiguration YAML.

#### Arrange for dual-homed nodes to run {{site.nodecontainer}} on each boot

{{site.prodname}}'s {{site.nodecontainer}} image normally runs as a Kubernetes pod, but
for dual ToR setup it must also run as a container after each boot of a dual-homed node.
For example:

```
podman run --privileged --net=host \
    -v /calico-early:/calico-early -e CALICO_EARLY_NETWORKING=/calico-early/cfg.yaml \
    {{page.registry}}{{site.imageNames["node"]}}:latest
```

The environment variable `CALICO_EARLY_NETWORKING` must point to the
EarlyNetworkConfiguration prepared above, so that EarlyNetworkConfiguration YAML must be
copied into a file on the node (here, `/calico-early/cfg.yaml`) and mapped into the
{{site.nodecontainer}} container.

> **Note**: Based on the EarlyNetworkConfiguration YAML, this container will:
>
> - identify the right part of the YAML for *this* node
>
> - provision the stable address for the node, and ensure that the stable address is used
>   for all subsequent connections to and from the node
>
> - start BGP, with peerings to the node's ToRs, to advertise the stable address to the
>   network, so that other nodes can route to this one.
>
> It's important that this all happens *before* any Kubernetes components start running on
> the node, because we want Kubernetes connections to use the stable address.
{: .alert .alert-info}

Exactly **how** to arrange for this container to run will depend on your platform's
workflow for adding a node to the cluster.

-  If the workflow allows intervention before Kubernetes starts installing on the new
   node, you can create a service to run the container, enabled to run on subsequent
   boots.  For example, as a systemd unit:

   ```
   [Service]
   ExecStartPre=-/bin/podman rm -f calico-early
   ExecStartPre=/bin/mkdir -p /calico-early
   ExecStartPre=/bin/curl -o /calico-early/cfg.yaml http://172.31.1.1:8080/calico-early/cfg.yaml
   ExecStart=/bin/podman run --privileged --net=host --name=calico-early -v /calico-early:/calico-early -e CALICO_EARLY_NETWORKING=/calico-early/cfg.yaml {{page.registry}}{{site.imageNames["node"]}}:latest
   [Install]
   WantedBy=multi-user.target
   ```

   This example also shows how you could serve the EarlyNetworkConfiguration for your
   deployment from a central location.

   Then reboot, so that the dual ToR setup happens, and then allow Kubernetes installation
   to continue.

-  If the workflow does not allow, the platform may have an abstraction for achieving the
   same thing.  For example, OpenShift's `MachineConfig` API can be used to specify files
   and a systemd unit (as above) to be installed and enabled on each new node.

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

For your planned Kubernetes installer, work out:

-  how to inject the Node resources (prepared above) as soon as possible after the
   Kubernetes API is first available

-  how to inject the `calico-resources` ConfigMap when the installer is about to start the
   Tigera operator (if the installer does this automatically).

By way of example:

-  With [OpenShift]({{site.baseurl}}/getting-started/openshift), simply add all the Node
   resources - with one Node resource per file - and the `calico-resources` ConfigMap into
   the manifests directory just before running `openshift-install create
   ignition-configs`.

   > **Note**: The prepared `calico-resources` ConfigMap should replace [our default empty
   > one]({{site.baseurl}}/manifests/ocp/tigera-operator/02-configmap-calico-resources.yaml).
   {: .alert .alert-info}

-  With kubeadm, `kubeadm init` will necessarily create the first node at the same time as
   bringing up the Kubernetes API.  In this case, therefore, you should add the
   appropriate `rack` label and AS number to the first node, and reboot it so that those
   can take effect.

   > **Note**: As already covered above, you should by now have arranged for the
   > {{site.nodecontainer}} image to run on each boot, on each node.
   {: .alert .alert-info}

   Then create the other Node resources, and continue with `kubeadm join` for those other
   nodes.

   Then, just before creating the Installation resource to kick off installing
   {{site.prodname}}, use `kubectl apply` to apply the `calico-resources` ConfigMap.

Now proceed with the installation, and the required dual ToR setup will be performed
automatically based on the above resources.

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
