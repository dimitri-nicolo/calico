---
title: Installing CNX on Kubernetes
canonical_url: https://docs.tigera.io/v2.3/getting-started/kubernetes/installation/
---

{{site.tseeprodname}} can be installed on a Kubernetes cluster in a number of configurations.  This document
gives an overview of the most popular approaches, and provides links to each for more detailed
information.

## Requirements

{{site.tseeprodname}} can run on any Kubernetes cluster which meets the following criteria.

- The kubelet must be configured to use CNI network plugins (e.g `--network-plugin=cni`).
- The kube-proxy must be started in `iptables` proxy mode.  This is the default as of Kubernetes v1.2.0.
- The kube-proxy must be started without the `--masquerade-all` flag, which conflicts with {{site.tseeprodname}} policy.
- The Kubernetes `NetworkPolicy` API requires at least Kubernetes version v1.3.0.
- When RBAC is enabled, the proper accounts, roles, and bindings must be defined
  and utilized by the {{site.tseeprodname}} components.  Examples exist for both the [etcd](rbac.yaml) and
  [kubernetes api](hosted/rbac-kdd.yaml) datastores.
- The cluster must be able to load the Tigera CNX docker images - see following section.
- Public clouds require special configuration. Refer to [AWS](../../../reference/public-cloud/aws), [Azure](../../../reference/public-cloud/azure), and [GCE](../../../reference/public-cloud/gce) configuration guides for details.

## [{{site.tseeprodname}} Hosted Install](hosted)

Installs the {{site.tseeprodname}} components as a DaemonSet entirely using Kubernetes manifests through a single
kubectl command.

## [Custom Installation](integration)

In addition to the hosted approach above, the {{site.tseeprodname}} components can also be installed using your
own orchestration mechanisms (e.g ansible, chef, bash, etc)

Follow the [integration guide](integration) if you're using a Kubernetes version < v1.4.0, or if you would like
to integrate {{site.tseeprodname}} into your own installation or deployment scripts.
