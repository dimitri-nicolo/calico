---
title: Tier Resource (tier)
no_canonical: true
---

A tier resource (`Tier`) represents an ordered collection of [NetworkPolicies]({{site.baseurl}}/{{page.version}}/reference/calicoctl/resources/networkpolicy)
and/or [GlobalNetworkPolicies]({{site.baseurl}}/{{page.version}}/reference/calicoctl/resources/globalnetworkpolicy).
Tiers are used to divide these policies into groups of different priorities.  These policies
are ordered within a Tier: the additional hierarchy of Tiers provides more flexibility
because the `pass` `action` in a Rule jumps to the next Tier.  Some example use cases for this are.
- Allowing privileged users to define security policy that takes precedence over other users.
- Translating hierarchies of physical firewalls directly into Calico Policy.

For `calicoctl` commands that specify a resource type on the CLI, the following
aliases are supported (all case insensitive): `tier`, `tiers`.

### How Policy Is Evaluated

When a new connection is processed by Calico, each Tier that contains a Policy that applies to the endpoint processes the packet.
Tiers are sorted first by their `order` (smallest number first), and as a tie-breaker by `name` (alphabetically).

Policies in each Tier are then processed in order (again, by `order` then by `name`).
- If a Policy in the Tier `allow`s or `deny`s the packet, then evaluation is done: the packet is handled accordingly.
- If a Policy in the Tier `pass`es the packet, the next Tier containing a Policy that applies to the endpoint processes the packet.

If the Tier applies to the endpoint, but takes no action on the packet the packet is dropped.

If the last Tier applying to the endpoint `pass`es the packet, that endpoint's [Profiles]({{site.baseurl}}/{{page.version}}/reference/calicoctl/resources/profile) are evaluated.

### Sample YAML

```yaml
apiVersion: projectcalico.org/v2
kind: Tier
metadata:
  name: internal-access
spec:
  order: 100

```

### Definition

#### Metadata

| Field | Description  | Accepted Values   | Schema |
|-------|--------------|-------------------|--------|
| name | The name of the tier.   |         | string |

#### Spec

| Field      | Description                                                                                                                                                         | Accepted Values | Schema                | Default |
|------------+---------------------------------------------------------------------------------------------------------------------------------------------------------------------+-----------------+-----------------------+---------|
| order      | (Optional) Indicates priority of this Tier, with lower order taking precedence.  No value indicates highest order (lowest precedence)                             |                 | float                 |         |

All Policies created by Calico orchestrator integrations are created in the default (last) Tier.
