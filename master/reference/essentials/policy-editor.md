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
Particularly on the web UI side - how do you actually figure out the
name / group for an OIDC user etc.
</aside>

The Web UI uses the underlying Kubernetes permissions for the logged in user
to access all resources; permissions for Web UI users are configured as standard
Kubernetes RBAC roles and role bindings.

## Access model

The authorization model for policies uses the [tier]({{site.baseurl}}/{{page.version}}/reference/calicoctl/resources/tier.md) each policy belongs to as an
additional layer of authorization.  To perform any operation on a policy,
the user must be allowed to "get" the tier that policy is in (or will be
created in).  The operation must still be authorized on the policy in the normal
way in addition to this check.

### UI minimum requirements

All users who are going to use the Web UI need to be able to list and watch tiers.
This is accomplished by applying (`kubectl apply`) the follow resources (this example
gives the `webapp-user` group the basic permissions needed to use the Web UI.
```
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: list-tiers
rules:
- apiGroups: ["projectcalico.org"]
  resources: ["tiers"]
  verbs: ["list", "watch"]
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
  name: list-tiers
  apiGroup: rbac.authorization.k8s.io
```

### Tier access

For a non-privileged user to be able to do something with the Web UI, they need to be
granted access to one or more tiers.

Tiers should be created by adminitrators, and the ability to perform CRUD operations
on tiers is tantamount to full administrative control of all network policy.  The ability
to update a tier or create tiers allows that user to place their tier first
and take control before other policy.

The following roles and role bindings allow the `netsec` group to manage the contents of
the `netsec` tier.

```
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: edit-netsec
rules:
- apiGroups: ["projectcalico.org"]
  resources: ["tiers"]
  resourceNames: ["netsec"]
  verbs: ["get", "list", "watch"]
---
kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: netsec-edit-netsec
subjects:
- kind: Group
  name: netsec
  apiGroup: rbac.authorization.k8s.io
roleRef:
  kind: ClusterRole
  name: edit-netsec
  apiGroup: rbac.authorization.k8s.io
```

Note that the NetworkPolicies and GlobalNetworkPolicies in each tier have RBAC
applied in the usual Kubernetes way in addition to the "get" tier requirement.
That is, to edit a policy, the user needs to be able to edit that policy, _and_
get the tier that it belongs to.

### The Default Tier

Typically, all users should be able to "get" the default (last) tier, and therefore
manage policies in it.  Policies created by the orchestrator integration are
created in this tier, such as [Kubernetes NetworkPolicies](https://kubernetes.io/docs/concepts/services-networking/network-policies/).

Note that policies in the default tier are generally created by the
orchestrator integration, and it is best to edit the original Kubernetes
NetworkPolicies instead.
