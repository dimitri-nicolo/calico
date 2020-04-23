---
title: BGP logs
description: Key/value pairs of BGP activity logs and how to construct queries. 
canonical_url: /security/logs/elastic/bgp
---
{{site.prodname}} pushes BGP activity logs to Elasticsearch. To view them, follow the steps below.
 
1. Log in to the {{site.prodname}} Manager. For more information on how to access the Manager, see [Configure access to the manager UI]({{site.baseurl}}/getting-started/cnx/access-the-manager).
2. Click on Kibana from the side navigation.
3. Log in to the Kibana dashboard. For more information on how to log in to your Kibana dashboard, see [Accessing logs from Kibana]({{site.baseurl}}/security/logs/elastic/view#accessing-logs-from-kibana)
4. Navigate to Discovery view and and select `tigera_secure_ee_bgp.*` from the dropdown menu to view all collected logs from BIRD and BIRD6.

The following tables details key/value pairs, including their Elasticsearch datatype. This information should assist you in constructing queries.

| Name                  | Datatype          | Description |
| --------------------- | ----------------- | ----------- |
| `logtime`             | date              | When the log was collected in UTC timestamp format. |
| `host`                | keyword           | When the collection of the log concluded in UNIX timestamp format. |
| `ip_version`          | keyword           | This field contains one of the following values:<br>&#x25cf;&nbsp;<code>IPv4</code>: Indicates the log is from the BIRD process <br>&#x25cf;&nbsp;<code>IPv6</code>: Indicates the log is from the BIRD6 process. |
| `message`             | text              | The message contained in the log. |

Once a set of BGP logs has accumulated in Elasticsearch, you can perform many interesting queries. For example:
* with a query on the `ip_version` field and sorting by the `logtime` field , you could view only BGP logs related to only IPv4 or IPv6.
* with a query on the `host` field, you could see all the logs from a specified node
* with a query on the `message` field, you could see when certain events occurred in the cluster. For example, when BIRD was restarted.

Different techniques are needed, depending on the field that you want to query on.