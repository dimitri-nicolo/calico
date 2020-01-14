---
title: Tigera CNX
description: Home
layout: docwithnav
canonical_url: https://docs.tigera.io/v2.3/introduction/
---

{{site.tseeprodname}} adds complementary monitoring and policy management
tools on top of [Project Calico](about-calico). Features included as part of
the {{site.tseeprodname}} release are:


* CNX Manager
  * Web interface for managing tiered security policy
* Policy Query Utility
  * Query policies that are currently being enforced for a given pod
  * Query pods that are affected by a given policy
* Policy Audit Mode
  * Ability to override policy deny actions with log-and-drop or log-and-allow behavior
* Policy Violation Alerting
  * Setup proactive alerting based on connection attempts that violate a policy

For guides on how to set up {{site.tseeprodname}} and a demo of some its basic function, see the following links:


* [Obtaining {{site.tseeprodname}}](../getting-started/)
* [Quickstart for {{site.tseeprodname}} on Kubernetes](../getting-started/kubernetes/)
* [Installing {{site.tseeprodname}} for Kubernetes](../getting-started/kubernetes/installation/hosted/)
* [Demo of {{site.tseeprodname}}](../getting-started/cnx/simple-policy-cnx/)
* [Demo of Tiered Policy using {{site.tseeprodname}}](../getting-started/cnx/tiered-policy-cnx/)
* [Policy Query Utility (calicoq)](../reference/calicoq/)
* [Policy Audit Mode](../reference/cnx/policy-auditing)
* [Policy Violation Alerting](../reference/cnx/policy-violations)
* [RBAC on Tiered Policies](../reference/cnx/rbac-tiered-policies)
* [Configuring user authentication to {{site.tseeprodname}} Manager](../reference/cnx/authentication)
* [Editing policies with {{site.tseeprodname}} Manager](../reference/cnx/policy-editor)
