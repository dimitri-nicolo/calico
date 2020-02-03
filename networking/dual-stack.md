---
title: Enable dual stack
description: Configure Kubernetes pods to get both IPv4 and IPv6 addresses.
canonical_url: '/networking/dual-stack'
---

### Big picture

By default, pods are assigned only IPv4 addresses.  Dual stack means that each Kubernetes pod gets
both an IPv4 and an IPv6 address, so that it can communicate over both IPv4 and IPv6.

### Value

Communication over IPv6 is increasingly desirable, and the natural approach for cluster pods is to
be IPv6-native themselves, while still supporting IPv4.  Native support for both IPv4 and IPv6
is known as "dual stack", and Kubernetes has alpha-level support for this in versions 1.16 and 1.17.

### Features

This how-to guide uses the following {{site.prodname}} features:

- [**Installation API**]({{ site.baseurl }}/reference/installation/api)

### Before you begin...

Set up a cluster following the Kubernetes
[prerequisites](https://kubernetes.io/docs/concepts/services-networking/dual-stack/#prerequisites)
and [enablement
steps](https://kubernetes.io/docs/concepts/services-networking/dual-stack/#enable-ipv4-ipv6-dual-stack).

### How to

1.  Follow our [installation docs]({{ site.baseurl }}/getting-started/kubernetes) to install using
    the Tigera operator on your cluster.

1.  When about to apply `custom-resources.yaml`, edit it first to define both IPv4 and IPv6 pod CIDR
    pools in the `Installation` resource.  For example, like this:

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
        - cidr: 10.244.0.0/16
        - cidr: fd5f:1801::/112
      ...
    ```

You should then observe that new pods get IPv6 addresses as well as IPv4, and can communicate with
each other and the outside world over IPv6.
