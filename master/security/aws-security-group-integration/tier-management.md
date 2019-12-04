---
title: Manage the tiers associated with security group integration
canonical_url: https://docs.tigera.io/v2.3/usage/aws-security-group-integration/tier-management
---


{{site.prodname}} installs the `default` and `allow-cnx` tiers as part of the
base set of tiers. Enabling the
[AWS security group integration](/{{page.version}}/reference/other-install-methods/kubernetes/installation/aws-sg-integration)
introduces three additional tiers: `sg-remote`, `sg-local`, and `metadata`.

| Tier        | Order  | Installed with ...         |
|-------------|--------|----------------------------|
| `default`   | n/a    | {{site.prodname}}          |
| `allow-cnx` | 100    | {{site.prodname}}          |
| `sg-remote` | 105    | AWS SG integration feature |
| `sg-local`  | 106    | AWS SG integration feature |
| `metadata`  | 107    | AWS SG integration feature |

The tiers above and the network policies in them should not be
modified since they are essential for proper functioning of {{site.prodname}}.
In addition, we recommend installing customer-specific tiers _after_ the above
tiers, i.e. with order > 107.

In order to avoid accidentally modifying the above tiers and associated
network policies, the best practice is to restrict non-admin users' ability to
view and modify the above tiers.
Although [RBAC for tiered policies](/{{page.version}}/reference/cnx/rbac-tiered-policies)
does not allow you to hide specific tiers from non-admin users, it is possible
to display only a subset of tiers in the {{site.prodname}} UI.

For an example of how to display only a subset of tiers in the {{site.prodname}} UI, see
[displaying only the net-sec tier](/{{page.version}}/reference/cnx/rbac-tiered-policies#displaying-only-the-net-sec-tier).

For an example of how to display tiers and associated network policies, but
disable write access to those tiers, see
[RBAC example fine-grained permissions](/{{page.version}}/reference/cnx/rbac-tiered-policies#example-fine-grained-permissions).

> **Note**: Although admin users have permission to modify the above tiers and associated
> network policies, it is best practice to not do so.
{: .alert .alert-info}
