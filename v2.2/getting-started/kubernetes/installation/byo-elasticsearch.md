---
title: Installing Tigera Secure EE monitoring with your own Elasticsearch
---

This page describes how to use a separate Elasticsearch cluster for {{site.prodname}} logs instead of using the bundled
[Elasticsearch operator](https://github.com/upmc-enterprises/elasticsearch-operator),
which is not suitable for production use.

## Before you begin

- Ensure that you have an Elasticsearch cluster set up that meets the 
  [requirements]({{site.baseurl}}/{{page.version}}/getting-started/kubernetes/requirements#elasticsearch-requirements).

- Ensure that you have followed the [installation instructions]({{site.baseurl}}/{{page.version}}/getting-started/kubernetes/installation)
  up until the step to download and apply `operator.yaml`.  This document replaces
  the install instructions from that point (inclusive) onwards.

## Installing {{site.prodname}} monitoring with your own Elasticsearch

### Set up access to your cluster from Kubernetes

{% include {{page.version}}/elastic-secure.md %}

### Installing Prometheus, Alertmanager and fluentd

{% include {{page.version}}/cnx-monitor-install.md elasticsearch="external" %}

{% include {{page.version}}/gs-next-steps.md %}
