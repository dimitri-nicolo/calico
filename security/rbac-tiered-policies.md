---
title: Configure RBAC for tiered policies
description: Configure RBAC to control access to policies and tiers. 
canonical_url: /security/rbac-tiered-policies
show_toc: false
---

### Big picture

Configure fine-grained user access controls for tiered policies.

### Value

Self-service is an important part of CI/CD processes for containerization and microservices. {{site.prodname}} provides fine-grained access control (RBAC) for:

- {{site.prodname}} policy and tiers 
- Kubernetes network policy

### Features

This how-to guide uses the following {{site.prodname}} features:

- **{{site.prodname}} API server**

### Concepts

#### Standard Kubernetes RBAC

{{site.prodname}} implements the standard **Kubernetes RBAC Authorization APIs** with `Role` and `ClusterRole` types. The {{site.prodname}} API server integrates with Kubernetes RBAC Authorization APIs as an extension API server. 

#### RBAC for policies and tiers

In {{site.prodname}}, global network policy and network policy resources are associated with a specific tier. Admins can configure access control for these {{site.prodname}} policies using standard Kubernetes `Role` and `ClusterRole` resource types. This makes it easy to manage RBAC for both Kubernetes network policies and {{site.prodname}} tiered network policies. RBAC permissions include managing resources using {{site.prodname}} Manager, and `kubectl`.

#### Fine-grained RBAC for policies and tiers

RBAC permissions can be split by resources ({{site.prodname}} and Kubernetes), and by actions (CRUD). Tiers should be created by administrators. Full CRUD operations on tiers is synonymous with full management of network policy. Full management to network policy and global network policy also requires `GET` permissions to 1) any tier a user can view/manage, and 2) the required access to the tiered policy resources. 

Here are a few examples of how you can fine-tune RBAC for tiers and policies.  

| **User**  | **Permissions**                                              |
| --------- | ------------------------------------------------------------ |
| Admin     | The default **tigera-network-admin** role lets you create, update, delete, get, watch, and list all {{site.prodname}} resources (full control). Examples of limiting Admin access: {::nomarkdown}<ul><li>List tiers only</li><li>List only specific tiers</li></ul>{:/}|
| Non-Admin | The default **tigera-ui-user** role allows users to only list {{site.prodname}} policy and tier resources. Examples of limiting user access: {::nomarkdown}<ul><li>Read-only access to all policy resources across all tiers, but only write access for NetworkPolicies with a specific tier and namespace.</li> <li>Perform any operations on NetworkPolicies and GlobalNetworkPolicies. </li><li>List tiers only.</li> <li>List or modify any policies in any tier.Fully manage only Kubernetes network policies in the default tier, in the default namespace, with read-only access for all other tiers.</li></ul>{:/} |

#### RBAC definitions for Calico Enterprise network policy

To specify per-tier RBAC for the {{site.prodname}} network policy and {{site.prodname}} global network policy, use pseudo resource kinds and names in the `Role` and `ClusterRole` definitions. For example,

```
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1
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
```
Where:
- **resources**: `tier.globalnetworkpolicies` and `tier.networkpolicies`
- **resourceNames**:
  - Blank - any policy of the specified kind across all tiers.
  - `<tiername>.*` - any policy of the specified kind within the named tier.
  - `<tiername.policyname>` - the specific policy of the specified kind. Because the policy name is prefixed with the tier name, this also specifies the tier.

### Before you begin...

**Required**

A **cluster-admin** role with full permissions to create and modify resources.

**Recommended**

A rough idea of your tiered policy workflow, and who should access what. See [Configure tiered policies]({{site.baseurl}}/security/tiered-policy).

### How to

