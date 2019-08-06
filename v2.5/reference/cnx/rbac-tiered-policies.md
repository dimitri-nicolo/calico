---
title: Configuring Tigera Secure EE RBAC for tiered policies
redirect_from: latest/reference/cnx/rbac-tiered-policies
canonical_url: https://docs.tigera.io/v2.3/reference/cnx/rbac-tiered-policies
---

The {{site.prodname}} API server adds the ability to manage tiered
policies as Kubernetes resources. This allows administrators to manage
access to {{site.prodname}} resources using Kubernetes RBAC
Authorization APIs.

If you are upgrading from a pre-v2.3 release of {{site.prodname}}, or you want to maintain the
pre-v2.3 RBAC behavior, [upgrade to 2.3](/v2.3/getting-started/kubernetes/upgrade/upgrade-tsee#upgrading-pre23),
following the instructions for handling RBAC. Then, continue to
[upgrade to this release](/{{page.version}}/maintenance/kubernetes-upgrade-tsee).

For RBAC related to elasticsearch, please refer to [Configuring Tigera Secure EE RBAC for Elasticsearch](/{{page.version}}/reference/cnx/rbac-elasticsearch).

### Policy and tier RBAC

In {{site.prodname}}, `GlobalNetworkPolicy` and `NetworkPolicy` resources
are associated with a specific tier. Access control for these resources can
be configured using standard Kubernetes `Role` and `ClusterRole` resource types, and may be
configured differently for each tier.

For example, it is possible to allow a user to have read-only access to all policy
resources across all tiers, but to only allow write access for NetworkPolicies within
a specific tier and namespace.

To allow a non-admin user to perform *any* operations on either Calico GlobalNetworkPolicy and
NetworkPolicy resources, you must give them the 'get' permission on the tier(s) they are allowed to
manage policies in. They must also have permission for the action (not necessarily 'get') on the
appropriate policy resources within the required tiers.

For all users of the {{site.prodname}} UI, 'watch' and 'list' permissions are required
for tiers. The {{site.prodname}} UI will only display tiers for which the user has 'get'
permission for.

For example, to 'create' a network policy in the default tier, a user must have
'create' permission for NetworkPolicies in the default tier and in the namespace they specify on
the resource, and 'get' permission on the default tier.

Permission to write (create, update, delete, etc) any tiers is approximately
equal to complete control of all network policies: we recommend non-admin users
not be given any write permissions on tiers.

#### The default tier

Policies created by the underlying orchestration integration such as Kubernetes
are placed in the `default` tier.

{{site.prodname}} `NetworkPolicy` resources that are derived from the Kubernetes `NetworkPolicy` resources
will have a prefix `knp.` added to the name. These are not directly configurable through `kubectl`
although you may use it to view the derived resources. Modification of these resources is handled through
the actual Kubernetes resources, and RBAC configuration for managing these resources is specified using the
actual Kubernetes resource types.

To allow a user to modify the Kubernetes `NetworkPolicy` resources through the UI the user requires both
`get` permissions to default tier as well as appropriate permissions for the Kubernetes `NetworkPolicy` resource
types.

#### Calico policy resource kind and names in RBAC definitions

The per-tier RBAC for the Calico policy resources is specified using pseudo resource kinds and names in the
`Role` and `ClusterRole` definitions.

-  For the `resources` field use the kinds `tier.globalnetworkpolicies` and `tier.networkpolicies` for the
   Calico resources.
-  For the `resourceNames` field use the format:
  -  Leave blank to mean any policy of the specified kind across all tiers
  -  `<tiername>.*` to mean any policy of the specified kind within the named tier
  -  `<policyname>` to mean a specific policy of the specified kind (note that since the policy name is prefixed
     with the tier name then this also specifies the tier).

Refer to the [Example fine-grained permissions](#examples) section below for a worked example. Also see the
[Non-admin users](#non-admin-users) section below for an example manifest that provides specific access control for
a user using the UI.

> **Note**: This is different from the pre-v2.3 RBAC configuration which used the real resource Calico kinds of
> `networkpolicies` and `globalnetworkpolicies`, and did not allow the wildcard format (`<tiername>.*`) for the
> policy names. The wildcard format is only supported for the pseudo-resource types and is interpreted by the
> {{site.prodname}} Aggregated API Server. It is the wildcard name format that allows per-tier granularity of the
> policy RBAC configuration.
{: .alert .alert-info}

### Associating a resource with a tier

For details on creating a [tier]({{site.url}}/{{page.version}}/reference/resources/tier)
resource and adding a Global/NetworkPolicy to that tier, refer to the
[Tiered Policy Demo]({{site.url}}/{{page.version}}/security/tiered-policy).

### Permissions required for {{site.prodname}} UI

All of the RBAC examples below require the user to be specified (by replacing the
text `<USER>`).  Consult the Kubernetes documentation for more information on
[how to identify users based on your chosen authentication method](https://kubernetes.io/docs/admin/authentication/),
and [how to use the RBAC resources](https://kubernetes.io/docs/admin/authorization/rbac/).

#### Admin users

The quickest way to test the {{site.prodname}} UI is by using an admin user, who
will have full access to the UI (as well as everything else in the cluster).

```bash
kubectl create clusterrolebinding permissive-binding \
    --clusterrole=cluster-admin \
    --user=<USER>
```

#### Non-admin users

All users of the UI require a minimum set of permissions in addition to the specific set of permissions
to access the various policy resources and tiers.

We provide two sample manifests for users of the UI. The first is the minimum set of permissions required
for *all* users of the UI. These minimum permissions will not allow the user to view or modify any policies.
The second manifest gives a non-admin user permission to fully manage policies in the default tier and default
namespace, and to provide read-only access for all other tiers.

##### Minimum permissions for all UI users

1. Download the [`min-ui-user-rbac.yaml` manifest]({{site.url}}/{{page.version}}/getting-started/kubernetes/installation/hosted/cnx/demo-manifests/min-ui-user-rbac.yaml).

1. Run the following command to replace `<USER>` with the `name or email` of
   the user you are providing permissions to:

   ```
   sed -i -e 's/<USER>/<name or email>/g' min-ui-user-rbac.yaml
   ```

1. Use the following command to install the bindings:

   ```
   kubectl apply -f min-ui-user-rbac.yaml
   ```

The roles and bindings in this file provide a minimum starting point for setting up RBAC for your users according to your
specific security requirements.

##### UI user can view all policies and can modify policies in the default namespace and tier

1. Download the [`read-all-crud-default-rbac.yaml` manifest]({{site.url}}/{{page.version}}/getting-started/kubernetes/installation/hosted/cnx/demo-manifests/read-all-crud-default-rbac.yaml).

1. Run the following command to replace `<USER>` with the `name or email` of
   the user you are providing permissions to:

   ```
   sed -i -e 's/<USER>/<name or email>/g' read-all-crud-default-rbac.yaml
   ```

1. Use the following command to install the bindings:

   ```
   kubectl apply -f read-all-crud-default-rbac.yaml
   ```

The roles and bindings in this file provide the permissions to read all policies across all tiers and to fully manage
policies in the default tier and default namespace. This file includes the minimum required `ClusterRole` and `ClusterRoleBinding`
definitions for all UI users (see `min-ui-user-rbac.yaml` above).

### <a name="examples"></a>Example fine-grained permissions

```
# Users:
- john (non-admin)
- kubernetes-admin (admin)
```

User 'john' is forbidden from reading policies in any tier (default, and
net-sec in this case).

When John issues the following command:

```
kubectl get networkpolicies.p
```

It returns:

```
Error from server (Forbidden): networkpolicies.projectcalico.org is forbidden: User "john" cannot list networkpolicies.projectcalico.org in tier "default" and namespace "default" (user cannot get tier)
```
{: .no-select-button}

Similarly, when John issues this command:

```
kubectl get networkpolicies.p -l projectcalico.org/tier==net-sec
```

It returns:

```
Error from server (Forbidden): networkpolicies.projectcalico.org is forbidden: User "john" cannot list networkpolicies.projectcalico.org in tier "net-sec" and namespace "default" (user cannot get tier)
```
{: .no-select-button}

> **Note**: The appended '.p' with the networkpolicies resource in the kubectl
  command. That is short for "networkpolicies.projectcalico.org" and is needed
  to differentiate from the Kubernetes namesake NetworkPolicy resource and
  (if using the Kubernetes Datastore Driver) the underlying CRDs.
{: .alert .alert-info}

> **Note**: Currently, the tier collection on a Policy resource through the
  kubectl client (pre 1.9) of the APIs is implemented using labels because
  kubectl lacks field selector support. The label used for tier collection
  is "projectcalico.org/tier". When a label selection is not specified, the
  server defaults the collection to the `default` tier. Field selection based
  policy collection is enabled at API level. spec.tier is the field to select
  on for the purpose.
{: .alert .alert-info}

Give user 'john' permission to read tier 'default':

```yaml
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1beta1
metadata:
  name: tier-default-reader
rules:
- apiGroups: ["projectcalico.org"]
  resources: ["tiers"]
  resourceNames: ["default"]
  verbs: ["get"]
- apiGroups: ["projectcalico.org"]
  resources: ["tier.networkpolicies"]
  resourceNames: ["default.*"]
  verbs: ["get", "list"]

---

kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1beta1
metadata:
  name: read-tier-default-global
subjects:
- kind: User
  name: john
  apiGroup: rbac.authorization.k8s.io
roleRef:
  kind: ClusterRole
  name: tier-default-reader
  apiGroup: rbac.authorization.k8s.io
```

With the above user john is able to read NetworkPolicy resources in default tier

```bash
kubectl get networkpolicies.p
```

If no NetworkPolicy resources exist it returns:

```
No resources found.
```
{: .no-select-button}

But John still cannot access net-sec.

```bash
kubectl get networkpolicies.p -l projectcalico.org/tier==net-sec
```

This returns:

```
Error from server (Forbidden): networkpolicies.projectcalico.org is forbidden: User "john" cannot list networkpolicies.projectcalico.org in tier "net-sec" and namespace "default" (user cannot get tier)
```
{: .no-select-button}

To provide permission to user 'john' to read policies under 'net-sec' tier,
use the following clusterrole and clusterrolebindings.

kubernetes-admin gives user 'john' the permission to read tier 'net-sec':

```yaml
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1beta1
metadata:
  # "namespace" omitted since ClusterRoles are not namespaced
  name: tier-net-sec-reader
rules:
- apiGroups: ["projectcalico.org"]
  resources: ["tiers"]
  resourceNames: ["net-sec"]
  verbs: ["get"]
- apiGroups: ["projectcalico.org"]
  resources: ["tier.networkpolicies"]
  resourceNames: ["net-sec.*"]
  verbs: ["get", "list"]

---

kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1beta1
metadata:
  name: read-tier-net-sec-global
subjects:
- kind: User
  name: john
  apiGroup: rbac.authorization.k8s.io
roleRef:
  kind: ClusterRole
  name: tier-net-sec-reader
  apiGroup: rbac.authorization.k8s.io
```

#### Additional examples

##### Displaying only the net-sec tier

The following ClusterRole can be used to provide 'get' access to the net-sec
tier. This has the effect of making the net-sec tier visible in the
{{site.prodname}} UI. Additional RBAC permissions are required in order to modify
or view policies within the net-sec tier.

```yaml
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1beta1
metadata:
  name: net-sec-tier-visible
rules:
- apiGroups: ["projectcalico.org"]
  resources: ["tiers"]
  verbs: ["get"]
  resourceNames: ["net-sec"]
```

##### Read all policies across all tiers

The following ClusterRole can be used to provide read access to all policy resource types across all tiers.

```yaml
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1beta1
metadata:
  name: all-tier-policy-reader
rules:
# To access Calico policy in a tier, the user requires get access to that tier. This provides get
# access to all tiers.
- apiGroups: ["projectcalico.org"]
  resources: ["tiers"]
  verbs: ["get"]
# This allows read access of the kubernetes NetworkPolicy resources (these are always in the default tier).
- apiGroups: ["networking.k8s.io", "extensions"]
  resources: ["networkpolicies"]
  verbs: ["get","watch","list"]
# This allows read access of the Calico NetworkPolicy and GlobalNetworkPolicy resources in all tiers.
- apiGroups: ["projectcalico.org"]
  resources: ["tier.networkpolicies","tier.globalnetworkpolicies"]
  verbs: ["get","watch","list"]
```

##### Full control over NetworkPolicy resources in the default tier

The following ClusterRole can be used to provide full access control of Calico and Kubernetes NetworkPolicy
resource types in the default tier.
-  If this is bound to a user using a ClusterRoleBinding, then the user will have full access of these
   policies across all namespaces.
-  If this is bound to a user using a RoleBinding, then the user will have full access of these
   policies within a specific namespace.  (This is useful because you only need this one ClusterRole to be
   defined, but it can be "reused" for users in different namespaces using a RoleBinding).

```yaml
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1beta1
metadata:
  name: default-tier-policy-cruder
rules:
# To access Calico policy in a tier, the user requires get access to that tier.
- apiGroups: ["projectcalico.org"]
  resources: ["tiers"]
  resourceNames: ["default"]
  verbs: ["get"]
# This allows configuration of the kubernetes NetworkPolicy resources (these are always in the default tier).
# This is required if the user needs to be able to modify the Calico-rendered Kubernetes resources in the UI.
- apiGroups: ["networking.k8s.io", "extensions"]
  resources: ["networkpolicies"]
  verbs: ["create","update","delete","patch","get","watch","list"]
# This allows configuration of the Calico NetworkPolicy resources in the default tier.
- apiGroups: ["projectcalico.org"]
  resources: ["tier.networkpolicies"]
  resourceNames: ["default.*"]
  verbs: ["*"]
```

##### Full control over NetworkPolicy resources in the net-sec tier

The following ClusterRole can be used to provide full access control of Calico NetworkPolicy
resource types in the net-sec tier.
-  If this is bound to a user using a ClusterRoleBinding, then the user will have full access of these
   policies across all namespaces.
-  If this is bound to a user using a RoleBinding, then the user will have full access of these
   policies within a specific namespace.  (This is useful because you only need this one ClusterRole to be
   defined, but it can be "reused" for users in different namespaces using a RoleBinding).

> **Note**: The Kubernetes NetworkPolicy resources are bound to the default tier, and so this ClusterRole
> does not contain any Kubernetes resource types.
{: .alert .alert-info}

```yaml
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1beta1
metadata:
  name: net-sec-tier-policy-cruder
rules:
# To access Calico policy in a tier, the user requires get access to that tier.
- apiGroups: ["projectcalico.org"]
  resources: ["tiers"]
  resourceNames: ["net-sec"]
  verbs: ["get"]
# This allows configuration of the Calico NetworkPolicy resources in the net-sec tier.
- apiGroups: ["projectcalico.org"]
  resources: ["tier.networkpolicies"]
  resourceNames: ["net-sec.*"]
  verbs: ["*"]
```

