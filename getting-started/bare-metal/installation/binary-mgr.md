---
title: Binary install with package manager
description: Install Calico on non-cluster host using a package manager.
canonical_url: '/getting-started/bare-metal/installation/binary-mgr'
---

### Big picture
Install {{site.prodname}} on non-cluster hosts using a package manager.

### Value
Packaged binaries of {{site.prodname}} are easy to consume and upgrade. This method automatically configures the init system to keep Felix running.

### Before you begin

1. Ensure the {{site.prodname}} datastore is up and accessible from the host
2. Ensure the host meets the minimum [system requirements](../requirements)
3. If your system is not an Ubuntu- or RedHat-derived system, you will need to choose a different install method.
4. If you want to install {{site.prodname}} with networking (so that you can communicate with cluster workloads), you should choose the [container install method](./container)
5. [Install and configure `calicoctl`]({{site.baseurl}}/maintenance/clis/calicoctl/)

### How to

This guide covers installing Felix, the {{site.prodname}} daemon that handles network policy.

#### Step 1: (Optional) Create a kubeconfig for the host
In order to run Calico Node as a binary, it will need a kubeconfig. You can skip this step if you already have a kubeconfig ready to use.

{% include content/create-kubeconfig.md %}

#### Step 2: (Optional) Grant the read-only RBAC permissions to your service account
Run the following two commands to create a cluster role with read-only access and a corresponding cluster role binding.

```bash
kubectl apply -f {{ "/manifests/non-cluster-host-clusterrole.yaml" | absolute_url }}
kubectl create clusterrolebinding $HOST_NAME --serviceaccount=calico-system:$HOST_NAME --clusterrole=non-cluster-host
```

#### Step 3: Install binaries

{% include ppa_repo_name %}

*PPA requires*: Ubuntu 14.04 or 16.04

    sudo add-apt-repository ppa:project-calico/{{ ppa_repo_name }}
    sudo apt-get update
    sudo apt-get upgrade
    sudo apt-get install calico-felix

*RPM requires*: RedHat 7-derived distribution

    cat > /etc/yum.repos.d/calico.repo <<EOF
    [calico]
    name=Calico Repository
    baseurl=http://binaries.projectcalico.org/rpm/{{ ppa_repo_name }}/
    enabled=1
    skip_if_unavailable=0
    gpgcheck=1
    gpgkey=http://binaries.projectcalico.org/rpm/{{ ppa_repo_name }}/key
    priority=97
    EOF

    yum install calico-felix

Until you initialize the database, Felix will make a regular log that it
is in state "wait-for-ready". The default location for the log file is
`/var/log/calico/felix.log`.

#### Step 4: Configure the datastore connection

{% include content/environment-file.md target="felix" %}

Modify the included init system unit to include the `EnvironmentFile`.  For example, on systemd, add the following line to the `[Service]` section of the `calico-felix` unit.

```
EnvironmentFile=/etc/calico/calico.env
```

#### Step 5: Initialize the datastore

{% include content/felix-init-datastore.md %}
