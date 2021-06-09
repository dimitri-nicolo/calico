---
title: Install Calico Enterprise on host endpoints
description: Choose a method to install Calico Enterprise on hosts.
canonical_url: '/getting-started/bare-metal/installation/overview'
---

You will need to install `calicoctl` and configure it to connect to your datastore.

-  [Install calicoctl as a binary]({{site.baseurl}}/maintenance/clis/calicoctl/install).

-  [Configure calicoctl to connect to database]({{site.baseurl}}/maintenance/clis/calicoctl/configure/database).

Then you can use any of the following methods to install and run Felix, on each bare metal
host where you want {{site.prodname}} host protection.

- [Container](container): On hosts equipped with Docker, you can run `{{site.nodecontainer}}`,
  which includes Felix and all of its dependencies.

- [Binary without package manager](binary): If you prefer not to run Docker on all of your
  hosts, you can use Docker in one place to extract the `{{site.noderunning}}` binary from a
  `{{site.nodecontainer}}` container image, then copy that binary to each of your hosts and
  run it as `{{site.noderunning}} -felix`.
