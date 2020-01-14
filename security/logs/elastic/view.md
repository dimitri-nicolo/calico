---
title: Access logs
canonical_url: https://docs.tigera.io/v2.3/usage/logs/elastic/view
---

## About accessing logs

You can access the Elasticsearch logs from:
- [{{site.prodname}} Manager](#view-in-mgr)
- [Kibana](#accessing-logs-from-kibana)
- [Elasticsearch API](#accessing-logs-from-the-elasticsearch-api)

## <a name="view-in-mgr"></a>Viewing logs in the {{site.prodname}} Manager

To view flow log visualizations from the {{site.prodname}} Manager, open the **Flow Visualizations** pane
and enter a query. An example of a flow log visualization for the query
`Filter: Source type: List [ "net", "ns", "wep", "hep" ], Destination type: List [ "net", "ns", "wep", "hep" ], Time range: 24` follows.

![Example flow log visualization]({{site.url}}/images/flow-log-visualization.png)

## Accessing logs from Kibana

You can access Kibana by clicking the **Kibana** button in the {{site.prodname}} Manager.

A default user `elastic` is created and stored in the `tigera-secure-es-elastic-user` secret during installation. You can obtain the password using the following command:

   ```
kubectl -n tigera-elasticsearch get secret tigera-secure-es-elastic-user -o yaml |  awk '/elastic:/{print $2}' | base64 --decode
   ```

## Accessing logs from the Elasticsearch API

You can access the logs from the
[Elasticsearch Search API](https://www.elastic.co/guide/en/elasticsearch/reference/current/search.html)
under the following indices.

- **Flow logs**: `tigera_secure_ee_flows*`

- **Audit logs**: `tigera_secure_ee_audit*`

- **DNS logs**: `tigera_secure_ee_dns*`
