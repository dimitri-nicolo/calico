---
title: Tigera Secure EE Manager Policy Editor
canonical_url: https://docs.tigera.io/v2.3/reference/cnx/policy-editor
---

{{site.prodname}} Manager includes a web client for viewing and editing
[tiered security policy]({{site.baseurl}}/{{page.version}}/reference/calicoctl/resources/tier)

## Policy Authentication

The Policy Editor uses standard Kubernetes-based authentication.

The Web UI uses the underlying Kubernetes permissions for the logged in user
to access all resources; permissions for Web UI users are configured as standard
Kubernetes RBAC roles and role bindings.

The options for authentication are described in detail [here](authentication).

## Access model

The authorization model for policies uses the [tier]({{site.baseurl}}/{{page.version}}/reference/calicoctl/resources/tier) each policy belongs to as an
additional layer of authorization.  To perform any operation on a policy,
the user must be allowed to `GET` the tier that policy is in (or will be
created in).  The operation must still be authorized on the policy in the normal
way in addition to this check.  A detailed description of configuring these
permissions is [here](rbac-tiered-policies).

### UI minimum requirements

All users who are going to use the Web UI should be able to list and watch tiers,
networksets and licenses.
This is accomplished by applying (`kubectl apply`) the follow resources (this example
gives the `webapp-user` group the basic permissions needed to use the Web UI.
```
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: list-cnx-ui
rules:
- apiGroups: ["projectcalico.org"]
  resources: ["tiers", "globalnetworksets", "licences"]
  verbs: ["list", "watch"]
- apiGroups: [""]
  resources: ["services/proxy"]
  verbs: ["get"]
---
kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: webapp-user-list-tiers
subjects:
- kind: Group
  name: webapp-user
  apiGroup: rbac.authorization.k8s.io
roleRef:
  kind: ClusterRole
  name: list-cnx-ui
  apiGroup: rbac.authorization.k8s.io
```

### Tier access

For a non-privileged user to be able to do something with the Web UI, they need to be
granted access to one or more tiers.

Tiers should be created by administrators, and the ability to perform CRUD operations
on tiers is tantamount to full administrative control of all network policy.  The ability
to update a tier or create tiers allows that user to place their tier first
and take control before other policy.

The ability to manage NetworkPolicy and GlobalNetworkPolicy in the Web UI requires GET access to any tier
the user can view/manage, plus the required access to the tiered policy resources. See 
[Configuring {{site.prodname}} RBAC]({{site.url}}/{{page.version}}/reference/cnx/rbac-tiered-policies)
for more details and example configurations.

### The Default Tier

Policies created by the orchestrator integration are
created in this tier, such as [Kubernetes network policy resources](https://kubernetes.io/docs/concepts/services-networking/network-policies/).

Kubernetes resources should be managed directly, rather than modifying the Calico policies created by the 
controller.

## Secure HTTPS

The {{site.prodname}} Manager Web UI uses HTTPS to securely access the {{site.prodname}} Manager and
Kubernetes and {{site.prodname}} API servers over TLS - where 'securely' means that these
communications are encrypted and that the browser can be sure that it is
speaking to those servers.  The web browser should display `Secure` in the
address bar, to indicate this. See [{{site.prodname}} Manager connections](../../usage/encrypt-comms#{{site.prodnamedash}}-manager-connections)
for more information.
