---
title: Component logs
redirect_from: latest/usage/logs/component
canonical_url: https://docs.tigera.io/v2.3/usage/logs/component
---

The {{site.prodname}} components normally run as pods; their logs can be accessed as usual for pod logs (`kubectl logs` etc).  The [Kubernetes](https://kubernetes.io/docs/concepts/cluster-administration/logging/) and
[OpenShift](https://docs.openshift.com/container-platform/3.9/install_config/aggregate_logging.html) logging pages describe how to collect and manage those logs.

When running Felix for host protection (not as part of the {{site.noderunning}} container), it writes its logs to a file.
That file can be configured as part of the [Felix configuration options](../../reference/felix/configuration) and log collection/rotation can be set up
as normal for a file.
