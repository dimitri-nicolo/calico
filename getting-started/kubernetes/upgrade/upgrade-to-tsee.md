---
title: Upgrading from Calico to Calico Enterprise
description: Steps to upgrade from open source Calico to Calico Enterprise.
canonical_url: /getting-started/kubernetes/upgrade/upgrade-to-tsee
---

{% assign calico_minor_version = site.data.versions.first["calico"].minor_version %}

## Prerequisite
Ensure that your Kubernetes cluster is running with open source Calico on the latest {{ calico_minor_version | append: '.x' }}
release. If not, follow the [Calico upgrade documentation](https://docs.projectcalico.org/{{calico_minor_version}}/maintenance/kubernetes-upgrade) before continuing.


If your cluster already has {{site.prodname}} installed, follow the [Upgrading {{site.prodname}} from an earlier release guide]({{site.baseurl}}/maintenance/kubernetes-upgrade-tsee)
instead.

## Upgrading Calico to {{site.prodname}}

If you used the manifests provided on the [Calico documentation site](https://docs.projectcalico.org/)
to install Calico, complete the {{site.prodname}} installation procedure that
corresponds to your Calico installation method.

- [Installing {{site.prodname}} for policy and networking]({{site.baseurl}}/reference/other-install-methods/kubernetes/installation/calico)

- [Installing {{site.prodname}} for policy]({{site.baseurl}}/reference/other-install-methods/kubernetes/installation/other)

If you modified the manifests or used the
[Integration Guide](https://docs.projectcalico.org/latest/getting-started/kubernetes/installation/integration)
to install Calico, contact Tigera support for assistance with your upgrade
to {{site.prodname}}.
