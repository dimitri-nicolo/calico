---
title: Upgrading a Kubernetes cluster with Calico to Tigera Secure EE
canonical_url: https://docs.tigera.io/v2.3/getting-started/kubernetes/upgrade/upgrade-to-tsee
---

## Prerequisite

Ensure that the open source Calico cluster is on the latest v3.1.x
release.

If not, follow the [Calico upgrade documentation](https://docs.projectcalico.org/v3.1/getting-started/kubernetes/upgrade/)
before continuing.

## Upgrading Calico to {{site.tseeprodname}}

If you used the manifests provided on the [Calico documentation site](https://docs.projectcalico.org/)
to install Calico, complete the {{site.tseeprodname}} installation procedure that
corresponds to your Calico installation method.

- [Installing {{site.tseeprodname}} for policy and networking](installation/calico)

- [Installing {{site.tseeprodname}} for policy](installation/other)

If you modified the manifests or used the
[Integration Guide](https://docs.projectcalico.org/latest/getting-started/kubernetes/installation/integration)
to install Calico, contact Tigera support for assistance with your upgrade
to {{site.tseeprodname}}.

