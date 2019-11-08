---
title: Upgrading from Calico to Calico Enterprise
redirect_from: latest/getting-started/kubernetes/upgrade/upgrade-to-tsee
canonical_url: https://docs.tigera.io/v2.3/getting-started/kubernetes/upgrade/upgrade-to-tsee
---

{% assign calico_minor_version = site.data.versions[page.version].first["calico"].minor_version %}

## Prerequisite
{% assign old_vers = "v3.1,v3.2,v3.4,v3.5" | split: "," %}

Ensure that your Kubernetes cluster is running with open source Calico on the latest {{ calico_minor_version | append: '.x' }}
release. If not, follow the {% unless old_vers contains calico_minor_version %}
[Calico upgrade documentation](https://docs.projectcalico.org/{{ calico_minor_version }}/maintenance/kubernetes-upgrade) before continuing.
{% else %}
[Calico upgrade documentation](https://docs.projectcalico.org/{{ calico_minor_version }}/getting-started/kubernetes/upgrade/upgrade) before continuing.
{% endunless %}

If your cluster already has {{site.prodname}} installed, follow the [Upgrading {{site.prodname}} from an earlier release guide](/{{page.version}}/maintenance/kubernetes-upgrade-tsee)
instead.

## Upgrading Calico to {{site.prodname}}

If you used the manifests provided on the [Calico documentation site](https://docs.projectcalico.org/)
to install Calico, complete the {{site.prodname}} installation procedure that
corresponds to your Calico installation method.

- [Installing {{site.prodname}} for policy and networking](/{{page.version}}/reference/other-install-methods/kubernetes/installation/calico)

- [Installing {{site.prodname}} for policy](/{{page.version}}/reference/other-install-methods/kubernetes/installation/other)

If you modified the manifests or used the
[Integration Guide](https://docs.projectcalico.org/latest/getting-started/kubernetes/installation/integration)
to install Calico, contact Tigera support for assistance with your upgrade
to {{site.prodname}}.
