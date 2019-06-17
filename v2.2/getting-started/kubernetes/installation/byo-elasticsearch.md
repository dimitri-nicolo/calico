---
title: Using your own Elasticsearch for logs
---

{% include {{page.version}}/byo-intro.md %}

## Completing a production install using your own Elasticsearch

### Set up access to your cluster from Kubernetes

{% include {{page.version}}/elastic-secure.md %}

### Installing Prometheus, Alertmanager, and Fluentd

{% include {{page.version}}/cnx-monitor-install.md elasticsearch="external" %}

{% include {{page.version}}/gs-next-steps.md %}
