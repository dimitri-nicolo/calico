---
title: Using your own Elasticsearch for logs
canonical_url: https://docs.tigera.io/v2.3/getting-started/kubernetes/installation/byo-elasticsearch
---

{% include {{page.version}}/byo-intro.md %}

## Completing a production install using your own Elasticsearch

### Set up access to your cluster from Kubernetes

{% include {{page.version}}/elastic-secure.md %}

### Installing Prometheus, Alertmanager, and Fluentd

{% include {{page.version}}/cnx-monitor-install.md elasticsearch="external" %}

{% include {{page.version}}/cnx-mgr-install.md init="kubernetes" elasticsearch="external" %}

{% include {{page.version}}/gs-next-steps.md %}
