---
title: Getting started with CNX
---

## Obtain the private binaries

Contact your support representative to gain access to the following files. 

   | TAR file                                       | Description                                |
   | ---------------------------------------------- | ------------------------------------------ |
   | `tigera-calicoctl_v2.0.0-cnx-beta1.tar.xz`     | {{site.prodname}}-enhanced `calicoctl`     |
   | `tigera-cnx-apiserver_v2.0.0-cnx-beta1.tar.xz` | Kubernetes extension API server component  |
   | `tigera-cnx-node_v2.0.0-cnx-beta1.tar.xz`      | {{site.prodname}}-enhanced `calico/node`   |
   | `tigera-calicoq_v2.0.0-cnx-beta1.tar.xz`       | Policy query command-line tool             |
   | `tigera-cnx-manager_v2.0.0-cnx-beta1.xz`       | {{site.prodname}} Manager component        | 
   
## Choose your orchestrator

To get started using {{site.prodname}}, we recommend running
through one or more of the available tutorials linked below.

These tutorials will help you understand the different environment options when
using {{site.prodname}}.  In most cases we provide worked examples using manual setup on
your own servers, a quick set-up in a virtualized environment using Vagrant and
a number of cloud services.

Then follow one of our getting started guides.
- [{{site.prodname}} with Kubernetes](kubernetes)
- [{{site.prodname}} with Mesos](mesos)
  - [{{site.prodname}} with DC/OS](mesos/installation/dc-os)
- [{{site.prodname}} with Docker](docker)
- [{{site.prodname}} with OpenStack](openstack)
- [Host protection](bare-metal/bare-metal)
