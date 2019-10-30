---
title: Upgrading Tigera Secure EE from an earlier release on Kubernetes
canonical_url: https://docs.tigera.io/v2.4/maintenance/kubernetes-upgrade-tsee
---

## Prerequisite

Ensure that your Kubernetes cluster is already running version 2.3 of {{site.prodname}}. If your cluster is on a version
earlier than 2.3, [first upgrade to 2.3](/v2.3/getting-started/kubernetes/upgrade/upgrade-tsee), then follow these instructions
to upgrade to {{page.version}}.

If your cluster has a Calico installation, follow the [Upgrading a Kubernetes cluster with Calico to {{site.prodname}} guide]({{site.url}}/{{page.version}}/getting-started/kubernetes/upgrade/upgrade-to-tsee)
instead.

## Upgrading to {{page.version}} {{site.prodname}}

If you used the manifests provided on the [Tigera documentation site](https://docs.tigera.io/)
to install {{site.prodname}}, re-install using the instructions below. To avoid unneccessary service impact, ensure you modify the various
manifests to include any changes that were previously made for your current deployment *prior* to applying the new
manifests.

Note: If you depend on the cluster IP of the `cnx-manager` service, you will need to update references to its port `8080` to
port `9443`.
{: .alert .alert-info}

{% include {{page.version}}/install-upgrade-main.md upgrade=true %}
