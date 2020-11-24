---
title: Overview
description: The default deployment for data collection and visualization for Calico Enterprise. 
canonical_url: /visibility/elastic/overview
---

{{site.prodname}} uses an Elasticsearch operator to deploy an Elasticsearch cluster and a Kibana instance. The 
Elasticsearch cluster is used to store all {{site.prodname}} related data according to the specified retention 
settings to make sure the cluster does not run out of disk space.

In this section we will elaborate on the specifics of the collected data and how to access and configure your cluster:
* [Flow logs]({{site.baseurl}}/visibility/elastic/flow/datatypes)
* [Audit logs]({{site.baseurl}}/visibility/elastic/ee-audit)
* [DNS logs]({{site.baseurl}}/visibility/elastic/dns)
* [BGP logs]({{site.baseurl}}/visibility/elastic/bgp)
* [Flow log visualization]({{site.baseurl}}/visibility/elastic/view#view-in-mgr)
* [Kibana]({{site.baseurl}}/visibility/elastic/view#accessing-logs-from-kibana)
* [Elasticsearch API]({{site.baseurl}}/visibility/elastic/view#accessing-logs-from-the-elasticsearch-api)
* [Data retention]({{site.baseurl}}/visibility/elastic/retention)
* [Filtering flow logs]({{site.baseurl}}/visibility/elastic/flow/filtering)
* [Filtering DNS logs]({{site.baseurl}}/visibility/elastic/filtering-dns)
* [Archive logs to storage]({{site.baseurl}}/visibility/elastic/archive-storage)
