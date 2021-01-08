---
title: Flow logs for Windows workload 
description: Flow logs for Calico Enterprise for Windows.
canonical_url: '/getting-started/windows-calico/flowlogs'
---

### Big picture

Use {{site.prodnameWindows}} flow log data for visibility and troubleshooting Windows workloads in Kubernetes clusters.
This is part of flow logs feature of {{site.prodname}}. 

### How to

{{site.prodnameWindows}} flow log can be enabled and configured in the same way as the Linux couterpart. 

The configuration items specific on Windows nodes are as followings: 

#### Felix Configurations

| Field                              | Description                 | Accepted Values   | Schema | Default    |
|------------------------------------|-----------------------------|-------------------|--------|------------|
| windowsFlowLogsFileDirectory | Set the directory where flow logs files are stored on Windows nodes. This parameter only takes effect when `flowLogsFileEnabled` is set to `true`. | string | string | `c:\\TigeraCalico\\flowlogs` |
| windowsFlowLogsPositionFilePath | Specify the position of the external pipeline that reads flow logs on Windows nodes. This parameter only takes effect when `FlowLogsDynamicAggregationEnabled` is set to `true`. | string | string | `c:\\TigeraCalico\\flowlogs\\flows.log.pos` |
| windowsStatsDumpFilePath | Specify the position of the file used for dumping flow log statistics on Windows nodes. Note this is an internal setting that users shouldn't need to modify.| string | string | `c:\\TigeraCalico\\stats\\dump` |
 
### Limitations

There are couple of limitations with {{site.prodnameWindows}} flow log.
- No packet/bytes stats for denied traffic.
- No DNS stats.
- No Http stats.
- No RuleTrace for tiers. 
- No BGP logs

### Above and beyond

- [Log storage requirements]({{site.baseurl}}/maintenance/logstorage/log-storage-requirements)
- [Configure RBAC for Elasticsearch logs]({{site.baseurl}}/visibility/elastic/rbac-elasticsearch)
- [Configure flow log aggregation]({{site.baseurl}}/visibility/elastic/flow/aggregation)
- [Archive logs]({{site.baseurl}}/visibility/elastic/archive-storage)
- [Log collection options]({{site.baseurl}}/reference/installation/api#operator.tigera.io/v1.LogCollectorSpec)
