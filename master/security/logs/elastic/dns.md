---
title: DNS logs
canonical_url: https://docs.tigera.io/v2.5/usage/logs/elastic/dns
---

{{site.prodname}} pushes DNS activity logs to Elasticsearch. The following table
details the key/value pairs in the JSON blob, including their
[Elasticsearch datatype](https://www.elastic.co/guide/en/elasticsearch/reference/current/mapping-types.html).
This information should assist you in constructing queries.

| Name                  | Datatype          | Description |
| --------------------- | ----------------- | ----------- |
| `start_time`          | date              | When the collection of the log began in UNIX timestamp format. |
| `end_time`            | date              | When the collection of the log concluded in UNIX timestamp format. |
| `count`               | long              | How many DNS lookups there were, during the log collection interval, with details matching this log.  |
| `client_ip`           | ip                | The IP address of the client pod. A hyphen indicates aggregation. |
| `client_name`         | keyword           | {::nomarkdown}<p>This field contains one of the following values:<br>&#x25cf;&nbsp;The name of the client pod.<br>&#x25cf;&nbsp;<code>-</code>: the name of the pod was aggregated. Check <code>client_name_aggr</code> for the pod name prefix.</p>{:/} |
| `client_name_aggr`    | keyword           | The aggregated name of the client pod. |
| `client_namespace`    | keyword           | Namespace of the client pod. |
| `client_labels`       | array of keywords | Labels applied to the client pod. With aggregation, the label name/value pairs that are common to all aggregated clients. |
| `qname`               | keyword           | The domain name that was looked up. |
| `qtype`               | keyword           | The type of the DNS query (e.g. A, AAAA). |
| `qclass`              | keyword           | The class of the DNS query (e.g. IN). |
| `rcode`               | keyword           | The result code of the DNS query response (e.g. NoError, NXDomain). |
| `rrsets`              | nested            | Detailed DNS query response data - see below. |
| `servers`             | nested            | Details of the DNS servers that provided this response. |

Each nested `rrsets` object contains response data for a particular name and a particular type and
class of response information.  Its key/value pairs are as follows.

| Name                  | Datatype          | Description |
| --------------------- | ----------------- | ----------- |
| `name`                | keyword           | The domain name that this information is for. |
| `type`                | keyword           | The type of the information (e.g. A, AAAA). |
| `class`               | keyword           | The class of the information (e.g. IN). |
| `rdata`               | array of keywords | Array of data, for the name, of that type and class.  For example, when `type` is A, this is an array of IPs for `name`. |

Each nested `servers` object provides details of a DNS server that provided the information in the
containing log.  Its key/value pairs are as follows.

| Name             | Datatype          | Description |
| ---------------- | ----------------- | ----------- |
| `ip`             | ip                | The IP address of the DNS server. |
| `name`           | keyword           | {::nomarkdown}<p>This field contains one of the following values:<br>&#x25cf;&nbsp;The name of the DNS server pod.<br>&#x25cf;&nbsp;<code>-</code>: the DNS server is not a pod.</p>{:/} |
| `name_aggr`      | keyword           | {::nomarkdown}<p>This field contains one of the following values:<br>&#x25cf;&nbsp;The aggregated name of the DNS server pod.<br>&#x25cf;&nbsp;<code>pvt</code>: the DNS server is not a pod. Its IP address belongs to a private subnet.<br>&#x25cf;&nbsp;<code>pub</code>: the DNS server is not a pod. Its IP address does not belong to a private subnet. It is probably on the public internet.</p>{:/} |
| `namespace`      | keyword           | Namespace of the DNS server pod. |
| `labels`         | array of keywords | Labels applied to the DNS server pod. |

## Queries on top-level fields

Queries on any of the top-level fields are supported in Kibana and can also be done with the
Elasticsearch API, for example:

```shell
curl 'http://10.111.1.235:9200/tigera_secure_ee_dns.cluster.20190711/_search?q=qname:example.com&pretty=true'
```

or

```shell
curl 'http://10.111.1.235:9200/tigera_secure_ee_dns.cluster.20190711/_search?pretty=true' \
    -d '{"query": {"match": {"qname": "example.com"}}}' \
    -H "Content-Type: application/json"
```

## Queries on nested fields

Queries on nested object fields are not supported by Kibana, but can be done with the Elasticsearch
API, for example:

```shell
curl 'http://10.111.1.235:9200/tigera_secure_ee_dns.cluster.20190711/_search?pretty=true' \
    -d '{"size": 2, "query": {"nested": {"path": "rrsets", "query": {"match": {"rrsets.type": "CNAME"}}}}}' \
    -H "Content-Type: application/json"
```
