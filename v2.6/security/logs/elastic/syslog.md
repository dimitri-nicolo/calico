---
title: Archiving logs to syslog
redirect_from: latest/security/logs/elastic/syslog
---

## Archiving logs to syslog

{{site.prodname}} supports archiving flow and audit logs to syslog.  This provides
a reliable option for maintaining your compliance data long term.  If you wish to use
this feature, follow the instructions below to configure it.

### Configuring syslog archiving when using the Tigera Operator

In order to copy the flow, audit, and dns logs to syslog, the following configuration is needed:

1. Update the
   [LogCollector](/{{page.version}}/reference/installation/api#operator.tigera.io/v1.LogCollector)
   resource named `calico-enterprise` to include
   a [Syslog section](/{{page.version}}/reference/installation/api#operator.tigera.io/v1.SyslogStoreSpec)
   with your syslog information.
   Example:
   ```
   apiVersion: operator.tigera.io/v1
   kind: LogCollector
   metadata:
     name: calico-enterprise
   spec:
     additionalStores:
       syslog:
         # Syslog endpoint, in the format protocol://host:port
         endpoint: tcp://1.2.3.4:514
         # Packetsize is optional, if messages are being truncated set this
         #packetSize: 1024
   ```
   This can be done during installation by editing the custom-resources.yaml
   before applying it or after installation by editing the resource with the command:
   ```
   kubectl edit logcollector calico-enterprise
   ```

### Configuring syslog archiving (non-Operator)

{% include {{page.version}}/syslog-fluentd.md %}
