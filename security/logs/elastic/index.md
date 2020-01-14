---
title: Overview
canonical_url: https://docs.tigera.io/v2.3/usage/logs/elastic/
---

{{site.prodname}} uses an Elasticsearch operator to deploy an Elasticsearch cluster and a Kibana instance. The 
Elasticsearch cluster is used to store all {{site.prodname}} related data according to the specified retention 
settings to make sure the cluster does not run out of disk space.

In this section we will elaborate on the specifics of the collected data and how to access and configure your cluster:
* [Flow logs](flow)
* [Audit logs](ee-audit)
* [DNS logs](dns)
* [Flow log visualization](view#view-in-mgr)
* [Kibana](view#accessing-logs-from-kibana)
* [Elasticsearch API](view#accessing-logs-from-the-elasticsearch-api)
* [Data Retention](../retention)
* [Filtering flow logs](filtering)
* [Filtering DNS logs](filtering-dns)
* [Archiving to syslog](syslog)
* [Archiving to S3](s3-archive)
* [Tracing external IP addresses](ingress)
