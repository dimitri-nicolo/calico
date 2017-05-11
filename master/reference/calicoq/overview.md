---
title: calicoq user reference
---

The command line tool, `calicoq`, makes it easy to check what your Calico security
policies apply to.

It can be downloaded from the [releases page of the
calicoctl repository](https://github.com/projectcalico/calicoctl/releases/latest/). TODO UPDATE LINK

calicoq uses the same setup as calicoctl.  Follow the setup in the [Configuing calicoctl]({{site.baseurl}}/{{page.version}}/reference/calicoctl/setup) section.
This section describes how to do the initial setup of the Calico command line tools, configuring
the connection information for your Calico datastore.

The calicoq command line interface provides a number of policy inspection
commands to allow you to confirm that your security policies are configured
as intended.

- The [host]({{site.baseurl}}/{{page.version}}/reference/calicoq/host) command displays the policies that apply to endpoints on a host.
- The [eval]({{site.baseurl}}/{{page.version}}/reference/calicoq/eval) command displays the workloads that a selector selects.
- The `calicoq version` command displays the version of the tool.

