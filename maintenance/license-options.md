---
title: Options to track Calico Enterprise license expiration
description: Learn the options for managing your Calico Enterprise license.
---

### Big picture

Review options for tracking {{site.prodname}} license expiration.

### Concepts

We highly recommend using the [license agent using Prometheus]({{site.baseurl}}/maintenance/monitor/license-agent) to get alerts on your {{site.prodname}} license expiration day to avoid disruption to services. Regardless if you using the alerting feature, here are some things you should know.

#### FAQ

The following questions and answers apply to both POC and non-POC {{site.prodname}} licenses.

**How long does it take to get a new {{site.prodname}} license?**

  After receiving a Sales purchase order, approximately 1-2 days.

**What happens to nodes during the license grace period?**

  All {{site.prodname}} features will work without interruption.

**What happens to nodes after the license grace period?**
- Initially, users can access the {{site.prodname}} Manager, and a message is displayed to change the license. Users will not be able to access {{site.prodname}} Manager after two days.
- Although components appear to function when the license expires and grace period is over, {{site.prodname}} tiers and policies no longer work, so any changes using {{site.prodname}} Manager do not take affect. Only policies in the default Kubernetes tier are applied.

**Are license metrics are available through {{site.prodname}} Manager?**

  No. Currently, the only interface is Prometheus. 

**What happens if I add nodes beyond what I'm licensed for?**
- New nodes that you add past your limit, are still added
- All {{site.prodname}} features still work

### Above and beyond

- [Monitor {{site.prodname}} license metrics]({{site.baseurl}}/maintenance/monitor/license-agent)
