---
title: Tigera CNX Manager Policy Editor
---

Tigera CNX Manager includes a web client for viewing and editing
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

Tiers should be created by administrators, and the ability to perform CRUD operations
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

Note that the `NetworkPolicy` and `GlobalNetworkPolicy` resources in each tier have RBAC
applied in the usual Kubernetes way in addition to the `GET` tier requirement.
That is, to edit a policy, the user needs to be able to edit that policy, _and_
`GET` the tier that it belongs to.

### The Default Tier

Typically, all users should be able to `GET` the default (last) tier, and therefore
manage policies in it.  Policies created by the orchestrator integration are
created in this tier, such as [Kubernetes network policy resources](https://kubernetes.io/docs/concepts/services-networking/network-policies/).

Note that policies in the default tier are generally created by the
orchestrator integration, and it is best to edit the original Kubernetes
`NetworkPolicy` resources instead.

## Secure HTTPS

The CNX Manager Web UI uses HTTPS to securely access the CNX Manager and
Kubernetes and CNX API servers over TLS - where 'securely' means that these
communications are encrypted and that the browser can be sure that it is
speaking to those servers.  The web browser should display `Secure` in the
address bar, to indicate this.

If the web browser does not display `Secure`, the likely reasons - and how to
fix them - are as follows.

- Your browser does not know about (and trust) the certificate authorities that
  issued the certificates that the Kubernetes API server and CNX Manager are
  using - typically because your installation process has generated and used
  its own certificate authority (CA) instead of using one of the recognized
  ones.  If so, you can fix that, on a per-browser basis, by importing the CA
  certificates into that browser.  In Google Chrome this can be done under
  Settings, Advanced, Privacy and security, Manage certificates, Authorities,
  Import.

- The certificates have a Common Name or Subject Alternative Name that is
  different from the host name or IP address - i.e. what it says in the browser
  address bar - that you are using to access the CNX Manager server.  In Google
  Chrome you can check the CNX Manager certificate with Developer Tools
  (Ctrl+Shift+I), Security.  In this case the fix is to reissue the certificate
  with correct Common Name and Subject Alternative Name, update the
  `cnx-manager-tls` secret to include the new certificate, and kill the CNX
  Manager pod so that it is restarted.
