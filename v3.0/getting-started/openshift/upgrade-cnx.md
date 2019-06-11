---
title: Upgrading to CNX in OpenShift
---

This document covers upgrading an open source Calico cluster to {{site.prodname}}.

The upgrade procedure is supported for Calico v3.0.x.

## Upgrading an open source Calico cluster to {{site.prodname}}

#### Prerequisites

This procedure assumes the following:

- Your system is running the latest 3.0.x release of Calico.
- You have obtained the {{site.prodname}} specific binaries by following the instructions in [getting started]({{site.baseurl}}/{{page.version}}/getting-started/) and uploaded them to a private registry.

#### Upgrade

1. Edit your openshift-ansible inventory file, setting `calico_node_image`
   to the `tigera/cnx-node` image in your private registry.

2. Re-run the ansible provisioner:

       ansible-playbook -i ./inventory.ini /usr/share/ansible/openshift-ansible/playbooks/byo/config.yml

3. Log onto each node and restart `calico.service` to pick up the changes:

       systemctl daemon-reload && systemctl restart calico

Once complete, skip to step 4. of [the CNX installation doc](cnx/installation) for information on launching CNX Manager.