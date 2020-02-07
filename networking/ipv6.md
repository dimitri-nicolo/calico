---
title: Enabling IPv6 support
description: Enable IPv6 support for workloads on Kubernetes and OpenStack.
canonical_url: '/networking/ipv6'
---

### About enabling IPv6

After enabling IPv6:
- Workloads can communicate over IPv6.
- Workloads can initiate connections to IPv6 services.
- Workloads can terminate incoming IPv6 connections.

Support for IPv6 and the procedure for enabling it varies by orchestrator.
Refer to the section that corresponds to your orchestrator for details.

- [Enabling IPv6 with Kubernetes](#enabling-ipv6-with-kubernetes)

## Enabling IPv6 with Kubernetes

### Limitations

- Kubernetes 1.15 and earlier only support one IP stack version at a time. This
  means that if you configure Kubernetes for IPv6 then {{site.prodname}}
  should be configured to assign only IPv6 addresses. Starting with 1.16, it is
  also possible to configure a dual-stack environment.
- The steps and setup here have not been tested against an existing IPv4
  cluster and are intended only for new clusters.

### Prerequisites

#### Host prerequisites

- Each Kubernetes host must have an IPv6 address that is reachable from
  the other hosts.
- Each host must have the sysctl setting `net.ipv6.conf.all.forwarding`
  setting it to `1`.  This ensures both Kubernetes service traffic
  and {{site.prodname}} traffic is forwarded appropriately.
- Each host must have a default IPv6 route.

#### Kubernetes components prerequisites

Kubernetes components must be configured to operate with IPv6.
To enable IPv6, set the following flags.

##### kube-apiserver

| Flag | Value/Content |
| ---- | ------------- |
| `--bind-address` or `--insecure-bind-address` | Should be set to the appropriate IPv6 address or `::` for all IPv6 addresses on the host. |
| `--advertise-address` | Should be set to the IPv6 address that nodes should use to access the kube-apiserver. |
| `--service-cluster-ip-range` | Should be set to an IPv6 CIDR that will be used for the Service IPs, the DNS service address must be in this range. |

##### kube-controller-manager

| Flag | Value/Content |
| ---- | ------------- |
| `--master` | Should be set with the IPv6 address where the kube-apiserver can be accessed. |
| `--cluster-cidr` | Should be set to match the {{site.prodname}} IPv6 IPPool. |

##### kube-scheduler

| Flag | Value/Content |
| ---- | ------------- |
| `--master` | Should be set with the IPv6 address where the kube-apiserver can be accessed. |

##### kubelet

| Flag | Value/Content |
| ---- | ------------- |
| `--address` | Should be set to the appropriate IPv6 address or `::` for all IPv6 addresses. |
| `--cluster-dns` | Should be set to the IPv6 address that will be used for the service DNS, this must be in the range used for `--service-cluster-ip-range`. |
| `--node-ip` | Should be set to the IPv6 address of the node. |

##### kube-proxy

| Flag | Value/Content |
| ---- | ------------- |
| `--bind-address` | Should be set to the appropriate IPv6 address or `::` for all IPv6 addresses on the host. |
| `--master` | Should be set with the IPv6 address where the kube-apiserver can be accessed. |
| `--cluster-cidr` | Should be set to match the {{site.prodname}} IPv6 IPPool. |

### Enabling IPv6 support in {{site.prodname}}

To enable IPv6 support when installing {{site.prodname}} follow the
steps below.

1.  Follow our [installation docs]({{ site.baseurl }}/getting-started/kubernetes) to install using
    the Tigera operator on your cluster.

1.  When about to apply `custom-resources.yaml`, edit it first to define an IPv6 pod CIDR
    pool in the `Installation` resource.  For example, like this:

    ```yaml
    apiVersion: operator.tigera.io/v1
    kind: Installation
    metadata:
      name: default
    spec:
      # Install Calico Enterprise
      variant: TigeraSecureEnterprise
      ...
      calicoNetwork:
        ipPools:
        - cidr: fd5f:1801::/112
      ...
    ```

### Modifying your DNS for IPv6

It will probably be necessary to modify your DNS pod for IPv6. If you are using
[kube-dns]({{site.baseurl}}/getting-started/kubernetes/installation/manifests/kubedns.yaml),
then the following changes will ensure IPv6 operation.

- Update the image versions to at least `1.14.8`.
- Ensure the clusterIP for the DNS service matches the one specified to
  the kubelet as `--cluster-dns`.
- Add `--dns-bind-address=[::]` to the arguments for the kubedns container.
- Add `--no-negcache` to the arguments for the dnsmasq container.
- Switch the arguments on the sidecar container from
  ```
  --probe=kubedns,127.0.0.1:10053,kubernetes.default.svc.cluster.local,5,A
  --probe=dnsmasq,127.0.0.1:53,kubernetes.default.svc.cluster.local,5,A
  ```
  {: .no-select-button}
  to
  ```
  --probe=kubedns,127.0.0.1:10053,kubernetes.default.svc.cluster.local,5,SRV
  --probe=dnsmasq,127.0.0.1:53,kubernetes.default.svc.cluster.local,5,SRV
  ```
  {: .no-select-button}
