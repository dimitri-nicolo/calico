---
title: Enabling HostEndpoint reporting in flow logs
description: Enabling hostendpoint reporting in flow logs
canonical_url: /visibility/elastic/flow/hep
---


### Big picture

Configure {{site.prodname}} flow logs to report HostEndpoint information.

### Value

Get visibility into the network activity at the HostEndpoint level using {{site.prodname}} flow logs.

### Before you begin

**Limitations**

HostEndpoint reporting is only supported on Kubernetes nodes.

### Features

This how-to guide uses the following {{site.prodname}} features:
- **Flow logs**
- **HostEndpoint** resource
- **FelixConfiguration** resource

### How to

#### Enable HostEndpoint reporting

{{site.prodname}} can be configured to enable reporting HostEndpoint metadata in flow logs using the command:

```
 kubectl patch felixconfiguration default -p '{"spec":{"flowLogsEnableHostEndpoint":true}}'
```

### Above and beyond

- [Protect Kubernetes Nodes]({{site.baseurl}}/security/kubernetes-nodes)
