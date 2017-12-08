---
title: Tigera CNX
redirect_from: latest/introduction/index
description: Home
layout: docwithnav
---

{{site.prodname}} adds complementary monitoring and policy management
tools on top of [Project Calico](about-calico). Features included as part of
the {{site.prodname}} release are:


* CNX Manager
  * Web interface for managing tiered security policy
* Policy Query Utility
  * Query policies that are currently being enforced for a given pod
  * Query pods that are affected by a given policy
* Policy Audit Mode
  * Ability to override policy deny actions with log-and-drop or log-and-allow behavior
* Policy Violation Alerting
  * Setup proactive alerting based on connection attempts that violate a policy

For guides on how to set up {{site.prodname}} and a demo of some its basic function, see the following links:


* [Obtaining {{site.prodname}}](../getting-started/)
* [Quickstart for {{site.prodname}} on Kubernetes](../getting-started/kubernetes/)
* [Installing {{site.prodname}} for Kubernetes](../getting-started/kubernetes/installation/hosted/)
* [Demo of {{site.prodname}}](../getting-started/essentials/simple-policy-essentials)
* [Demo of Tiered Policy using {{site.prodname}}](../getting-started/essentials/tiered-policy-essentials)
* [Policy Query Utility (calicoq)](../reference/calicoq/)
* [Policy Audit Mode](../reference/essentials/policy-auditing)
* [Policy Violation Alerting](../reference/essentials/policy-violations)
* [RBAC on Tiered Policies](../reference/essentials/rbac-tiered-policies)
* [Configuring user authentication to {{site.prodname}} Manager](../reference/essentials/authentication)
* [Editing policies with {{site.prodname}} Manager](../reference/essentials/policy-editor)
