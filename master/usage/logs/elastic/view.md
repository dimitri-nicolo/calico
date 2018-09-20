---
title: Accessing logs
---

## About accessing logs

You can access the Elasticsearch logs from:
- [{{site.prodname}} Manager](#view-in-mgr)
- [Kibana or Elasticsearch API](#accessing-logs-from-kibana-or-the-elasticsearch-api)

## <a name="view-in-mgr"></a>Viewing logs in the {{site.prodname}} Manager

To view flow log visualizations from the {{site.prodname}} Manager, open the **Flow Visualizations** pane
and enter a query. An example of a flow log visualization for the query
`Filter: Source type: List [ "net", "ns", "wep", "hep" ], Destination type: List [ "net", "ns", "wep", "hep" ], Time range: 24` follows.

![Example flow log visualization]({{site.baseurl}}/images/flow-log-visualization.png)

## Accessing logs from Kibana or the Elasticsearch API

You can access the logs from a Kibana dashboard or via the
[Elasticsearch Search API](https://www.elastic.co/guide/en/elasticsearch/reference/current/search.html)
under the following indices.

- **Flow logs**: `tigera_secure_ee_flows`

- **Audit logs**: `tigera_secure_ee_audit*`
