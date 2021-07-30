---
title: Enabling Process executable path and arguments in flow logs
description: Enabling Process executable path and arguments in flow logs
canonical_url: /visibility/elastic/flow/processpath
---


### Big picture

Configure {{site.prodname}} to collect process executable path and the arguments with which the executable was invoked. The path and arguments are read from `/proc/pid/cmdline` or obtained using eBPF kprobes and add them to flow logs.

### Value

Get visibility into the network activity at the process level using {{site.prodname}} flow logs.

### Privileges

For full functionality, this feature requires the calico-node daemonset to have access to the host's PID namespace. The Tigera Operator will automatically grant this extra privilege to the daemonset if the feature is enabled in the operator's LogCollector resource, as described below.

### Concepts

#### eBPF kprobe programs

eBPF is a Linux kernel technology that allows safe mini-programs to be attached to various hooks inside the kernel. To collect the path and arguments of short-lived processes, this feature uses an eBPF kprobe program.

#### Reading from /proc/pid

For processes which were spawned before kprobes are attached, path and arguments are read from /proc/pid/cmdline.

### Before you begin

Ensure that your kernel contains support for eBPF kprobes that {{site.prodname}} uses. The minimum supported
kernel for this is feature is: `v4.4.0`.

## How to

#### Enable process path and argument collection

{{site.prodname}} can be configured to enable process path and argument collection on supported Linux kernels
using the command:

```
 kubectl patch logcollector.operator.tigera.io tigera-secure --type merge -p '{"spec":{"collectProcessPath":"Enabled"}}'
```

Enabling/Disabling collectProcessPath causes a rolling update of the `calico-node`.

#### View process path and arguments in flow logs using Kibana.

Navigate to the Kibana Flow logs dashboard to view process path and arguments associated with a flow log entry.

The executable path will appear in the `process_name` field and `process_args` will have the executable arguments. Executable path
and arguments cannot be collected under certain circumstances, in that `process_name` will have the task name and `process_args`
will be empty. Information about these fields are described in the [Flow log datatype document](datatypes)

