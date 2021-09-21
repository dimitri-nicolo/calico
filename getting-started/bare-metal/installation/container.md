---
title: Docker container install
description: Install Calico Enterprise on non-cluster hosts using a Docker container.
canonical_url: '/getting-started/bare-metal/installation/container'
---

### Big picture
Install {{site.prodname}} on non-cluster hosts using a Docker container for both networking and policy.

### Value
Installing {{site.prodname}} with a Docker container includes everything you need for both networking and policy.

### Before you begin

1. Ensure Docker is installed
2. Ensure the {{site.prodname}} datastore is up and accessible from the host
3. Ensure the host meets the minimum [system requirements](../requirements)
4. Create a kubeconfig for your node and bind it to an RBAC role. The minimum permissions can be found in [this manifest]({{ "/manifests/non-cluster-host-clusterrole.yaml" | absolute_url }}).

### How to

The `{{site.nodecontainer}}` container should be started at boot time by your init system and the init system must be configured to restart it if stopped. {{site.prodname}} relies on that behavior for certain configuration changes.
{% include content/docker-container-service.md %}

