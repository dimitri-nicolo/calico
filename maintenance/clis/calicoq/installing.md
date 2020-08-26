---
title: Install calicoq
description: Install the CLI for Calico Enterprise.
canonical_url: /maintenance/clis/calicoq/
---

## About installing calicoq

You can run `calicoq` on any host with network access to the
{{site.prodname}} datastore as either a binary or a container.
For step-by-step instructions, refer to the section that
corresponds to your desired deployment.

- [As a binary on a single host](#install-calicoq-as-a-binary-on-a-single-host)

- [As a container on a single host](#install-calicoq-as-a-container-on-a-single-host)

{% include content/ctl-binary-install.md cli="calicoq" codepath="/calicoq" %}

{% include content/ctl-container-install.md cli="calicoq" codepath="/calicoq" %}
