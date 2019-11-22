---
title: RBAC on tiered Policies
canonical_url: https://docs.tigera.io/v2.3/reference/cnx/rbac-tiered-policies
---

The {{site.prodname}} API server adds the ability to manage tiered
policies as Kubernetes resources. This allows administrators to manage
access to {{site.prodname}} resources using Kubernetes RBAC
Authorization APIs.

### Policy and tier RBAC

In {{site.prodname}}, `GlobalNetworkPolicy` and `NetworkPolicy` resources
are associated with a specific tier. In addition to the permissions associated
with the policy resources themselves, {{site.prodname}} also gives you the
ability to set permissions based on the tier the policies are in.

To allow a non-admin user to perform any operations on [Global]NetworkPolicies,
you must give them the 'get' permission on the tier(s) they are allowed to
manage policies in.  They must also have permission for the action
(not necessarily 'get') on the policy resources themselves.

For example, to 'create' a network policy in the default tier, a user must have
the 'create' permission for NetworkPolicies (in the namespace they specify on
the resource), and the 'get' permission on the default tier.

Permission to write (create, update, delete, etc) any tiers is approximately
equal to complete control of all network policies: we recommend non-admin users
not be given any write permissions on tiers.

#### The default tier

Policies created by the underlying orchestration integration such as Kubernetes
are placed in the `default` tier.

#### Associating a resource with a tier

For details on creating a [tier]({{site.baseurl}}/{{page.version}}/reference/calicoctl/resources/tier)
resource and adding a Global/NetworkPolicy to that tier, refer to the
[Tiered Policy Demo]({{site.baseurl}}/{{page.version}}/getting-started/cnx/tiered-policy-cnx/).

### Permissions required for {{site.prodname}} UI

All of the RBAC examples below require the user to be specified (by replacing the
text `<USER>`).  Consult the Kubernetes documentation for more information on
[how to identify users based on your chosen authentication method](https://kubernetes.io/docs/admin/authentication/),
and [how to use the RBAC resources](https://kubernetes.io/docs/reference/access-authn-authz/rbac/).

#### Admin users

The quickest way to test the {{site.prodname}} UI is by using an admin user, who
will have full access to the UI (as well as everything else in the cluster).

```
kubectl create clusterrolebinding permissive-binding \
    --clusterrole=cluster-admin \
    --user=<USER>
```

#### Non-admin users

We provide an example manifest to give a non-admin user permission to use the
{{site.prodname}} UI and manage policiies in the default tier and default
namespace.

1. Download the [`min-rbac.yaml` manifest]({{site.baseurl}}/{{page.version}}/getting-started/kubernetes/installation/hosted/cnx/demo-manifests/min-rbac.yaml).

1. Run the following command to replace `<USER>` with the `name or email` of
   the user you are providing permissions to:

   ```
   sed -i -e 's/<USER>/<name or email>/g' min-rbac.yaml
   ```

1. Use the following command to install the bindings:

   ```
   kubectl apply -f min-rbac.yaml
   ```

The roles and bindings in that file should provide a starting point for setting
up RBAC for your users according to your specific security requirements.

### Example fine-grained permissions

```
# Users:
- jane (non-admin)
- kubernetes-admin (admin)
```

User 'jane' is forbidden from reading policies in any tier (default, and
net-sec in this case):

```
# kubectl get networkpolicies.p
Error from server: (Forbidden) Policy operation is associated with tier default. User "jane" cannot list tiers.projectcalico.org at the cluster scope. (get tiers.projectcalico.org)

# kubectl get networkpolicies.p -l projectcalico.org/tier==net-sec
Error from server: (Forbidden) Policy operation is associated with tier net-sec. User "jane" cannot list tiers.projectcalico.org at the cluster scope. (get tiers.projectcalico.org)
```

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

Give user 'jane' permission to read tier 'default':

```
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
  resources: ["networkpolicies"]
  verbs: ["get", "list"]

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

To provide permission to user 'jane' to read policies under 'net-sec' tier,
use the following clusterrole and clusterrolebindings.

kubernetes-admin gives user 'jane' the permission to read tier 'net-sec':
```
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

---

kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1beta1
metadata:
  name: read-tier-net-sec-global
subjects:
- kind: User
  name: jane
  apiGroup: rbac.authorization.k8s.io
roleRef:
  kind: ClusterRole
  name: tier-net-sec-reader
  apiGroup: rbac.authorization.k8s.io
```
