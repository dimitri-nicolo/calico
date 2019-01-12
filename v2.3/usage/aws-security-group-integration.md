---
title: AWS security group integration
canonical_url: https://docs.tigera.io/master/usage/aws-security-group-integration
---

## Managing the tiers associated with security group integration

{{site.prodname}} installs the `default` and `allow-cnx` tiers as part of the
base set of tiers. Enabling the
[AWS security group integration]({{site.baseurl}}/{{page.version}}/getting-started/kubernetes/installation/aws-sg-integration)
introduces three additional tiers: `sg-remote`, `sg-local`, and `metadata`.

| Tier        | Order  | Installed with ...         |
|-------------|--------|----------------------------|
| `default`   | n/a    | {{site.prodname}}          |
| `allow-cnx` | 100    | {{site.prodname}}          |
| `sg-remote` | 105    | AWS SG integration feature |
| `sg-local`  | 106    | AWS SG integration feature |
| `metadata`  | 107    | AWS SG integration feature |

None of the above tiers, or the network policies in those tiers, should be
modified since they're essential for proper functioning of {{site.prodname}}.
In addition, we recommend installing customer-specific tiers _after_ the above
tiers, i.e. with order > 107.

In order to avoid accidentally modifying the above tiers and associated
network policies, the best practice is to restrict non-admin users' ability to
view and modify the above tiers.
Although [RBAC for tiered policies]({{site.baseurl}}/{{page.version}}/reference/cnx/rbac-tiered-policies) 
doesn't allow you to hide specific tiers from non-admin users, it is possible
to display only a subset of tiers in the {{site.prodname}} UI. 

For an example of how to display only a subset of tiers in the {{site.prodname}} UI, see
[displaying only the net-sec tier]({{site.baseurl}}/{{page.version}}/reference/cnx/rbac-tiered-policies#displaying-only-the-net-sec-tier).

For an example of how to display tiers and associated network policies, but
disable write access to those tiers, see
[RBAC example fine-grained permissions]({{site.baseurl}}/{{page.version}}/reference/cnx/rbac-tiered-policies#example-fine-grained-permissions).

{: .alert .alert-info}
Although admin users have permission to modify the above tiers and associated
network policies, it is best practice to not do so.
