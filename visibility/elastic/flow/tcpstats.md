---
title: Enabling TCP socket stats in flow logs
description: Enabling TCP socket stats information in flow logs
canonical_url: /visibility/elastic/flow/tcpstats
---


### Big picture

Configure {{site.prodname}} flow logs to collect TCP socket stats using eBPF TC programs

### Value

Get visibility into the network activity at the socket level using {{site.prodname}} flow logs.

### Concepts

#### eBPF TC programs

The Linux kernel provides the ability to attach eBPF programs to the tc layer of an interface.
Programs are attached to the ingress hook of the workload interfaces on the host. This program does a socket lookup 
in the appropriate namespace and gets the socket stats. {{site.prodname}} leverages this to enrich network
activity flow logs with tcp socket stats information.


### Before you begin

Ensure that your kernel contains support for eBPF that {{site.prodname}} uses. The minimum supported
kernel for tcp socket stats is: `5.3.0`.

## How to

#### Enable tcp stats collection

{{site.prodname}} can be configured to enable tcp socket stats collection on supported Linux kernels
using the command:

```
 kubectl patch felixconfiguration default -p '{"spec":{"flowLogsCollectTcpStats":true}}'
```
#### View tcp stats in flow logs using Kibana.

Navigate to the Kibana Flow logs dashboard to view tcp stats associated with a flow log entry.

The additional fields collected are `tcp_mean_send_congestion_window`, `tcp_min_send_congestion_window`, `tcp_mean_smooth_rtt`, `tcp_max_smooth_rtt`, 
`tcp_mean_min_rtt`, `tcp_max_min_rtt`, `tcp_mean_mss`, `tcp_min_mss`, `tcp_total_retransmissions`, `tcp_lost_packets`, `tcp_unrecovered_to`.
Information about these fields are described in the [Flow log datatype document](datatypes)