- [Create Admin users, full permissions](#create-admin-users-full-permissions)
- [Create minimum permissions for all non-Admin users](#create-minimum-permissions-for-all-non-admin-users)

> **Note**: ` kubectl auth can-i` cannot be used the check RBAC for tiered policy. 
{: .alert .alert-info}

#### Create Admin users, full permissions

Create an Admin user with full access to the {{site.prodname}} Manager (as well as everything else in the cluster) using the following command. See the Kubernetes documentation to identify users based on your chosen [authentication method](https://kubernetes.io/docs/admin/authentication/){:target="_blank"}, and how to use the [RBAC resources](https://kubernetes.io/docs/reference/access-authn-authz/rbac/){:target="_blank"}.

```
kubectl create clusterrolebinding permissive-binding \
    --clusterrole=cluster-admin \
    --user=<USER>
```
#### Create minimum permissions for all non-Admin users

All users using {{site.prodname}} Manager should be able to create authorizationreviews and authorizationrequests as well as access
license information through the services/proxy https:tigera-api:8080.   

1. Download the [min-ui-user-rbac.yaml manifest]({{ "/getting-started/kubernetes/installation/hosted/cnx/demo-manifests/min-ui-user-rbac.yaml" | absolute_url }}).
   The roles and bindings in this file provide a minimum starting point for setting up RBAC for your users according to your specific security requirements.
   This manifest provides basic RBAC to view some statistical data in the UI but does not provide permissions to 
   view or modify any network policy related configuration.

1. Run the following command to replace <USER> with the name or email of the user you are providing permissions to:

   ```
   sed -i -e 's/<USER>/<name or email>/g' min-ui-user-rbac.yaml
   ```
1. Use the following command to install the bindings:

   ```
   kubectl apply -f min-ui-user-rbac.yaml
   ```

### Tutorial

This tutorial shows how to use RBAC to control access to resources and CRUD actions for a non-Admin user called, **john**.

```
# Users:
- john (non-Admin)
- kubernetes-admin (Admin)
```

RBAC examples include:

- [User can view all policies, and modify policies in the default namespace and tier](#user-can-view-all-policies-and-modify-policies-in-the-default-namespace-and-tier)
- [User cannot read policies in any tier](#user-cannot-read-policies-in-any-tier)
- [User can read policies in the default tier and namespace](#user-can-read-policies-in-the-default-tier-and-namespace)
- [User can read policies only in a specific tier and in the default namespace](#user-can-read-policies-only-in-a-specific-tier-and-in-the-default-namespace)
- [User can view only a specific tier](#user-can-view-only-a-specific-tier)
- [User can read all policies across all tiers and namespaces](#user-can-read-all-policies-across-all-tiers-and-namespaces)
- [User has full control over NetworkPolicy resources in a specific tier and in the default namespace](#user-has-full-control-over-networkpolicy-resources-in-a-specific-tier-and-in-the-default-namespace)

#### User can view all policies, and modify policies in the default namespace and tier

1. Download the [`read-all-crud-default-rbac.yaml` manifest]({{ "/getting-started/kubernetes/installation/hosted/cnx/demo-manifests/read-all-crud-default-rbac.yaml" | absolute_url }}).

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
policies in the default tier and default namespace. This file includes the minimum required `ClusterRole` and `ClusterRoleBinding` definitions for all UI users (see `min-ui-user-rbac.yaml` above).

#### User cannot read policies in any tier

User 'john' is forbidden from reading policies in any tier (default tier, and net-sec tier).

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

> **Note**: The .p' extension (`networkpolicies.p`) is short 
  for "networkpolicies.projectcalico.org" and used to
  differentiate it from the Kubernetes NetworkPolicy resource and
  the underlying CRDs (if using the Kubernetes Datastore Driver).
{: .alert .alert-info}

> **Note**: The label for selecting a tier is `projectcalico.org/tier`.
  When a label selector is not specified, the server defaults the selection to the
  `default` tier. Alternatively, a field selector (`spec.tier`) may be used to select
  a tier.
  ```
  kubectl get networkpolicies.p --field-selector spec.tier=net-sec
  ```
{: .alert .alert-info}

#### User can read policies in the default tier and namespace

In this example, we give user 'john' permission to read the default tier.

```yaml
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: tier-default-tier-reader
rules:
# To access Calico policy in a tier, the user requires get access to that tier, globally.
- apiGroups: ["projectcalico.org"]
  resources: ["tiers"]
  resourceNames: ["default"]
  verbs: ["get"]

---

kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: tier-default-policy-reader
rules:
# This allows get and list of the Calico NetworkPolicy resources in the default tier, namespaced.
- apiGroups: ["projectcalico.org"]
  resources: ["tier.networkpolicies"]
  resourceNames: ["default.*"]
  verbs: ["get", "list"]

---

# Applied globally
kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: read-tier-default-global
subjects:
- kind: User
  name: john
  apiGroup: rbac.authorization.k8s.io
roleRef:
  kind: ClusterRole
  name: tier-default-tier-reader
  apiGroup: rbac.authorization.k8s.io

---

# Applied per-namespace
kind: RoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: read-tier-default-in-this-namespace
subjects:
- kind: User
  name: john
  apiGroup: rbac.authorization.k8s.io
roleRef:
  kind: ClusterRole
  name: tier-default-policy-reader
  apiGroup: rbac.authorization.k8s.io
```

With the above, user john is able to read NetworkPolicy resources in default tier.

```bash
kubectl get networkpolicies.p
```

If no NetworkPolicy resources exist it returns:

```
No resources found.
```
{: .no-select-button}

But John still cannot access tier, **net-sec**.

```bash
kubectl get networkpolicies.p -l projectcalico.org/tier==net-sec
```

This returns:

```
Error from server (Forbidden): networkpolicies.projectcalico.org is forbidden: User "john" cannot list networkpolicies.projectcalico.org in tier "net-sec" and namespace "default" (user cannot get tier)
```
{: .no-select-button}

#### User can read policies only in a specific tier and in the default namespace

Let's assume that the kubernetes-admin gives user 'john' the permission to read tier, **net-sec**.
To provide permission to user 'john' to read policies under 'net-sec' tier, use the following `ClusterRoles`,`ClusterRoleBinding` and `RoleBinding`.

```yaml
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: tier-net-sec-tier-reader
rules:
# To access Calico policy in a tier, the user requires get access to that tier, globally.
- apiGroups: ["projectcalico.org"]
  resources: ["tiers"]
  resourceNames: ["net-sec"]
  verbs: ["get"]

---

kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: tier-net-sec-policy-reader
rules:
# This allows get and list of the Calico NetworkPolicy resources in the net-sec tier, namespaced.
- apiGroups: ["projectcalico.org"]
  resources: ["tier.networkpolicies"]
  resourceNames: ["net-sec.*"]
  verbs: ["get", "list"]

---

# Applied globally
kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: read-tier-net-sec-global
subjects:
- kind: User
  name: john
  apiGroup: rbac.authorization.k8s.io
roleRef:
  kind: ClusterRole
  name: tier-net-sec-tier-reader
  apiGroup: rbac.authorization.k8s.io

---

# Applied per-namespace
kind: RoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: read-tier-net-sec-in-this-namespace
subjects:
- kind: User
  name: john
  apiGroup: rbac.authorization.k8s.io
roleRef:
  kind: ClusterRole
  name: tier-net-sec-policy-reader
  apiGroup: rbac.authorization.k8s.io
```

#### User can view only a specific tier

In this example, the following `ClusterRole` can be used to provide 'get' access to the **net-sec**
tier. This has the effect of making the net-sec tier visible in the {{site.prodname}} Manager. To modify or view policies within the net-sec tier, additional RBAC permissions are required.

```yaml
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: net-sec-tier-visible
rules:
- apiGroups: ["projectcalico.org"]
  resources: ["tiers"]
  verbs: ["get"]
  resourceNames: ["net-sec"]
```

#### User can read all policies across all tiers and namespaces

In this example, the `ClusterRole` is used to provide read access to all policy resource types across all tiers. In this case, there is no need to use both `ClusterRoleBindings` and `RoleBindings`, because this will apply across all namespaces to which the user has access.

```yaml
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1
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

---

# Applied globally
kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: read-all-tier
subjects:
- kind: User
  name: john
  apiGroup: rbac.authorization.k8s.io
roleRef:
  kind: ClusterRole
  name: all-tier-policy-reader
  apiGroup: rbac.authorization.k8s.io
```

#### User has full control over NetworkPolicy resources in a specific tier and in the default namespace

In this example, two `ClusterRole` objects are used to provide full access control of Calico NetworkPolicy
resource types in the **net-sec** tier:
-  The `tiers` resource is bound to a user using a `ClusterRoleBinding`, because it is a global resource.
   This results in the user having the ability to read the contents of the tier across all namespaces.
-  The `networkpolicies` resources are bound to a user using a `RoleBinding`, because they are a namespaced resource.
   You only need this one `ClusterRole` to be defined, but it can be "reused" for users in different namespaces
   using additional `RoleBinding` objects).

> **Note**: The Kubernetes NetworkPolicy resources are bound to the default tier, and so this `ClusterRole`
> does not contain any Kubernetes resource types.
{: .alert .alert-info}

```yaml
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: net-sec-tier-tier-reader
rules:
# To access Calico policy in a tier, the user requires get access to that tier, globally.
- apiGroups: ["projectcalico.org"]
  resources: ["tiers"]
  resourceNames: ["net-sec"]
  verbs: ["get"]

---

kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: net-sec-tier-policy-cruder
rules:
# This allows configuration of the Calico NetworkPolicy resources in the net-sec tier, in this namespace.
- apiGroups: ["projectcalico.org"]
  resources: ["tier.networkpolicies"]
  resourceNames: ["net-sec.*"]
  verbs: ["*"]

---

# Applied globally
kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: read-tier-net-sec-global
subjects:
- kind: User
  name: john
  apiGroup: rbac.authorization.k8s.io
roleRef:
  kind: ClusterRole
  name: net-sec-tier-tier-reader
  apiGroup: rbac.authorization.k8s.io

---

# Applied per-namespace
kind: RoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: crud-tier-net-sec-in-this-namespace
subjects:
- kind: User
  name: john
  apiGroup: rbac.authorization.k8s.io
roleRef:
  kind: ClusterRole
  name: net-sec-tier-policy-cruder
  apiGroup: rbac.authorization.k8s.io
```
