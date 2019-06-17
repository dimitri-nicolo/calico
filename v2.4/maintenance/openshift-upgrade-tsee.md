---
title: Upgrading Tigera Secure EE from an earlier release on OpenShift
redirect_from: latest/maintenance/openshift-upgrade-tsee
canonical_url: https://docs.tigera.io/v2.4/maintenance/openshift-upgrade-tsee
---

## Prerequisite

Ensure that your Kubernetes cluster is already running version 2.3 of {{site.prodname}}. Upgrade from earlier versions to {{page.version}} is not supported.

If your cluster has a Calico installation, follow the [Upgrading an OpenShift cluster with Calico to {{site.prodname}} guide](/{{page.version}}/getting-started/openshift/upgrade-ee)
instead.

## Upgrading to {{page.version}} {{site.prodname}}

If you used the manifests provided on the [Tigera documentation site](https://docs.tigera.io/)
to install {{site.prodname}}, re-install using the instructions below. To avoid unneccessary service impact, ensure you modify the various
manifests to include any changes that were previously made for your current deployment *prior* to applying the new
manifests.

Note: If you depend on the cluster IP of the `cnx-manager` service, you will need to update references to its port `8080` to
port `9443`.
{: .alert .alert-info}

{% include {{page.version}}/install-upgrade-openshift.md upgrade=true %}
