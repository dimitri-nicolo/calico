---
title: Overview
description: The default deployment for data collection and visualization for Calico Enterprise. 
canonical_url: /security/logs/elastic/
---

{{site.prodname}} uses an Elasticsearch operator to deploy an Elasticsearch cluster and a Kibana instance. The 
Elasticsearch cluster is used to store all {{site.prodname}} related data according to the specified retention 
settings to make sure the cluster does not run out of disk space.

In this section we will elaborate on the specifics of the collected data and how to access and configure your cluster:
* [Flow logs]({{site.baseurl}}/security/logs/elastic/datatypes)
* [Audit logs]({{site.baseurl}}/security/logs/elastic/ee-audit)
* [DNS logs]({{site.baseurl}}/security/logs/elastic/dns)
* [BGP logs]({{site.baseurl}}/security/logs/elastic/bgp)
* [Flow log visualization]({{site.baseurl}}/security/logs/elastic/view#view-in-mgr)
* [Kibana]({{site.baseurl}}/security/logs/elastic/view#accessing-logs-from-kibana)
* [Elasticsearch API]({{site.baseurl}}/security/logs/elastic/view#accessing-logs-from-the-elasticsearch-api)
* [Data retention]({{site.baseurl}}/security/logs/retention)
* [Filtering flow logs]({{site.baseurl}}/security/logs/elastic/filtering)
* [Filtering DNS logs]({{site.baseurl}}/security/logs/elastic/filtering-dns)
* [Archive logs to storage]({{site.baseurl}}/security/logs/elastic/archive-storage)
* [Tracing external IP addresses]({{site.baseurl}}/security/ingress)
