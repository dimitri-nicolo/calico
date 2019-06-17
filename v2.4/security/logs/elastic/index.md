---
title: Overview
redirect_from: latest/security/logs/elastic/index
canonical_url: https://docs.tigera.io/v2.3/usage/logs/elastic/
---

The [Quickstart](../../../getting-started/kubernetes/) uses an Elasticsearch operator to deploy an
Elasticsearch cluster and a Kibana instance. You can use these to explore the feature set on a non-production cluster.

For production, you must set up your own Elasticsearch cluster before [installing {{site.prodname}}](../../../getting-started/kubernetes/installation/).

{{site.prodname}} pushes detailed [flow logs](flow) as well as [audit logs](ee-audit) to Elasticsearch.
The {{site.prodname}} Manager provides [flow log visualization](view#view-in-mgr). You can also use
either [Kibana](view#accessing-logs-from-kibana) or [the Elasticsearch API](view#accessing-logs-from-the-elasticsearch-api)
to query both flow and audit logs.
