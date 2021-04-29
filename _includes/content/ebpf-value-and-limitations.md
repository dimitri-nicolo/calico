### Value

The eBPF dataplane mode has several advantages over standard linux networking pipeline mode:

* It scales to higher throughput.
* It uses less CPU per GBit.
* It has native support for Kubernetes services (without needing kube-proxy) that:

    * Reduces first packet latency for packets to services.
    * Preserves external client source IP addresses all the way to the pod.
    * Supports DSR (Direct Server Return) for more efficient service routing.
    * Uses less CPU than kube-proxy to keep the dataplane in sync.

To learn more and see performance metrics from our test environment, see the blog, {% include open-new-window.html text='Introducing the Calico eBPF dataplane' url='https://www.projectcalico.org/introducing-the-calico-ebpf-dataplane/' %}.

### Limitations

eBPF mode currently has some limitations relative to the standard Linux pipeline mode:

- eBPF mode only supports x86-64.  (The eBPF programs are not currently built for the other platforms.)
- eBPF mode does not yet support IPv6.
- eBPF mode does not yet support host endpoint `doNotTrack` policy (but it does support normal, pre-DNAT and apply-on-forward policy for host endpoints).
- When enabling eBPF mode, pre-existing connections continue to use the non-BPF datapath; such connections should not be disrupted, but they do not benefit from eBPF mode's advantages.
- Disabling eBPF mode _is_ disruptive; connections that were handled through the eBPF dataplane may be broken and services that do not detect and recover may need to be restarted.
- Hybrid clusters (with some eBPF nodes and some standard dataplane nodes) are not supported.  (In such a cluster, NodePort traffic from eBPF nodes to non-eBPF nodes will be dropped.)  This includes clusters with Windows nodes.
- eBPF mode does not support floating IPs.
- eBPF mode does not support SCTP, either for policy or services.
- eBPF mode requires that node  [IP autodetection]({{site.baseurl}}/networking/ip-autodetection) is enabled even in environments where {{site.prodname}} CNI and BGP are not in use.  In eBPF mode, the node IP is used to originate VXLAN packets when forwarding traffic from external sources to services.
- eBPF mode does not support the "Log" action in policy rules.
