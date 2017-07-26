---
title: Calico Integrations
---

To get started using Calico and Tigera Essentials, we recommend running
through one or more of the available tutorials linked below.

These tutorials will help you understand the different environment options when
using Calico.  In most cases we provide worked examples using manual setup on
your own servers, a quick set-up in a virtualized environment using Vagrant and
a number of cloud services.

First download and set up the Tigera Essentials binaries.
- [Obtaining Tigera Essentials](essentials)

Then follow one of our getting started guides.
- [Calico with Kubernetes](kubernetes)
- [Calico with Mesos](mesos)
  - [Calico with DC/OS](mesos/installation/dc-os)
- [Calico with Docker](docker)
- [Calico with OpenStack](openstack)
- [Calico with rkt](rkt)
- [Host protection](bare-metal/bare-metal)

If you're already familiar with Calico, you may wish to consult the
Essentials for Kubernetes [demo](essentials/simple-policy-essentials), which
demonstrates the main features.

For more detailed documentation on Essentials features, see here:
- [Setting up essentials binaries](essentials)
- [calicoq documentation]({{site.baseurl}}/{{page.version}}/reference/calicoq)
- [Denied Packet Notifications]({{site.baseurl}}/{{page.version}}/reference/essentials/policy-violations)
- [Configuring Felix]({{site.baseurl}}/{{page.version}}/reference/felix/configuration)
- [Configuring Prometheus]({{site.baseurl}}/{{page.version}}/usage/configuration/prometheus)
- [Configuring Alertmanager]({{site.baseurl}}/{{page.version}}/usage/configuration/alertmanager)
