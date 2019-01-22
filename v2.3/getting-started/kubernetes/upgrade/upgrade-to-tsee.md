---
title: Upgrading from Calico to Tigera Secure EE
canonical_url: https://docs.tigera.io/v2.2/getting-started/kubernetes/upgrade-cnx
---

## Prerequisite

Ensure that your Kubernetes cluster is running with open source Calico on the latest v3.x
release. If not, follow the [Calico upgrade documentation](https://docs.projectcalico.org/latest/getting-started/kubernetes/upgrade/) 
before continuing.

If your cluster already has {{site.prodname}} installed, follow the [Upgrading {{site.prodname}} from an earlier release guide](./upgrade-tsee) 
instead.

## Upgrading Calico to {{site.prodname}}

If you used the manifests provided on the [Calico documentation site](https://docs.projectcalico.org/) 
to install Calico, complete the {{site.prodname}} installation procedure that 
corresponds to your Calico installation method.

- [Installing {{site.prodname}} for policy and networking](../installation/calico)

- [Installing {{site.prodname}} for policy](../installation/other)

If you modified the manifests or used the 
[Integration Guide](https://docs.projectcalico.org/latest/getting-started/kubernetes/installation/integration) 
to install Calico, contact Tigera support for assistance with your upgrade 
to {{site.prodname}}.
