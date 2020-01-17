---
title: Installing calicoctl
description: Install the CLI for Calico.
canonical_url: '/getting-started/calicoctl/install'
---

## About installing calicoctl

`calicoctl` allows you to create, read, update, and delete {{site.prodname}} objects
from the command line. These objects represent the networking and policy
of your cluster.

You should limit access to `calicoctl` and your {{site.prodname}} datastore to
trusted administrators. We discuss methods of limiting access to the
{{site.prodname}} datastore in the [configuration section](configure).

You can run `calicoctl` on any host with network access to the
{{site.prodname}} datastore as either a binary or a container.
For step-by-step instructions, refer to the section that
corresponds to your desired deployment.

- [As a binary on a single host](#installing-calicoctl-as-a-binary-on-a-single-host)

- [As a container on a single host](#installing-calicoctl-as-a-container-on-a-single-host)

## Installing calicoctl as a binary on a single host

{% include content/ctl-binary-install.md cli="calicoctl" codepath="/calicoctl" %}

**Next step**:

[Configure `calicoctl` to connect to your datastore](configure).

{% include content/ctl-container-install.md cli="calicoctl" %}
