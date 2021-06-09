---
title: Binary install without package manager
description: Install Calico Enterprise binary on non-cluster hosts without a package manager.
canonical_url: '/getting-started/bare-metal/installation/binary'
---

### Big picture
Install {{site.prodname}} binary on non-cluster hosts without a package manager.

### Value
Install {{site.prodname}} directly when a package manager isn't available, or your provisioning system can easily handle copying binaries to hosts.

### Before you begin

1. Ensure the {{site.prodname}} datastore is up and accessible from the host
1. Ensure the host meets the minimum [system requirements](../requirements)
1. If you want to install {{site.prodname}} with networking (so that you can communicate with cluster workloads), you should choose the [container install method](./container)
1. [Install and configure `calicoctl`]({{site.baseurl}}/maintenance/clis/calicoctl/)

### How to

This guide covers installing Felix, the {{site.prodname}} daemon that handles network policy.

#### Step 1: Download and extract the binary

This step requires Docker, but it can be run from any machine with Docker installed. It doesn't have to be the host you will run it on (i.e your laptop is fine).

1. Use the following command to download the {{site.nodecontainer}} image.

   ```bash
   docker pull {{page.registry}}{% include component_image component="cnx-node" %}
   docker pull {{site.nodecontainer}}:{{site.data.versions.first.components["calico/node"].version}}
   ```

1. Confirm that the image has loaded by typing `docker images`.
{%- assign n = site.data.versions.first.components["cnx-node"] %}

   ```
   REPOSITORY       TAG           IMAGE ID       CREATED         SIZE
   {{page.registry}}{{ n.image }}      {{ n.version }}        e07d59b0eb8a   2 minutes ago   42MB
   ```
   {: .no-select-button}

1. Create a temporary {{site.nodecontainer}} container.

   ```bash
   docker create --name container {{page.registry}}{% include component_image component="cnx-node" %}
   ```

1. Copy the calico-node binary from the container to the local file system.

   ```bash
   docker cp container:/bin/calico-node {{site.nodecontainer}}
   ```

1. Delete the temporary container.

   ```bash
   docker rm container
   ```

1. Set the extracted binary file to be executable.

   ```bash
   chmod +x {{site.nodecontainer}}
   ```

#### Step 2: Copy the `calico-node` binary

Copy the binary from Step 1 to the target machine, using any means (`scp`, `ftp`, USB stick, etc.).

#### Step 3: Create environment file

{% include content/environment-file.md install="binary" target="felix" %}

#### Step 4: Create a start-up script

Felix should be started at boot by your init system and the init system
**must** be configured to restart Felix if it stops. Felix relies on
that behavior for certain configuration changes.

If your distribution uses systemd, then you could use the following unit
file:

    [Unit]
    Description=Calico Felix agent
    After=syslog.target network.target

    [Service]
    User=root
    EnvironmentFile=/etc/calico/calico.env
    ExecStartPre=/usr/bin/mkdir -p /var/run/calico
    ExecStart=/usr/local/bin/{{site.nodecontainer}} -felix
    KillMode=process
    Restart=on-failure
    LimitNOFILE=32000

    [Install]
    WantedBy=multi-user.target

Or, for upstart:

    description "Felix (Calico agent)"
    author "Project Calico Maintainers <maintainers@projectcalico.org>"

    start on stopped rc RUNLEVEL=[2345]
    stop on runlevel [!2345]

    limit nofile 32000 32000

    respawn
    respawn limit 5 10

    chdir /var/run

    pre-start script
      mkdir -p /var/run/calico
      chown root:root /var/run/calico
    end script

    exec /usr/local/bin/{{site.nodecontainer}} -felix

## Configure Felix

Optionally, you can create a file at `/etc/calico/felix.cfg` to
configure Felix. The configuration file as well as other options for
configuring Felix (including environment variables) are described in
[this]({{site.baseurl}}/reference/felix/configuration) document.

Felix tries to detect whether IPv6 is available on your platform but
the detection can fail on older (or more unusual) systems.  If Felix
exits soon after startup with `ipset` or `iptables` errors try
setting the `Ipv6Support` setting to `false`.

### Kubernetes datastore

To configure Felix to interact with a Kubernetes datastore,
it is essential to set the `DatastoreType` setting to `kubernetes`.
You will also need to set the environment variable `CALICO_KUBECONFIG`
to point to a valid kubeconfig for your kubernetes cluster and
`CALICO_NETWORKING_BACKEND` to `none`.

> **Note**: Felix only works in policy-only mode when running
in Kubernetes datastore mode. This means that pod networking is
disabled on the baremetal host Felix is running on but policy can
still be used to secure the host.
{: .alert .alert-info}

## Start Felix

Once you've configured Felix, start it up via your init system.

```bash
service calico-felix start
```
#### Step 5: Initialize the datastore

{% include content/felix-init-datastore.md %}

For debugging, it's sometimes useful to run Felix manually and tell it
to emit its logs to screen. You can accomplish that using the appropriate
command depending on what mode Felix is running in.

```bash
FELIX_DATASTORETYPE=kubernetes CALICO_KUBECONFIG=<YOUR_KUBECONFIG_PATH> FELIX_LOGSEVERITYSCREEN=INFO /usr/local/bin/{{site.nodecontainer}} -felix
```
