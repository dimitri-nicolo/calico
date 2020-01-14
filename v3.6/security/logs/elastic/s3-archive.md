---
title: Archiving logs to S3
canonical_url: https://docs.tigera.io/v2.3/usage/logs/elastic/s3-archive
---

## Archiving logs to S3

{{site.tseeprodname}} supports archiving flow and audit logs to Amazon S3.  This provides
a reliable option for maintaining your compliance data long term.  If you wish to use
this feature, follow the instructions below to configure it.

### Configuring S3 archiving

{% include {{page.version}}/s3_fluentd.md %}