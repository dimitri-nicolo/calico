---
title: RBAC on tiered Policies
redirect_from: latest/reference/essentials/rbac-tiered-policies
---

{{site.prodname}} with the CNX Kubernetes extension API Server adds the ability to manage tiered policies as Kubernetes resources. And with that comes the ability to use Kubernetes RBAC Authorization APIs, under apiGroup rbac.authorization.k8s.io, with CNX resources.

### Overview
- Policy resources: NetworkPolicy and GlobalNetworkPolicy, along with Tier resource will be exposed via CNX Kubernetes extension API Server.
- You can create a tier using the [Tier]({{site.baseurl}}/{{page.version}}/reference/calicoctl/resources/tier) resource and add policies to the tier using the tier field in the spec section of a global/networkpolicy resource. For a tutorial on tiers, refer to [Tiered Policy Demo]({{site.baseurl}}/{{page.version}}/getting-started/essentials/tiered-policy-essentials).
- A user’s CRUD-ability (also watch, list etc, throughout) on a global/networkpolicy resource will be subject to the RBAC rules as set by the administrator on the global/networkpolicy resource for the user, as well as on the RBAC rules as set by the administrator on the tier resource associated with that given global/networkpolicy resource for the same user.
- The associated RBAC role on tier for the user will only be evaluated for Read/"get" permission. I.e. as long as a user can "get" the tier, the user is free to perform any operations on policies in that tier subject to their roles for those policies.
- See the [authentication guide](authentication) for how usernames and groups will be presented.

### Operational considerations
- Tier permissions beyond read, like create, update and delete, are tantamount to admin - even if only granted for a single tier (e.g. change its order so it applies first).
- Every user is expected to have Read permission on the default tier to start with. Any Policy CRUD by default is expected to happen against the ‘default’ tier until otherwise specified.
- Admin user will have the ability to set tier Read authorization on an individual tier resource level granularity.
- A non-admin user could be given the permission to "list" tier resource at cluster-scope, so they need not be explicitly aware of the individual tiers resource they will have Read/"get" access to, but a "get" permission on an instance of a tier resource would be needed for the non-admin user to be able to CRUD Global/NetworkPolicies under the associated tier.

### Demo
```
# Users: 
- jane (non-admin)
- kubernetes-admin (admin)
```
User ‘jane’ doesn’t have permissions to CRUD either any of the networkpolicy or tier resources. 

User ‘kubernetes-admin’ gives permission to ‘jane’ to read NetworkPolicies in Default Namespace. (can be extended with the verbs “create”, “update”, “delete”)

```
# cat policy-reader-auth.yaml
kind: Role
apiVersion: rbac.authorization.k8s.io/v1beta1
metadata:
  namespace: default
  name: policy-reader
rules:
- apiGroups: ["projectcalico.org"] 
  resources: ["networkpolicies"]
  verbs: ["get", "watch", "list"]
---
# This role binding allows "jane" to read policies in the "default" namespace.
kind: RoleBinding
apiVersion: rbac.authorization.k8s.io/v1beta1
metadata:
  name: read-policies
  namespace: default
subjects:
- kind: User
  name: jane
  apiGroup: rbac.authorization.k8s.io
roleRef:
  kind: Role
  name: policy-reader
  apiGroup: rbac.authorization.k8s.io
```

User jane is forbidden from reading policies in any present tier (default, and net-sec in this case):
```
# kubectl get networkpolicies.p
Error from server: (Forbidden) Policy operation is associated with tier default. User "jane" cannot list tiers.projectcalico.org at the cluster scope. (get tiers.projectcalico.org)

# kubectl get networkpolicies.p -l projectcalico.org/tier==net-sec
Error from server: (Forbidden) Policy operation is associated with tier net-sec. User "jane" cannot list tiers.projectcalico.org at the cluster scope. (get tiers.projectcalico.org)
```

> **Note**: The appended '.p' with the networkpolicies resource in the kubectl command. That is short for "networkpolicies.projectcalico.org" and is needed to differentiate from the Kubernetes namesake NetworkPolicy resource. It won't be necessary when dealing with GlobalNetworkPolicy resource.
{: .alert .alert-info}

> **Note**: Currently, the tier collection on a Policy resource through the kubectl client of the APIs is being done using labels, since kubectl lacks field selector support (at least pre 1.9 release). The label being used for tier collection is "projectcalico.org/tier". Label selector support for policies can be deprecated at will without requiring a store update. Field Selection based Policy collection is enabled at API level. spec.tier is the field to select on for the purpose.
{: .alert .alert-info}

kubernetes-admin gives User jane the permission to read tier ‘default’:
```
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1beta1
metadata:
  # "namespace" omitted since ClusterRoles are not namespaced
  name: tier-default-reader
rules:
- apiGroups: ["projectcalico.org"]
  resources: ["tiers"]
  resourceNames: ["default"]
  verbs: ["get", "watch", "list"]
---

kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1beta1
metadata:
  name: read-tier-default-global
subjects:
- kind: User
  name: jane
  apiGroup: rbac.authorization.k8s.io
roleRef:
  kind: ClusterRole
  name: tier-default-reader
  apiGroup: rbac.authorization.k8s.io
```

With the above user jane is able to read policies in default tier
```
# kubectl get networkpolicies.p
No resources found.
```
But still not net-sec
```
# kubectl get networkpolicies.p -l projectcalico.org/tier==net-sec
Error from server: (Forbidden) Policy operation is associated with tier net-sec. User "jane" cannot list tiers.projectcalico.org at the cluster scope. (get tiers.projectcalico.org)
```
