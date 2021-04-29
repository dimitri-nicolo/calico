---
title: Enable the eBPF dataplane
description: Step-by-step instructions for enabling the eBPF dataplane.
---

>**Note**: Support for eBPF mode is in tech preview in this release.  We recommend against deploying it in production
> because it has had less testing (particularly scale and robustness testing) than a full GA release.  This
> tech preview release has support for tiered policy; host endpoints (with normal, pre-DNAT and apply-on-forward
> policy); flow logs; and DNS policy.
{: .alert .alert-info}

### Big picture

This guide explains how to enable the eBPF dataplane; a high-performance alternative to the standard (iptables based) dataplane for both {{site.prodname}} and kube-proxy.

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
- eBPF mode does not support host ports (these are normally implemented by the "portmap" CNI plugin, which is incompatible with eBPF mode).
- eBPF mode does not support SCTP, either for policy or services.
- eBPF mode requires that node  [IP autodetection]({{site.baseurl}}/networking/ip-autodetection) is enabled even in environments where {{site.prodname}} CNI and BGP are not in use.  In eBPF mode, the node IP is used to originate VXLAN packets when forwarding traffic from external sources to services.
- eBPF mode does not support the "Log" action in policy rules. This limitation also applies to the Drop Action Override feature: `LOGandDROP` and `LOGandACCEPT` are interpreted as `DROP` and `ACCEPT`, respectively.

### Features

This how-to guide uses the following {{site.prodname}} features:

- **calico/node**
- **eBPF dataplane**

### Concepts

#### eBPF

eBPF (or "extended Berkeley Packet Filter"), is a technology that allows safe mini programs to be attached to various low-level hooks in the Linux kernel. eBPF has a wide variety of uses, including networking, security, and tracing. You’ll see a lot of non-networking projects leveraging eBPF, but for {{site.prodname}} our focus is on networking, and in particular, pushing the networking capabilities of the latest Linux kernels to the limit.

### Before you begin...

This document assumes that you have [installed {{site.prodname}}]({{site.baseurl}}/getting-started/kubernetes/) on a 
Kubernetes cluster that meets pre-requisites for eBPF mode:

- A supported Linux distribution:

  - Ubuntu 20.04.
  - Red Hat v8.2 with Linux kernel v4.18.0-193 or above (Red Hat have backported the required features to that build).
  - Another [supported distribution]({{site.baseurl}}/getting-started/kubernetes/requirements) with Linux kernel v5.3 or above.

  If {{site.prodname}} does not detect a compatible kernel, {{site.prodname}} will emit a warning and fall back to standard linux networking.

- On each node, the BPF filesystem must be mounted at `/sys/fs/bpf`.  This is required so that the BPF filesystem persists
  when {{site.prodname}} is restarted.  If the filesystem does not persist then pods will temporarily lose connectivity when
  {{site.prodname}} is restarted and host endpoints may be left unsecured (because their attached policy program will be
  discarded).
- For best pod-to-pod performance, an underlying network that doesn't require Calico to use an overlay.  For example:

  - A cluster within a single AWS subnet.
  - A cluster using a compatible cloud provider's CNI (such as the AWS VPC CNI plugin).
  - An on-prem cluster with BGP peering configured.

  If you must use an overlay, we recommend that you use VXLAN, not IPIP.  VXLAN has much better performance than IPIP in
  eBPF mode due to various kernel optimisations.

- The underlying network must be configured to allow VXLAN packets between {{site.prodname}} hosts (even if you normally
  use IPIP or non-overlay for Calico traffic).  In eBPF mode, VXLAN is used to forward Kubernetes NodePort traffic,
  while preserving source IP.  eBPF mode honours the Felix `VXLANMTU` setting (see [Configuring MTU]({{ site.baseurl }}/networking/mtu)).
- A stable way to address the Kubernetes API server. Since eBPF mode takes over from kube-proxy, {{site.prodname}}
  needs a way to reach the API server directly.
- The base [requirements]({{site.baseurl}}/getting-started/kubernetes/requirements) also apply.

> **Note**: The default kernel used by EKS is not compatible with eBPF mode.  If you wish to try eBPF mode with EKS,
> follow the [Creating an EKS cluster for eBPF mode](./ebpf-and-eks) guide, which explain how to set up a suitable cluster.
{: .alert .alert-info}

