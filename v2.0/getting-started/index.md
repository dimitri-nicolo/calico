---
title: Getting started with CNX
canonical_url: https://docs.tigera.io/v2.3/getting-started/
---

## Obtain the private binaries and images

### Images

Contact your support representative to gain access to the following images.

   | File name                                                                      | Description                   |
   | ------------------------------------------------------------------------------ | ----------------------------- |
   | `tigera-calicoctl_{{site.data.versions[page.version].first.title}}.tar.xz`     | `calicoctl`                   |
   | `tigera-calicoq_{{site.data.versions[page.version].first.title}}.tar.xz`       | `calicoq`                     |
   | `tigera-cnx-apiserver_{{site.data.versions[page.version].first.title}}.tar.xz` | {{site.tseeprodname}} API server  |
   | `tigera-cnx-manager_{{site.data.versions[page.version].first.title}}.tar.xz`   | {{site.tseeprodname}} Manager     |
   | `tigera-cnx-node_{{site.data.versions[page.version].first.title}}.tar.xz`      | `{{site.nodecontainer}}`      |
   | `tigera-typha_{{site.data.versions[page.version].first.title}}.tar.xz`         | Typha                         |

### Binaries

Contact your support representative to gain access to the following binaries.

   | File name                                                    | Description  |
   | ------------------------------------------------------------ | ------------ |
   | `calicoctl_{{site.data.versions[page.version].first.title}}` | `calicoctl`  |
   | `calicoq_{{site.data.versions[page.version].first.title}}`   | `calicoq`    |
   | `felix_{{site.data.versions[page.version].first.title}}`     | Felix        |

## Choose your orchestrator

To get started using {{site.tseeprodname}}, we recommend running
through one or more of the available tutorials linked below.

These tutorials will help you understand the different environment options when
using {{site.tseeprodname}}.  In most cases we provide worked examples using manual setup on
your own servers, a quick set-up in a virtualized environment using Vagrant and
a number of cloud services.

Then follow one of our getting started guides.
- [{{site.tseeprodname}} with Kubernetes](kubernetes/)
- [{{site.tseeprodname}} with OpenShift](openshift/installation)
- [Host protection](bare-metal/bare-metal)
