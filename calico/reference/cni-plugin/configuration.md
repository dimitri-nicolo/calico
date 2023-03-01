---
title: Configuring the Calico Enterprise CNI plugins
description: Details for configuring the Calico Enterprise CNI plugins.
canonical_url: '/reference/cni-plugin/configuration'
---

>**Note**: The {{site.prodname}} CNI plugins do not need to be configured directly when installed by the operator.
>For a complete operator configuration reference, see [the installation API reference documentation][installation].
{: .alert .alert-info}

{% tabs %}
  <label:Operator,active:true>
<%

The `host-local` IPAM plugin can be configured by setting the `Spec.CNI.IPAM.Plugin` field to `HostLocal` on the [operator.tigera.io/Installation]({{site.baseurl}}/reference/installation/api#operator.tigera.io/v1.Installation) API.

Calico will use the `host-local` IPAM plugin to allocate IPv4 addresses from the node's IPv4 pod CIDR if there is an IPv4 pool configured in `Spec.IPPools`, and an IPv6 address from the node's IPv6 pod CIDR if
there is an IPv6 pool configured in `Spec.IPPools`.

The following example configures Calico to assign dual-stack IPs to pods using the host-local IPAM plugin.

```yaml
kind: Installation
apiVersion: operator.tigera.io/v1
metadata:
  name: default
spec:
  calicoNetwork:
    ipPools:
    - cidr: 192.168.0.0/16
    - cidr: 2001:db8::/64
  cni:
    type: Calico
    ipam:
      type: HostLocal
```

%>
  <label:Manifest>
<%

When using the CNI `host-local` IPAM plugin, two special values - `usePodCidr` and `usePodCidrIPv6` - are allowed for the subnet field (either at the top-level, or in a "range").  This tells the plugin to determine the subnet to use from the Kubernetes API based on the Node.podCIDR field. {{site.prodname}} does not use the `gateway` field of a range so that field is not required and it will be ignored if present.

> **Note**: `usePodCidr` and `usePodCidrIPv6` can only be used as the value of the `subnet` field, it cannot be used in
> `rangeStart` or `rangeEnd` so those values are not useful if `subnet` is set to `usePodCidr`.
{: .alert .alert-info}

{{site.prodname}} supports the host-local IPAM plugin's `routes` field as follows:

* If there is no `routes` field, {{site.prodname}} will install a default `0.0.0.0/0`, and/or `::/0` route into the pod (depending on whether the pod has an IPv4 and/or IPv6 address).

* If there is a `routes` field then {{site.prodname}} will program *only* the routes in the routes field into the pod.  Since {{site.prodname}} implements a point-to-point link into the pod, the `gw` field is not required and it will be ignored if present.  All routes that {{site.prodname}} installs will have {{site.prodname}}'s link-local IP as the next hop.

{{site.prodname}} CNI plugin configuration:

* `node_name`
    * The node name to use when looking up the CIDR value (defaults to current hostname)

```json
{
    "name": "any_name",
    "cniVersion": "0.1.0",
    "type": "calico",
    "kubernetes": {
        "kubeconfig": "/path/to/kubeconfig",
        "node_name": "node-name-in-k8s"
    },
    "ipam": {
        "type": "host-local",
        "ranges": [
            [
                { "subnet": "usePodCidr" }
            ],
            [
                { "subnet": "usePodCidrIPv6" }
            ]
        ],
        "routes": [
            { "dst": "0.0.0.0/0" },
            { "dst": "2001:db8::/96" }
        ]
    }
}
```

When making use of the `usePodCidr` or `usePodCidrIPv6` options, the {{site.prodname}} CNI plugin requires read-only Kubernetes API access to the `Nodes` resource.

#### Configuring node and typha

When using `host-local` IPAM with the Kubernetes API datastore, you must configure both {{site.nodecontainer}} and the Typha deployemt to use the `Node.podCIDR` field by setting the environment variable `USE_POD_CIDR=true` in each.

%>
{% endtabs %}

### Using Kubernetes annotations

#### Specifying IP pools on a per-namespace or per-pod basis

In addition to specifying IP pools in the CNI config as discussed above, {{site.prodname}} IPAM supports specifying IP pools per-namespace or per-pod using the following [Kubernetes annotations](https://kubernetes.io/docs/user-guide/annotations/){:target="_blank"}.

- `cni.projectcalico.org/ipv4pools`: A list of configured IPv4 Pools from which to choose an address for the pod.

   Example:

   ```yaml
   annotations:
      "cni.projectcalico.org/ipv4pools": "[\"default-ipv4-ippool\"]"
   ```

- `cni.projectcalico.org/ipv6pools`: A list of configured IPv6 Pools from which to choose an address for the pod.

   Example:

   ```yaml
   annotations:
      "cni.projectcalico.org/ipv6pools": "[\"2001:db8::1/120\"]"
   ```

If provided, these IP pools will override any IP pools specified in the CNI config.

> **Note**: This requires the IP pools to exist before `ipv4pools` or
> `ipv6pools` annotations are used. Requesting a subset of an IP pool
> is not supported. IP pools requested in the annotations must exactly
> match a configured [IPPool]({{site.baseurl}}/reference/resources/ippool) resource.
{: .alert .alert-info}

> **Note**: The {{site.prodname}} CNI plugin supports specifying an annotation per namespace.
> If both the namespace and the pod have this annotation, the pod information will be used.
> Otherwise, if only the namespace has the annotation the annotation of the namespace will
> be used for each pod in it.
{: .alert .alert-info}

#### Requesting a specific IP address

You can also request a specific IP address through [Kubernetes annotations](https://kubernetes.io/docs/user-guide/annotations/){:target="_blank"} with {{site.prodname}} IPAM.
There are two annotations to request a specific IP address:

- `cni.projectcalico.org/ipAddrs`: A list of IPv4 and/or IPv6 addresses to assign to the Pod. The requested IP addresses will be assigned from {{site.prodname}} IPAM and must exist within a configured IP pool.

  Example:

   ```yaml
   annotations:
        "cni.projectcalico.org/ipAddrs": "[\"192.168.0.1\"]"
   ```

- `cni.projectcalico.org/ipAddrsNoIpam`: A list of IPv4 and/or IPv6 addresses to assign to the Pod, bypassing IPAM. Any IP conflicts and routing have to be taken care of manually or by some other system.
{{site.prodname}} will only distribute routes to a Pod if its IP address falls within a {{site.prodname}} IP pool using BGP mode. Calico will not distribute ipAddrsNoIpam routes when operating in VXLAN mode. If you assign an IP address that is not in a {{site.prodname}} IP pool or if its IP address falls within a {{site.prodname}} IP pool that uses VXLAN encapsulation, you must ensure that routing to that IP address is taken care of through another mechanism.

  Example:

   ```yaml
   annotations:
        "cni.projectcalico.org/ipAddrsNoIpam": "[\"10.0.0.1\"]"
   ```

   The ipAddrsNoIpam feature is disabled by default. It can be enabled in the feature_control section of the CNI network config:

   ```json
   {
        "name": "any_name",
        "cniVersion": "0.1.0",
        "type": "calico",
        "ipam": {
            "type": "calico-ipam"
        },
       "feature_control": {
           "ip_addrs_no_ipam": true
       }
   }
   ```

   > **Warning**: This feature allows for the bypassing of network policy via IP spoofing.
   > Users should make sure the proper admission control is in place to prevent users from selecting arbitrary IP addresses.
   {: .alert .alert-danger}

> **Note**:
> - The `ipAddrs` and `ipAddrsNoIpam` annotations can't be used together.
> - You can only specify one IPv4/IPv6 or one IPv4 and one IPv6 address with these annotations.
> - When `ipAddrs` or `ipAddrsNoIpam` is used with `ipv4pools` or `ipv6pools`, `ipAddrs` / `ipAddrsNoIpam` take priority.
{: .alert .alert-info}

#### Requesting a floating IP

You can request a floating IP address for a pod through [Kubernetes annotations](https://kubernetes.io/docs/user-guide/annotations/){:target="_blank"} with {{site.prodname}}.

> **Note**:
> The specified address must belong to an IP Pool for advertisement to work properly.
{: .alert .alert-info}

- `cni.projectcalico.org/floatingIPs`: A list of floating IPs which will be assigned to the pod's workload endpoint.

  Example:

   ```yaml
   annotations:
        "cni.projectcalico.org/floatingIPs": "[\"10.0.0.1\"]"
   ```

   The floatingIPs feature is disabled by default. It can be enabled in the feature_control section of the CNI network config:

   ```json
   {
        "name": "any_name",
        "cniVersion": "0.1.0",
        "type": "calico",
        "ipam": {
            "type": "calico-ipam"
        },
       "feature_control": {
           "floating_ips": true
       }
   }
   ```

   > **Warning**: This feature can allow pods to receive traffic which may not have been intended for that pod.
   > Users should make sure the proper admission control is in place to prevent users from selecting arbitrary floating IP addresses.
   {: .alert .alert-danger}

### Using IP pools node selectors

Nodes will only assign workload addresses from IP pools which select them. By
default, IP pools select all nodes, but this can be configured using the
`nodeSelector` field. Check out the [IP pool resource document]({{site.baseurl}}/reference/resources/ippool)
for more details.

Example:

1. Create (or update) an IP pool that only allocates IPs for nodes where it
   contains a label `rack=0`.

   ```bash
   kubectl create -f -<<EOF
   apiVersion: projectcalico.org/v3
   kind: IPPool
   metadata:
      name: rack-0-ippool
   spec:
      cidr: 192.168.0.0/24
      ipipMode: Always
      natOutgoing: true
      nodeSelector: rack == "0"
   EOF
   ```

2. Label a node with `rack=0`.

   ```bash
   kubectl label nodes kube-node-0 rack=0
   ```

Check out the usage guide on [assign IP addresses based on
topology]({{site.baseurl}}/networking/assign-ip-addresses-topology)
for a full example.

### CNI network configuration lists

The CNI 0.3.0 [spec](https://github.com/containernetworking/cni/blob/spec-v0.3.0/SPEC.md#network-configuration-lists){:target="_blank"} supports "chaining" multiple CNI plugins together. {{site.prodname}} supports the following Kubernetes CNI plugins, which are enabled by default. Although chaining other CNI plugins may work, we support only the following tested CNI plugins. 

**Port mapping plugin**

{{site.prodname}} is required to implement Kubernetes host port functionality and is enabled by default. 

> **Note**: Be aware of the following {% include open-new-window.html text='portmap plugin CNI issue' url='https://github.com/containernetworking/cni/issues/605' %} where draining nodes
> may take a long time with a cluster of 100+ nodes and 4000+ services.
{: .alert .alert-info}

To disable it, remove the portmap section from the CNI network configuration in the {{site.prodname}} manifests. 

```json
        {
          "type": "portmap",
          "snat": true,
          "capabilities": {"portMappings": true}
        }
```
{: .no-select-button}

**Traffic shaping plugin**

The {% include open-new-window.html text='traffic shaping Kubernetes CNI plugin' url='https://kubernetes.io/docs/concepts/extend-kubernetes/compute-storage-net/network-plugins/' %} supports pod ingress and egress traffic shaping. This bandwidth management technique delays the flow of certain types of network packets to ensure network performance for higher priority applications. It is enabled by default. 

You can add the `kubernetes.io/ingress-bandwidth` and `kubernetes.io/egress-bandwidth` annotations to your pod. For example, the following sets a 1 megabit-per-second connection for ingress and egress traffic.

```bash
apiVersion: v1
kind: Pod
metadata:
  annotations:
    kubernetes.io/ingress-bandwidth: 1M
    kubernetes.io/egress-bandwidth: 1M
...
```
To disable it, remove the bandwidth section from the the CNI network configuration in the {{site.prodname}} manifests.

```json
        { 
          "type": "bandwidth",
          "capabilities": {"bandwidth": true}
        }
```   
{: .no-select-button}     

### Order of precedence

If more than one of these methods are used for IP address assignment, they will
take on the following precedence, 1 being the highest:

1. Kubernetes annotations
2. CNI configuration
3. IP pool node selectors

> **Note**: {{site.prodname}} IPAM will not reassign IP addresses to workloads
> that are already running. To update running workloads with IP addresses from
> a newly configured IP pool, they must be recreated. We recommend doing this
> before going into production or during a maintenance window.
{: .alert .alert-info}

### Specify num_queues for veth interfaces

`num_rx_queues` and `num_tx_queues` can be set using the `num_queues` option in the CNI configuration. Default: 1

For example:

```json
{
  "num_queues": 3
}
```



[installation]: {{site.baseurl}}/reference/installation/api