### How to

- [Verify that your cluster is ready for eBPF mode](#verify-that-your-cluster-is-ready-for-ebpf-mode)
- [Configure {{site.prodname}} to talk directly to the API server](#configure-{{site.prodnamedash}}-to-talk-directly-to-the-api-server)
- [Configure kube-proxy](#configure-kube-proxy)
- [Enable eBPF mode](#enable-ebpf-mode)
- [Try out DSR mode](#try-out-dsr-mode)
- [Reversing the process](#reversing-the-process)

#### Verify that your cluster is ready for eBPF mode

This section explains how to make sure your cluster is suitable for eBPF mode.

1. To check that the kernel on a node is suitable, you can run

   ```bash
   uname -rv
   ```

   The output should look like this:

   ```
   5.4.0-42-generic #46-Ubuntu SMP Fri Jul 10 00:24:02 UTC 2020
   ```

   In this case the kernel version is v5.4, which is suitable.

   On Red Hat-derived distributions, you may see something like this:
   ```
   4.18.0-193.el8.x86_64 (mockbuild@x86-vm-08.build.eng.bos.redhat.com)
   ```
   Since the Red Hat kernel is v4.18 with at least build number 193, this kernel is suitable.

1. To verify that the BPF filesystem is mounted, on the host, you can run the following command:

   ```
   mount | grep "/sys/fs/bpf"
   ```

   If the BPF filesystem is mounted, you should see:

   ```
   none on /sys/fs/bpf type bpf (rw,nosuid,nodev,noexec,relatime,mode=700)
   ```

   If you see no output, then the BPF filesystem is not mounted; consult the documentation for your OS distribution to see how to make sure the file system is mounted at boot in its standard location  /sys/fs/bpf.  This may involve editing `/etc/fstab` or adding a `systemd` unit, depending on your distribution. If the file system is not mounted on the host then eBPF mode will work normally until {{site.prodname}} is restarted, at which point workload networking will be disrupted for several seconds.

#### Configure {{site.prodname}} to talk directly to the API server

In eBPF mode, {{site.prodname}} implements Kubernetes service networking directly (rather than relying on `kube-proxy`).  This means that, like `kube-proxy`,  {{site.prodname}} must connect _directly_ to the Kubernetes API server rather than via the API server's ClusterIP.

First, make a note of the address of the API server:

   * If you have a single API server, you can use its IP address and port.  The IP can be found by running:

     ```
     kubectl get endpoints kubernetes -o wide
     ```

     The output should look like the following, with a single IP address and port under "ENDPOINTS":

     ```
     NAME         ENDPOINTS             AGE
     kubernetes   172.16.101.157:6443   40m
     ```

     If there are multiple entries under "ENDPOINTS" then your cluster must have more than one API server.  In that case, you should try to determine the load balancing approach used by your cluster and use the appropriate option below.

   * If using DNS load balancing (as used by `kops`), use the FQDN and port of the API server `api.internal.<clustername>`.
   * If you have multiple API servers with a load balancer in front, you should use the IP and port of the load balancer.

> **Tip**: If your cluster uses a ConfigMap to configure `kube-proxy` you can find the "right" way to reach the API
> server by examining the config map.  For example:
> ```
> $ kubectl get configmap -n kube-system kube-proxy -o yaml | grep server`
>     server: https://d881b853ae312e00302a84f1e346a77.gr7.us-west-2.eks.amazonaws.com
> ```
> In this case, the server is `d881b853aea312e00302a84f1e346a77.gr7.us-west-2.eks.amazonaws.com` and the port is
> 443 (the standard HTTPS port).
{: .alert .alert-success}

Then, create the following config map in the `tigera-operator` namespace using the host and port determined above:

```
kind: ConfigMap
apiVersion: v1
metadata:
  name: kubernetes-services-endpoint
  namespace: tigera-operator
data:
  KUBERNETES_SERVICE_HOST: "<API server host>"
  KUBERNETES_SERVICE_PORT: "<API server port>"
```
The operator will pick up the change to the config map automatically and do a rolling update of {{site.prodname}} to pass on the change.  Confirm that pods restart and then reach the `Running` state with the following command:

```
watch kubectl get pods -n calico-system
```

If you do not see the pods restart then it's possible that the `ConfigMap` wasn't picked up (sometimes Kubernetes is slow to propagate `ConfigMap`s (see Kubernetes [issue #30189](https://github.com/kubernetes/kubernetes/issues/30189){:target="_blank"})). You can try restarting the operator.

#### Configure kube-proxy

In eBPF mode {{site.prodname}} replaces `kube-proxy` so it wastes resources to run both.  This section explains how
to disable `kube-proxy` in some common environments.

##### Clusters that run `kube-proxy` with a `DaemonSet` (such as `kubeadm`)

For a cluster that runs `kube-proxy` in a `DaemonSet` (such as a `kubeadm`-created cluster), you can disable `kube-proxy` reversibly by adding a node selector to `kube-proxy`'s `DaemonSet` that matches no nodes, for example:

```
kubectl patch ds -n kube-system kube-proxy -p '{"spec":{"template":{"spec":{"nodeSelector":{"non-calico": "true"}}}}}'
```

Then, should you want to start `kube-proxy` again, you can simply remove the node selector.

If you choose not to disable `kube-proxy` (for example, because it is managed by your Kubernetes distribution), then you *must* change Felix configuration parameter `BPFKubeProxyIptablesCleanupEnabled` to `false`.  This can be done with `kubectl` as follows:

```
kubectl patch felixconfiguration.p default --patch='{"spec": {"bpfKubeProxyIptablesCleanupEnabled": false}}'
```

If both `kube-proxy` and `BPFKubeProxyIptablesCleanupEnabled` is enabled then `kube-proxy` will write its iptables rules and Felix will try to clean them up resulting in iptables flapping between the two.

##### OpenShift

If you are running OpenShift, you can disable `kube-proxy` as follows:

```
kubectl patch networks.operator.openshift.io cluster --type merge -p '{"spec":{"deployKubeProxy": false}}'
```

To re-enable it:

```
kubectl patch networks.operator.openshift.io cluster --type merge -p '{"spec":{"deployKubeProxy": true}}'
```

#### Enable eBPF mode

To enable eBPF mode, change the `spec.calicoNetwork.linuxDataplane` parameter in the operator's `Installation` 
resource to `"BPF"`; you must also clear the `hostPorts` setting because host ports are not supported in BPF mode:

```bash
kubectl patch installation.operator.tigera.io default --type merge -p '{"spec":{"calicoNetwork":{"linuxDataplane":"BPF", "hostPorts":null}}}'
```

> **Note**: the operator rolls out the change with a rolling update which means that some nodes will be in eBPF mode
> before others.  This can disrupt the flow of traffic through node ports.  We plan to improve this in an upcoming release
> by having the operator do the update in two phases.
{: .alert .alert-info}

#### Try out DSR mode

Direct return mode skips a hop through the network for traffic to services (such as node ports) from outside the cluster.  This reduces latency and CPU overhead but it requires the underlying network to allow nodes to send traffic with each other's IPs.  In AWS, this requires all your nodes to be in the same subnet and for the source/dest check to be disabled.

DSR mode is disabled by default; to enable it, set the `BPFExternalServiceMode` Felix configuration parameter to `"DSR"`.  This can be done with `kubectl`:

```
kubectl patch felixconfiguration.p default --patch='{"spec": {"bpfExternalServiceMode": "DSR"}}'
```

To switch back to tunneled mode, set the configuration parameter to `"Tunnel"`:

```
kubectl patch felixconfiguration.p default --patch='{"spec": {"bpfExternalServiceMode": "Tunnel"}}'
```

Switching external traffic mode can disrupt in-progress connections.

#### Reversing the process

To revert to standard Linux networking:

1. Reverse the changes to the operator's `Installation`:

   ```bash
   kubectl patch installation.operator.tigera.io default --type merge -p '{"spec":{"calicoNetwork":{"linuxDataplane":"Iptables"}}}'
   ```

1. If you disabled `kube-proxy`, re-enable it (for example, by removing the node selector added above).
   ```
   kubectl patch ds -n kube-system kube-proxy --type merge -p '{"spec":{"template":{"spec":{"nodeSelector":{"non-calico": null}}}}}'
   ```

1. Monitor existing workloads to make sure they re-establish any connections disrupted by the switch.
