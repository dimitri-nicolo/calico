---
title: Installing calicoctl
canonical_url: 'https://docs.tigera.io/v2.3/usage/calicoctl/install'
---

## About installing calicoctl

`calicoctl` allows you to create, read, update, and delete {{site.tseeprodname}} objects
from the command line. These objects represent the networking and policy
of your cluster.

You should limit access to `calicoctl` and your {{site.tseeprodname}} datastore to
trusted administrators. We discuss methods of limiting access to the
{{site.tseeprodname}} datastore in the [configuration section](configure).

You can run `calicoctl` on any host with network access to the
{{site.tseeprodname}} datastore as either a binary or a container.
For step-by-step instructions, refer to the section that
corresponds to your desired deployment.

- [As a binary on a single host](#installing-calicoctl-as-a-binary-on-a-single-host)

- [As a container on a single host](#installing-calicoctl-as-a-container-on-a-single-host)

- [As a Kubernetes pod](#installing-calicoctl-as-a-kubernetes-pod)

> **Note**: We highly recommend you install calicoctl as a Kubernetes pod in OpenShift.
This ensures that you are using the latest version of calicoctl and its accompanying configuration.
If you choose to [install calicoctl as a binary on a single host](#installing-calicoctl-as-a-binary-on-a-single-host),
we recommend you uninstall any versions of calicoctl that may have shipped alongside OpenShift with the following commands.

```
rm /usr/local/bin/calicoctl
rm /etc/calico/calicoctl.cfg
```

## Installing calicoctl as a binary on a single host

{% include {{page.version}}/ctl-binary-install.md cli="calicoctl" codepath="/calicoctl" %}

**Next step**:

[Configure `calicoctl` to connect to your datastore](configure).

{% include {{page.version}}/ctl-container-install.md cli="calicoctl" %}
