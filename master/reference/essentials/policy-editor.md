---
title: Policy Editor
---

Tigera CNX Enterprise includes a web-based Policy Editor, as well as a
REST Policy API.

## Detailed description of function and usage

## Policy Authentication

The Policy Editor uses standard Kubernetes-based authentication.
<aside class="warning">
This needs to have all the documentation for how to set up login.
Particularly on the web UI side.
</aside>

## Access model

The authorization model for policies uses the [tier]({{site.baseurl}}/{{page.version}}/reference/calicoctl/resources/tier.md) each policy belongs to as an
additional layer of authorization.  To perform any operation on a policy,
the user must be allowed to "get" the tier that policy is in (or will be
created in).  The operation must still be authorized on the policy in the normal
way in addition to this check.

### Recommendations

1. Avoid giving too many users any form of write access to tiers - it's
tantamount to full administrative control of all network policy.  The ability
to update a tier or create tiers allows that user to place their tier first
and take control before other policy.
1. In general, administrators should be able to perform CRUD operations on tiers,
and users should only be able to get the tiers they need to manage policy in.

### The Default Tier

Typically, all users will be able to "get" the default (last) tier, and therefore
manage policies in it.  Policies created by the orchestrator integration are
created in this tier, such as [Kubernetes NetworkPolicies](https://kubernetes.io/docs/concepts/services-networking/network-policies/) .

### Example

Add this once the user creation workflow is clear (UI side particularly).