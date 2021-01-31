---
title: Enabling process information in flow logs
description: Enabling process information in flow logs
canonical_url: /visibility/elastic/flow/process
---


### Big picture

Configure {{site.prodname}} flow logs to collect process information using eBPF kprobes.

### Value

Get visibility into the network activity at the process level using {{site.prodname}} flow logs.

### Concepts

#### eBPF kprobes

The Linux kernel provides the ability to attach eBPF programs to various "tracepoints" in the kernel. These
probes can provide insights into network activity. {{site.prodname}} leverages this to enrich network
activity flow logs with process information.


### Before you begin

Ensure that your kernel contains support for kprobes that {{site.prodname}} uses. The minimum supported
kernel for process information is: `4.3.0`.

## How to

#### Enable process information collection

{{site.prodname}} can be configured to enable process information collection on supported Linux kernels
using the command:

```
 kubectl patch felixconfiguration default -p '{"spec":{"flowLogsCollectProcessInfo":true}}'
```

#### View process information in flow logs using Kibana.

Navigate to the Kibana Flow logs dashboard to view process information associated with a flow log entry.

The additional fields collected are `process_name`, `num_process_names`, `process_id`, and `num_process_ids`.
Information about these fields are described in the [Flow log datatype document](datatypes)
