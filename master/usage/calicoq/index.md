---
title: Installing calicoq
---

## About installing calicoq

You can run `calicoq` on any host with network access to the
{{site.prodname}} datastore as either a binary or a container.
For step-by-step instructions, refer to the section that
corresponds to your desired deployment.

- [As a binary on a single host](#installing-calicoq-as-a-binary-on-a-single-host)

- [As a container on a single host](#installing-calicoq-as-a-container-on-a-single-host)

- [As a container on each node](#installing-calicoq-as-a-container-on-each-node)


## Installing calicoq as a binary on a single host

1. Log into the host and open a terminal prompt.

{% include {{page.version}}/ctl-binary-install.md cli="calicoq" %}

**Next step**:

[Configure `calicoq` to connect to your datastore](/{{page.version}}/usage/calicoq/configure/).


{% include {{page.version}}/ctl-container-install.md cli="calicoq" %}

**Next step**:

[Configure `calicoq` to connect to your datastore](/{{page.version}}/usage/calicoq/configure/).


