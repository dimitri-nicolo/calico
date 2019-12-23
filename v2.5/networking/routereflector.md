---
title: Route reflectors
canonical_url: 'https://docs.tigera.io/v2.3/usage/routereflector'
---

BGP route reflectors are useful in large scale deployments, to reduce the number of BGP
connections that are needed for correct and complete route propagation.  {{site.tseeprodname}}
includes optional route reflector function in the {{site.nodecontainer}} image, which is
enabled by provisioning the `spec.bgp.routeReflectorClusterID` field of the relevant [node
resource]({{site.url}}/{{page.version}}/reference/resources/node).

(simultaneously with their function as workload hosts).

To run a standalone route reflector outside the cluster, you can also use the
{{site.nodecontainer}} image.  Use [calicoctl node
run]({{site.url}}/{{page.version}}/reference/calicoctl/node/run) to run a
{{site.nodecontainer}} container, then modify the relevant node resource similarly as in the
in-cluster case.

> **Note**: The only difference between the 'in-cluster' and 'standalone' cases is that, in
> the 'standalone' case, the orchestrator is somehow instructed not to schedule any workloads
> onto the standalone route reflector nodes.
{: .alert .alert-info}

Of course there are many other ways to set up and run a non-{{site.tseeprodname}} route reflector
outside the cluster.  You then need to [configure some or all of the {{site.tseeprodname}} nodes
to peer with that route reflector]({{site.url}}/{{page.version}}/networking/bgp).

In addition the non-{{site.tseeprodname}} route reflector may need configuration to accept
peerings from the {{site.tseeprodname}} nodes, but in general that is outside the scope of this
documentation.  For example, if you installed [BIRD](https://bird.network.cz/) to be your
route reflector, you would need to configure BGP peerings like the following for each
{{site.tseeprodname}} node that you expect to connect to it.

    protocol bgp <node_shortname> {
      description "<node_ip>";
      local as <as_number>;
      neighbor <node_ip> as <as_number>;
      multihop;
      rr client;
      graceful restart;
      import all;
      export all;
    }
