---
title: Tigera Secure EE Log Management
---

{{site.prodname}} produces three types of logs - audit logs, iptables logs and logs from the {{site.prodname}} components themselves.
This document describes how to access and manage those logs.

## Audit logs

Audit logs for key changes to {{site.prodname}} configuration are produced using the Kubernetes / OpenShift audit logging mechanism.

Links to the documentation for managing these logs are provided for [Kubernetes](auditing) and [OpenShift](openshift-auditing).
The Kubernetes / OpenShift documentation describes how to configure batching, collecting and managing those logs.

## iptables logs

iptables logs are produced by [policy audit mode](policy-auditing) or by using the [`log` action in Policy Rules](../calicoctl/resources/networkpolicy).
These logs are written to syslog (specifically the `/dev/log` socket) on the nodes where the events are generated.
Collection, rotation and other management of these logs is provided by your syslog agent - for example journald or rsyslogd.

## {{site.prodname}} component logs

The {{site.prodname}} components normally run as pods; their logs can be accessed as usual for pod logs (`kubectl logs` etc).  The [Kubernetes](https://kubernetes.io/docs/concepts/cluster-administration/logging/) and
[OpenShift](https://docs.openshift.com/container-platform/3.9/install_config/aggregate_logging.html) logging pages describe how to collect and manage those logs.

When running Felix for host protection (not as part of the {{site.noderunning}} container), it writes its logs to a file.
That file can be configured as part of the [Felix configuration options](../felix/configuration); and log collection / rotation can be set up
as normal for a file.