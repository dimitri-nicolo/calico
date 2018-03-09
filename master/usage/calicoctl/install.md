---
title: Installing calicoctl
canonical_url: 'https://docs.projectcalico.org/v3.0/usage/calicoctl/install'
---

## About installing calicoctl

`calicoctl` allows you to create, read, update, and delete {{site.prodname}} objects
from the command line. 

You can run `calicoq` on any host with network access to the
{{site.prodname}} datastore as either a binary or a container.
For step-by-step instructions, refer to the section that
corresponds to your desired deployment.

- [As a binary on a single host](#installing-calicoctl-as-a-binary-on-a-single-host)

- [As a container on a single host](#installing-calicoctl-as-a-container-on-a-single-host)

- [As a container on each node](#installing-calicoctl-as-a-container-on-each-node)


## Installing calicoctl as a binary on a single host

1. Log into the host and open a terminal prompt.

{% include {{page.version}}/ctl-binary-install.md cli="calicoctl" %}

**Next step**:

[Configure `calicoctl` to connect to your datastore](/{{page.version}}/usage/calicoctl/configure/).

{% include {{page.version}}/ctl-container-install.md cli="calicoctl" %}

**Next step**:

[Configure `calicoctl` to connect to your datastore](/{{page.version}}/usage/calicoctl/configure/).
