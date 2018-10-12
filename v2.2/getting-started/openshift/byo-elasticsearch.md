---
title: Using your own Elasticsearch for logs
redirect_from: latest/getting-started/openshift/byo-elasticsearch
---

{% include {{page.version}}/byo-intro.md orch="openshift" %}

## Completing a production install using your own Elasticsearch

### Set up namespace

1. Create the `calico-monitoring` namespace to store configuration and set up other {{site.prodname}} components

   ```
   kubectl create -f - <<EOF
   apiVersion: v1
   kind: Namespace
   metadata:
     name: calico-monitoring
   EOF
   ```

### Set up access to your cluster from Kubernetes

{% include {{page.version}}/elastic-secure.md orch="openshift" %}

### Installing Prometheus, Alertmanager, and Fluentd

{% include {{page.version}}/cnx-monitor-install.md orch="openshift" elasticsearch="external" %}

{% include {{page.version}}/gs-openshift-next-steps.md %}
