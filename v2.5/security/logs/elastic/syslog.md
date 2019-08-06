---
title: Archiving logs to syslog
redirect_from: latest/security/logs/elastic/syslog
---

## Archiving logs to syslog

{{site.prodname}} supports archiving flow and audit logs to syslog.  This provides
a reliable option for maintaining your compliance data long term.  If you wish to use
this feature, follow the instructions below to configure it.

### Configuring syslog archiving

{% include {{page.version}}/syslog-fluentd.md %}
