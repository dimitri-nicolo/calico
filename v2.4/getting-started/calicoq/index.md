---
title: Installing calicoq
canonical_url: https://docs.tigera.io/v2.3/usage/calicoq/
---

## About installing calicoq

You can run `calicoq` on any host with network access to the
{{site.tseeprodname}} datastore as either a binary or a container.
For step-by-step instructions, refer to the section that
corresponds to your desired deployment.

- [As a binary on a single host](#installing-calicoq-as-a-binary-on-a-single-host)

- [As a container on a single host](#installing-calicoq-as-a-container-on-a-single-host)

- [As a Kubernetes pod](#installing-calicoq-as-a-kubernetes-pod)


## Installing calicoq as a binary on a single host

{% include {{page.version}}/ctl-binary-install.md cli="calicoq" codepath="/calicoq" %}

**Next step**:

[Configure `calicoq` to connect to your datastore](/{{page.version}}/getting-started/calicoq/configure/).

{% include {{page.version}}/ctl-container-install.md cli="calicoq" %}
