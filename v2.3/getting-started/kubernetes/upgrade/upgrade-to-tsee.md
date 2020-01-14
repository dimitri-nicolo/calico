---
title: Upgrading from Calico to Tigera Secure EE
canonical_url: https://docs.tigera.io/v2.3/getting-started/kubernetes/upgrade/upgrade-to-tsee
---

## Prerequisite
{% assign old_vers = "v3.1,v3.2,v3.4,v3.5" | split: "," %}

Ensure that your Kubernetes cluster is running with open source Calico on the latest {{site.data.versions[page.version].first.components["calico"].minor_version | append: '.x' }}
release. If not, follow the {% unless old_vers contains site.data.versions[page.version].first.components["calico"].minor_version %}
[Calico upgrade documentation](https://docs.projectcalico.org/{{site.data.versions[page.version].first.components["calico"].minor_version}}/maintenance/kubernetes-upgrade) before continuing.
{% else %}
[Calico upgrade documentation](https://docs.projectcalico.org/{{site.data.versions[page.version].first.components["calico"].minor_version}}/getting-started/kubernetes/upgrade/upgrade) before continuing.
{% endunless %}


If your Kubernetes cluster is running with a higher version of open source Calico {{site.data.versions[page.version].first.components["calico"].minor_version | append: '.x' }} please contact Tigera support
for assistance with your upgrade.

If your cluster already has {{site.tseeprodname}} installed, follow the [Upgrading {{site.tseeprodname}} from an earlier release guide](./upgrade-tsee) 
instead.

## Upgrading Calico to {{site.tseeprodname}}

If you used the manifests provided on the [Calico documentation site](https://docs.projectcalico.org/) 
to install Calico, complete the {{site.tseeprodname}} installation procedure that 
corresponds to your Calico installation method.

- [Installing {{site.tseeprodname}} for policy and networking](../installation/calico)

- [Installing {{site.tseeprodname}} for policy](../installation/other)

If you modified the manifests or used the 
[Integration Guide](https://docs.projectcalico.org/latest/getting-started/kubernetes/installation/integration) 
to install Calico, contact Tigera support for assistance with your upgrade 
to {{site.tseeprodname}}.
